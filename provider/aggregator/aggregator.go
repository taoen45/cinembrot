package aggregator

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"cinembrot/config"
	"cinembrot/model"
	"cinembrot/scraper"

	"github.com/PuerkitoBio/goquery"
)

// Client handles automated scraping of direct video download links (Google Drive, Acefile, Pixeldrain, Mediafire, Mega)
// from Indonesian & Asian movie, anime, and drama aggregators without manual input.
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewClient creates a new Aggregator download scraper client
func NewClient(cfg *config.Config) *Client {
	timeout := 15
	if cfg != nil && cfg.ScraperTimeoutSec > 0 {
		timeout = cfg.ScraperTimeoutSec
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DisableKeepAlives: false,
	}

	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   time.Duration(timeout) * time.Second,
		},
	}
}

// ScrapeDirectDownloads automatically discovers and extracts direct video download links for a given movie/anime/drama
func (c *Client) ScrapeDirectDownloads(movie *model.Movie) []model.DownloadLink {
	if movie == nil || strings.TrimSpace(movie.Title) == "" {
		return nil
	}

	cleanTitle := strings.TrimSpace(movie.Title)
	var allLinks []model.DownloadLink
	seenURLs := make(map[string]bool)

	addUnique := func(links []model.DownloadLink) {
		for _, l := range links {
			cleanU := strings.TrimSpace(l.URL)
			if cleanU == "" || seenURLs[cleanU] {
				continue
			}
			seenURLs[cleanU] = true
			l.MovieID = movie.ID
			allLinks = append(allLinks, l)
		}
	}

	isAnime := movie.Type == "anime"
	isDrama := movie.Type == "drama_pendek" || movie.Type == "series"

	// 1. Khusus Anime: Scrape Otakudesu & Samehadaku
	if isAnime {
		otakuLinks := c.scrapeOtakudesu(cleanTitle)
		addUnique(otakuLinks)

		if len(allLinks) == 0 && movie.OriginalTitle != "" {
			addUnique(c.scrapeOtakudesu(movie.OriginalTitle))
		}

		samehadakuLinks := c.scrapeSamehadaku(cleanTitle)
		addUnique(samehadakuLinks)
	}

	// 2. Khusus Drama Asia: Scrape Dramaid & Nodrakor
	if isDrama {
		dramaLinks := c.scrapeDramaid(cleanTitle)
		addUnique(dramaLinks)
	}

	// 3. Film Hollywood, Bioskop & Fallback Anime/Series: Scrape Pahe & Melongmovie
	if !isAnime || len(allLinks) == 0 {
		paheLinks := c.scrapePahe(cleanTitle, movie.Year)
		addUnique(paheLinks)

		melongLinks := c.scrapeMelongmovie(cleanTitle, movie.Year)
		addUnique(melongLinks)
	}

	return allLinks
}

// --- SCRAPER 1: PAHE (Film Hollywood, Box Office & Serial) ---
func (c *Client) scrapePahe(title string, year int) []model.DownloadLink {
	searchQuery := url.QueryEscape(title)
	mirrors := []string{"https://pahe.ink", "https://pahe.li"}

	var results []model.DownloadLink
	for _, base := range mirrors {
		searchURL := fmt.Sprintf("%s/?s=%s", base, searchQuery)
		doc, err := c.fetchDocument(searchURL)
		if err != nil || doc == nil {
			continue
		}

		// Cari artikel yang cocok dengan judul
		var targetPostURL string
		doc.Find("article.post, .content-area article, .post-box, .box-item, li.timeline-post, div.post").Each(func(i int, s *goquery.Selection) {
			if targetPostURL != "" {
				return
			}
			postTitle := strings.TrimSpace(s.Find(".post-box-title a, .entry-title a, .post-title a, h2 a").Text())
			postHref, exists := s.Find(".post-box-title a, .entry-title a, .post-title a, h2 a").Attr("href")
			if !exists || postHref == "" {
				return
			}

			// Cek kemiripan judul
			if matchTitle(postTitle, title) {
				targetPostURL = postHref
			}
		})

		if targetPostURL == "" {
			// Coba link pertama dari search result jika ada
			targetPostURL, _ = doc.Find(".post-box-title a, .entry-title a, .post-title a, h2 a").First().Attr("href")
		}

		if targetPostURL != "" {
			postDoc, err := c.fetchDocument(targetPostURL)
			if err == nil && postDoc != nil {
				results = append(results, c.extractPaheLinks(postDoc)...)
				if len(results) > 0 {
					break
				}
			}
		}
	}

	return results
}

