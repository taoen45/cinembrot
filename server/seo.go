package server

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cinembrot/database"
	"cinembrot/model"
	"cinembrot/scraper"
)

// Sitemap XML Structs (Sitemaps Protocol 0.9 + Google Image Extension)
type SitemapImage struct {
	XMLName xml.Name `xml:"image:image"`
	Loc     string   `xml:"image:loc"`
	Title   string   `xml:"image:title,omitempty"`
}

type SitemapURL struct {
	XMLName    xml.Name       `xml:"url"`
	Loc        string         `xml:"loc"`
	LastMod    string         `xml:"lastmod,omitempty"`
	ChangeFreq string         `xml:"changefreq,omitempty"`
	Priority   string         `xml:"priority,omitempty"`
	Images     []SitemapImage `xml:"image:image,omitempty"`
}

type URLSet struct {
	XMLName    xml.Name     `xml:"urlset"`
	XMLNS      string       `xml:"xmlns,attr"`
	XMLNSImage string       `xml:"xmlns:image,attr"`
	URLs       []SitemapURL `xml:"url"`
}

// GetSiteURL determines the canonical base URL of the website
func (s *Server) GetSiteURL(r *http.Request) string {
	// 1. Cek pengaturan eksplisit dari database MariaDB
	if custom := strings.TrimSpace(database.GetSetting(s.db, "site_url", "")); custom != "" {
		return strings.TrimRight(custom, "/")
	}

	// 2. Jika ada request HTTP aktif, tentukan scheme dan host
	if r != nil {
		scheme := "https"
		if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
			// Jika di localhost tanpa reverse proxy SSL
			if strings.HasPrefix(r.Host, "localhost") || strings.HasPrefix(r.Host, "127.0.0.1") {
				scheme = "http"
			}
		}
		host := r.Host
		if host != "" {
			return fmt.Sprintf("%s://%s", scheme, host)
		}
	}

	// 3. Fallback default ke domain produksi resmi
	return "https://cinembrot.my.id"
}

// PopulateSEO enriches PageData with world-class metadata, OpenGraph, Canonical, and Schema.org
func (s *Server) PopulateSEO(r *http.Request, data *PageData) {
	s.PopulatePageData(r, data)
	settings := database.GetAllSettings(s.db)

	siteURL := s.GetSiteURL(r)
	data.SiteURL = siteURL

	// Verifikasi Search Console
	data.GoogleVerification = settings["google_site_verification"]
	data.BingVerification = settings["bing_site_verification"]
	data.YandexVerification = settings["yandex_verification"]

	// Canonical URL
	if data.CanonicalURL == "" && r != nil {
		data.CanonicalURL = siteURL + r.URL.Path
	}

	// OpenGraph Type Default
	if data.OGType == "" {
		data.OGType = "website"
	}

	// Default Meta Image
	if data.MetaImage == "" {
		data.MetaImage = siteURL + "/favicon.png"
	} else if !strings.HasPrefix(data.MetaImage, "http://") && !strings.HasPrefix(data.MetaImage, "https://") {
		data.MetaImage = siteURL + data.MetaImage
	}

	// Default Meta Description jika belum dispesifikasikan oleh handler
	if data.MetaDescription == "" {
		if val, ok := settings["meta_description"]; ok && strings.TrimSpace(val) != "" {
			data.MetaDescription = val
		} else {
			data.MetaDescription = fmt.Sprintf("Nonton film streaming dan download gratis subtitle Indonesia legal kualitas HD. Koleksi lengkap film bioskop, anime terbaru, dan drama pendek hanya di %s.", data.SiteName)
		}
	}

	// Default Meta Keywords jika belum dispesifikasikan
	if data.MetaKeywords == "" {
		if val, ok := settings["meta_keywords"]; ok && strings.TrimSpace(val) != "" {
			data.MetaKeywords = val
		} else {
			data.MetaKeywords = fmt.Sprintf("nonton film sub indo, streaming movie %s, download film gratis, nonton anime sub indo, drama pendek, cinembrot, film bioskop terbaru, lk21, rebahin", data.SiteName)
		}
	}

	// JSON-LD Structured Data untuk Homepage jika belum diisi
	if data.JSONLD == "" && r != nil && (r.URL.Path == "/" || r.URL.Path == "") {
		data.JSONLD = s.GenerateJSONLDWebSite(siteURL, data.SiteName)
	}
}