func (c *Client) extractPaheLinks(doc *goquery.Document) []model.DownloadLink {
	var links []model.DownloadLink
	seen := make(map[string]bool)

	currentQuality := "720p HD"
	currentSize := "900 MB"

	doc.Find("p, div, a, li").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		textLower := strings.ToLower(text)

		// Deteksi kualitas
		if strings.Contains(textLower, "1080p") {
			currentQuality = "1080p FHD"
			currentSize = "1.8 GB"
		} else if strings.Contains(textLower, "720p") {
			currentQuality = "720p HD"
			currentSize = "950 MB"
		} else if strings.Contains(textLower, "480p") {
			currentQuality = "480p SD"
			currentSize = "450 MB"
		} else if strings.Contains(textLower, "2160p") || strings.Contains(textLower, "4k") {
			currentQuality = "4K UHD"
			currentSize = "4.5 GB"
		}

		// Ekstrak tag <a>
		s.Find("a[href]").Each(func(j int, a *goquery.Selection) {
			href, ok := a.Attr("href")
			if !ok || href == "" || seen[href] {
				return
			}

			anchorText := strings.TrimSpace(a.Text())
			providerName := detectHostProvider(href, anchorText)
			if providerName == "" {
				return
			}

			seen[href] = true
			links = append(links, model.DownloadLink{
				Provider:   providerName + " (Pahe)",
				Quality:    currentQuality,
				Resolution: resolveResolution(currentQuality),
				FileSize:   currentSize,
				Format:     "MP4",
				URL:        href,
				IsValid:    true,
				Status:     "ACTIVE",
			})
		})
	})

	return links
}

// --- SCRAPER 2: OTAKUDESU (Anime Subtitle Indonesia) ---
func (c *Client) scrapeOtakudesu(title string) []model.DownloadLink {
	searchQuery := url.QueryEscape(title)
	mirrors := []string{
		"https://otakudesu.blog",
		"https://otakudesu.best",
		"https://otakudesu.cam",
		"https://otakudesu.cloud",
	}

	var results []model.DownloadLink
	for _, base := range mirrors {
		searchURL := fmt.Sprintf("%s/?s=%s&post_type=anime", base, searchQuery)
		doc, err := c.fetchDocument(searchURL)
		if err != nil || doc == nil {
			continue
		}

		var targetAnimeURL string
		doc.Find("ul.chivsrc li").Each(func(i int, s *goquery.Selection) {
			if targetAnimeURL != "" {
				return
			}
			animeTitle := strings.TrimSpace(s.Find("h2 a").Text())
			href, ok := s.Find("h2 a").Attr("href")
			if ok && matchTitle(animeTitle, title) {
				targetAnimeURL = href
			}
		})

		if targetAnimeURL == "" {
			targetAnimeURL, _ = doc.Find("ul.chivsrc li h2 a").First().Attr("href")
		}

		if targetAnimeURL == "" {
			continue
		}

		animeDoc, err := c.fetchDocument(targetAnimeURL)
		if err != nil || animeDoc == nil {
			continue
		}

		seen := make(map[string]bool)

		// Cari kotak download (batch download atau episode download)
		animeDoc.Find(".download ul li, .batchlink ul li").Each(func(i int, s *goquery.Selection) {
			qualityText := strings.TrimSpace(s.Find("strong").Text())
			if qualityText == "" {
				qualityText = "720p HD"
			}

			s.Find("a[href]").Each(func(j int, a *goquery.Selection) {
				href, ok := a.Attr("href")
				if !ok || seen[href] {
					return
				}

				anchorText := strings.TrimSpace(a.Text())
				providerName := detectHostProvider(href, anchorText)
				if providerName == "" {
					return
				}

				seen[href] = true
				results = append(results, model.DownloadLink{
					Provider:   providerName + " (Hardsub Indo)",
					Quality:    cleanAnimeQuality(qualityText),
					Resolution: resolveResolution(qualityText),
					FileSize:   "350 MB",
					Format:     "MP4",
					URL:        href,
					IsValid:    true,
					Status:     "ACTIVE",
				})
			})
		})

		if len(results) > 0 {
			break
		}
	}

	return results
}