// GenerateJSONLDMovie creates rich snippet Schema.org JSON-LD for Movie / TVSeries
func (s *Server) GenerateJSONLDMovie(siteURL string, m *model.Movie) template.HTML {
	if m == nil {
		return ""
	}

	schemaType := "Movie"
	if m.Type == "anime" || m.Type == "drama_pendek" || m.Type == "series" {
		schemaType = "TVSeries"
	}

	cleanSynopsis := scraper.CleanHTMLToPlainText(m.Synopsis)
	if len(cleanSynopsis) > 300 {
		cleanSynopsis = cleanSynopsis[:297] + "..."
	}
	if cleanSynopsis == "" {
		cleanSynopsis = fmt.Sprintf("Nonton streaming dan download film %s (%d) subtitle Indonesia gratis kualitas %s di CINEMBROT.", m.Title, m.Year, m.Quality)
	}

	posterURL := m.PosterURL
	if posterURL == "" {
		posterURL = m.ThumbnailURL
	}
	if posterURL != "" && !strings.HasPrefix(posterURL, "http://") && !strings.HasPrefix(posterURL, "https://") {
		posterURL = siteURL + posterURL
	}

	var genreNames []string
	for _, g := range m.Genres {
		genreNames = append(genreNames, g.Name)
	}

	var directors []map[string]string
	for _, d := range m.Directors {
		directors = append(directors, map[string]string{
			"@type": "Person",
			"name":  d.Name,
		})
	}

	var actors []map[string]string
	for _, a := range m.Actors {
		actors = append(actors, map[string]string{
			"@type": "Person",
			"name":  a.Name,
		})
	}

	// Format ISO 8601 Duration (e.g. PT120M)
	durationISO := ""
	if m.DurationMinutes > 0 {
		durationISO = fmt.Sprintf("PT%dM", m.DurationMinutes)
	}

	ratingVal := m.Rating
	if ratingVal <= 0 {
		ratingVal = 7.5 // Nilai wajar default untuk Rich Snippet
	}
	ratingCount := int((m.Views / 10) + 18)
	if ratingCount < 5 {
		ratingCount = 5
	}

	obj := map[string]interface{}{
		"@context":      "https://schema.org",
		"@type":         schemaType,
		"name":          m.Title,
		"headline":      fmt.Sprintf("%s (%d)", m.Title, m.Year),
		"description":   cleanSynopsis,
		"image":         posterURL,
		"inLanguage":    "id",
		"datePublished": fmt.Sprintf("%d-01-01", m.Year),
		"genre":         genreNames,
		"contentRating": m.AgeRating,
		"aggregateRating": map[string]interface{}{
			"@type":       "AggregateRating",
			"ratingValue": fmt.Sprintf("%.1f", ratingVal),
			"bestRating":  "10",
			"worstRating": "1",
			"ratingCount": strconv.Itoa(ratingCount),
		},
	}

	if m.OriginalTitle != "" && m.OriginalTitle != m.Title {
		obj["alternateName"] = m.OriginalTitle
	} else if m.AlternativeTitles != "" {
		obj["alternateName"] = m.AlternativeTitles
	}
	if durationISO != "" {
		obj["duration"] = durationISO
	}
	if len(directors) > 0 {
		obj["director"] = directors
	}
	if len(actors) > 0 {
		obj["actor"] = actors
	}
	if m.TrailerURL != "" {
		obj["trailer"] = map[string]interface{}{
			"@type":        "VideoObject",
			"name":         fmt.Sprintf("Trailer Resmi %s", m.Title),
			"embedUrl":     m.TrailerURL,
			"thumbnailUrl": posterURL,
			"description":  fmt.Sprintf("Tonton trailer resmi film %s (%d)", m.Title, m.Year),
			"uploadDate":   fmt.Sprintf("%d-01-01T00:00:00Z", m.Year),
		}
	}

	b, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return ""
	}
	return template.HTML(string(b))
}