// --- SCRAPER 3: SAMEHADAKU (Anime Subtitle Indonesia) ---
func (c *Client) scrapeSamehadaku(title string) []model.DownloadLink {
	searchQuery := url.QueryEscape(title)
	mirrors := []string{
		"https://samehadaku.care",
		"https://samehadaku.email",
	}

	var results []model.DownloadLink
	for _, base := range mirrors {
		searchURL := fmt.Sprintf("%s/?s=%s", base, searchQuery)
		doc, err := c.fetchDocument(searchURL)
		if err != nil || doc == nil {
			continue
		}

		var targetURL string
		doc.Find("article.animpost, .animpost").Each(func(i int, s *goquery.Selection) {
			if targetURL != "" {
				return
			}
			postTitle := strings.TrimSpace(s.Find(".title").Text())
			href, ok := s.Find("a").Attr("href")
			if ok && matchTitle(postTitle, title) {
				targetURL = href
			}
		})

		if targetURL == "" {
			targetURL, _ = doc.Find("article.animpost a, .animpost a").First().Attr("href")
		}

		if targetURL == "" {
			continue
		}

		detailDoc, err := c.fetchDocument(targetURL)
		if err != nil || detailDoc == nil {
			continue
		}

		seen := make(map[string]bool)
		detailDoc.Find(".download-eps li, .areadl ul li").Each(func(i int, s *goquery.Selection) {
			qualityText := strings.TrimSpace(s.Find("strong").Text())
			if qualityText == "" {
				qualityText = "720p HD"
			}

			s.Find("a[href]").Each(func(j int, a *goquery.Selection) {
				href, ok := a.Attr("href")
				if !ok || seen[href] {
					return
				}

				anchorText := strings.TrimSpace(a.Text())
				providerName := detectHostProvider(href, anchorText)
				if providerName == "" {
					return
				}

				seen[href] = true
				results = append(results, model.DownloadLink{
					Provider:   providerName + " (Samehadaku)",
					Quality:    cleanAnimeQuality(qualityText),
					Resolution: resolveResolution(qualityText),
					FileSize:   "380 MB",
					Format:     "MP4",
					URL:        href,
					IsValid:    true,
					Status:     "ACTIVE",
				})
			})
		})

		if len(results) > 0 {
			break
		}
	}

	return results
}

// --- SCRAPER 4: MELONGMOVIE (Box Office & Bioskop Indonesia) ---
func (c *Client) scrapeMelongmovie(title string, year int) []model.DownloadLink {
	searchQuery := url.QueryEscape(title)
	searchURL := fmt.Sprintf("https://melongmovie.site/?s=%s", searchQuery)

	doc, err := c.fetchDocument(searchURL)
	if err != nil || doc == nil {
		return nil
	}

	var targetURL string
	doc.Find("article.item").Each(func(i int, s *goquery.Selection) {
		if targetURL != "" {
			return
		}
		movieTitle := strings.TrimSpace(s.Find(".entry-title a").Text())
		href, ok := s.Find(".entry-title a").Attr("href")
		if ok && matchTitle(movieTitle, title) {
			targetURL = href
		}
	})

	if targetURL == "" {
		targetURL, _ = doc.Find("article.item .entry-title a").First().Attr("href")
	}

	if targetURL == "" {
		return nil
	}

	detailDoc, err := c.fetchDocument(targetURL)
	if err != nil || detailDoc == nil {
		return nil
	}

	var links []model.DownloadLink
	seen := make(map[string]bool)

	detailDoc.Find(".box-download a[href], .download-links a[href], table a[href]").Each(func(i int, a *goquery.Selection) {
		href, ok := a.Attr("href")
		if !ok || seen[href] {
			return
		}

		anchorText := strings.TrimSpace(a.Text())
		providerName := detectHostProvider(href, anchorText)
		if providerName == "" {
			return
		}

		seen[href] = true
		links = append(links, model.DownloadLink{
			Provider:   providerName + " (Melongmovie)",
			Quality:    "720p HD",
			Resolution: "HD",
			FileSize:   "950 MB",
			Format:     "MP4",
			URL:        href,
			IsValid:    true,
			Status:     "ACTIVE",
		})
	})

	return links
}