// GenerateJSONLDWebSite creates Schema.org JSON-LD for WebSite with Sitelinks Searchbox
func (s *Server) GenerateJSONLDWebSite(siteURL string, siteName string) template.HTML {
	obj := map[string]interface{}{
		"@context":      "https://schema.org",
		"@type":         "WebSite",
		"name":          siteName,
		"alternateName": []string{"Cinembrot Streaming", "Cinembrot Nonton Film", "Cinembrot Anime"},
		"url":           siteURL + "/",
		"potentialAction": map[string]interface{}{
			"@type": "SearchAction",
			"target": map[string]interface{}{
				"@type":       "EntryPoint",
				"urlTemplate": fmt.Sprintf("%s/search?q={search_term_string}", siteURL),
			},
			"query-input": "required name=search_term_string",
		},
	}

	b, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return ""
	}
	return template.HTML(string(b))
}

// HandleSitemapXML generates an automated XML Sitemap indexing all pages and movies
func (s *Server) HandleSitemapXML(w http.ResponseWriter, r *http.Request) {
	siteURL := s.GetSiteURL(r)
	nowStr := time.Now().Format("2006-01-02T15:04:05Z07:00")

	// Static Core Pages
	urls := []SitemapURL{
		{
			Loc:        siteURL + "/",
			LastMod:    nowStr,
			ChangeFreq: "hourly",
			Priority:   "1.0",
		},
		{
			Loc:        siteURL + "/anime",
			LastMod:    nowStr,
			ChangeFreq: "daily",
			Priority:   "0.9",
		},
		{
			Loc:        siteURL + "/drama-pendek",
			LastMod:    nowStr,
			ChangeFreq: "daily",
			Priority:   "0.9",
		},
		{
			Loc:        siteURL + "/filter",
			LastMod:    nowStr,
			ChangeFreq: "daily",
			Priority:   "0.8",
		},
		{
			Loc:        siteURL + "/free",
			LastMod:    nowStr,
			ChangeFreq: "daily",
			Priority:   "0.8",
		},
	}

	// Fetch all movies from MariaDB
	type MovieSummary struct {
		Slug      string
		Title     string
		Type      string
		PosterURL string
		UpdatedAt time.Time
	}

	var movies []MovieSummary
	s.db.Model(&model.Movie{}).
		Select("slug, title, type, poster_url, updated_at").
		Order("id desc").
		Find(&movies)

	for _, m := range movies {
		item := SitemapURL{
			Loc:        fmt.Sprintf("%s/movie/%s", siteURL, m.Slug),
			LastMod:    m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			ChangeFreq: "weekly",
			Priority:   "0.8",
		}

		if m.PosterURL != "" {
			imgLoc := m.PosterURL
			if !strings.HasPrefix(imgLoc, "http://") && !strings.HasPrefix(imgLoc, "https://") {
				imgLoc = siteURL + imgLoc
			}
			item.Images = []SitemapImage{
				{
					Loc:   imgLoc,
					Title: fmt.Sprintf("Nonton %s Sub Indo", m.Title),
				},
			}
		}

		urls = append(urls, item)
	}

	urlSet := URLSet{
		XMLNS:      "http://www.sitemaps.org/schemas/sitemap/0.9",
		XMLNSImage: "http://www.google.com/schemas/sitemap-image/1.1",
		URLs:       urls,
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("X-Robots-Tag", "noindex, follow")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	_ = enc.Encode(urlSet)
}

// HandleRobotsTXT outputs a clean crawler instruction file with dynamic sitemap location
func (s *Server) HandleRobotsTXT(w http.ResponseWriter, r *http.Request) {
	siteURL := s.GetSiteURL(r)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")

	robotsContent := fmt.Sprintf(`# Robots.txt for %s
User-agent: *
Allow: /
Disallow: /admin/
Disallow: /api/
Disallow: /set-lang

# Crawl Delay untuk crawler ramah
Crawl-delay: 1

# XML Sitemap Resmi
Sitemap: %s/sitemap.xml
`, siteURL, siteURL)

	w.Write([]byte(robotsContent))
}