// --- SCRAPER 5: DRAMAID (Serial Drama Asia: Korea, China, Jepang) ---
func (c *Client) scrapeDramaid(title string) []model.DownloadLink {
	searchQuery := url.QueryEscape(title)
	searchURL := fmt.Sprintf("https://dramaid.tv/?s=%s", searchQuery)

	doc, err := c.fetchDocument(searchURL)
	if err != nil || doc == nil {
		return nil
	}

	var targetURL string
	doc.Find("article.post, .entry-title a").Each(func(i int, s *goquery.Selection) {
		if targetURL != "" {
			return
		}
		postTitle := strings.TrimSpace(s.Text())
		href, ok := s.Attr("href")
		if !ok {
			href, ok = s.Find("a").Attr("href")
		}
		if ok && matchTitle(postTitle, title) {
			targetURL = href
		}
	})

	if targetURL == "" {
		targetURL, _ = doc.Find(".entry-title a").First().Attr("href")
	}

	if targetURL == "" {
		return nil
	}

	detailDoc, err := c.fetchDocument(targetURL)
	if err != nil || detailDoc == nil {
		return nil
	}

	var links []model.DownloadLink
	seen := make(map[string]bool)

	detailDoc.Find(".links_table a[href], .dl-box a[href]").Each(func(i int, a *goquery.Selection) {
		href, ok := a.Attr("href")
		if !ok || seen[href] {
			return
		}

		anchorText := strings.TrimSpace(a.Text())
		providerName := detectHostProvider(href, anchorText)
		if providerName == "" {
			return
		}

		seen[href] = true
		links = append(links, model.DownloadLink{
			Provider:   providerName + " (Dramaid)",
			Quality:    "720p HD",
			Resolution: "HD",
			FileSize:   "550 MB",
			Format:     "MP4",
			URL:        href,
			IsValid:    true,
			Status:     "ACTIVE",
		})
	})

	return links
}

// --- HELPER UTILITIES ---

func (c *Client) fetchDocument(targetURL string) (*goquery.Document, error) {
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}

	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
	if c.cfg != nil && c.cfg.ScraperUserAgent != "" {
		ua = c.cfg.ScraperUserAgent
	}

	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,id;q=0.8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error: %d", resp.StatusCode)
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

func matchTitle(candidate, target string) bool {
	cNorm := normalizeTitle(candidate)
	tNorm := normalizeTitle(target)
	if cNorm == "" || tNorm == "" {
		return false
	}
	return strings.Contains(cNorm, tNorm) || strings.Contains(tNorm, cNorm)
}

func normalizeTitle(s string) string {
	s = strings.ToLower(s)
	reg := regexp.MustCompile(`[^a-z0-9\s]+`)
	s = reg.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}

func detectHostProvider(rawURL, anchorText string) string {
	u := strings.ToLower(rawURL)
	t := strings.ToLower(anchorText)

	if strings.Contains(u, "drive.google.com") || strings.Contains(t, "gdrive") || strings.Contains(t, "google drive") {
		return "Google Drive"
	}
	if strings.Contains(u, "pixeldrain.com") || strings.Contains(t, "pixeldrain") {
		return "PixelDrain"
	}
	if strings.Contains(u, "buzzheavier.com") || strings.Contains(t, "buzzheavier") {
		return "Buzzheavier"
	}
	if strings.Contains(u, "acefile.co") || strings.Contains(t, "acefile") {
		return "Acefile"
	}
	if strings.Contains(u, "mega.nz") || strings.Contains(t, "mega") {
		return "Mega"
	}
	if strings.Contains(u, "mediafire.com") || strings.Contains(t, "mediafire") {
		return "Mediafire"
	}
	if strings.Contains(u, "fastdrive.io") || strings.Contains(t, "fastdrive") {
		return "FastDrive"
	}
	if strings.Contains(u, "1fichier.com") || strings.Contains(t, "1fichier") {
		return "1Fichier"
	}
	if strings.Contains(u, "gofile.io") || strings.Contains(t, "gofile") {
		return "Gofile"
	}

	return ""
}

func resolveResolution(q string) string {
	q = strings.ToLower(q)
	if strings.Contains(q, "1080p") || strings.Contains(q, "fhd") {
		return "FHD"
	}
	if strings.Contains(q, "720p") || strings.Contains(q, "hd") {
		return "HD"
	}
	if strings.Contains(q, "4k") || strings.Contains(q, "2160p") {
		return "UHD"
	}
	return "SD"
}

func cleanAnimeQuality(q string) string {
	clean := strings.TrimSpace(scraper.CleanHTMLToPlainText(q))
	if strings.Contains(clean, "1080p") {
		return "1080p FHD"
	}
	if strings.Contains(clean, "720p") {
		return "720p HD"
	}
	if strings.Contains(clean, "480p") {
		return "480p SD"
	}
	if strings.Contains(clean, "360p") {
		return "360p SD"
	}
	if clean == "" {
		return "720p HD"
	}
	return clean
}
