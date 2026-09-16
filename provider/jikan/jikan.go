package jikan

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cinembrot/config"
	"cinembrot/model"
	"cinembrot/scraper"
)

const BaseURL = "https://api.jikan.moe/v4"

type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewClient(cfg *config.Config) *Client {
	timeout := 15
	if cfg != nil && cfg.ScraperTimeoutSec > 0 {
		timeout = cfg.ScraperTimeoutSec
	}
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

// JikanAnimeResponse represents the list response from Jikan v4
type JikanAnimeResponse struct {
	Pagination struct {
		LastVisiblePage int  `json:"last_visible_page"`
		HasNextPage     bool `json:"has_next_page"`
		CurrentPage     int  `json:"current_page"`
	} `json:"pagination"`
	Data []JikanAnime `json:"data"`
}

type JikanAnime struct {
	MalID         int      `json:"mal_id"`
	URL           string   `json:"url"`
	Title         string   `json:"title"`
	TitleEnglish  string   `json:"title_english"`
	TitleJapanese string   `json:"title_japanese"`
	Type          string   `json:"type"` // TV, Movie, OVA, Special, ONA
	Source        string   `json:"source"`
	Episodes      int      `json:"episodes"`
	Status        string   `json:"status"` // Currently Airing, Finished Airing
	Duration      string   `json:"duration"`
	Rating        string   `json:"rating"` // PG-13, R - 17+, etc.
	Score         float64  `json:"score"`
	ScoredBy      int      `json:"scored_by"`
	Synopsis      string   `json:"synopsis"`
	Year          int      `json:"year"`
	Images        struct {
		JPG struct {
			ImageURL      string `json:"image_url"`
			SmallImageURL string `json:"small_image_url"`
			LargeImageURL string `json:"large_image_url"`
		} `json:"jpg"`
		WebP struct {
			ImageURL      string `json:"image_url"`
			SmallImageURL string `json:"small_image_url"`
			LargeImageURL string `json:"large_image_url"`
		} `json:"webp"`
	} `json:"images"`
	Trailer struct {
		YoutubeID string `json:"youtube_id"`
		URL       string `json:"url"`
		EmbedURL  string `json:"embed_url"`
	} `json:"trailer"`
	Genres []struct {
		MalID int    `json:"mal_id"`
		Type  string `json:"type"`
		Name  string `json:"name"`
	} `json:"genres"`
	Studios []struct {
		MalID int    `json:"mal_id"`
		Type  string `json:"type"`
		Name  string `json:"name"`
	} `json:"studios"`
}

// FetchTopAnime fetches top/popular anime from MyAnimeList via Jikan API
func (c *Client) FetchTopAnime(limit int, page int) ([]model.Movie, error) {
	if limit <= 0 {
		limit = 15
	}
	if page <= 0 {
		page = 1
	}

	endpoint := fmt.Sprintf("%s/top/anime?filter=bypopularity&page=%d&limit=%d", BaseURL, page, limit)
	return c.fetchAndParse(endpoint)
}

// FetchSeasonalAnime fetches currently airing/seasonal anime
func (c *Client) FetchSeasonalAnime(limit int, page int) ([]model.Movie, error) {
	if limit <= 0 {
		limit = 15
	}
	if page <= 0 {
		page = 1
	}

	endpoint := fmt.Sprintf("%s/seasons/now?page=%d&limit=%d", BaseURL, page, limit)
	return c.fetchAndParse(endpoint)
}

// FetchAnimeByYear fetches anime released in a specific year from MyAnimeList/Jikan
func (c *Client) FetchAnimeByYear(year int, limit int, page int) ([]model.Movie, error) {
	movies, _, _, err := c.FetchAnimeByYearWithPagination(year, limit, page)
	return movies, err
}

// FetchAnimeByYearWithPagination fetches anime by release year and returns pagination metadata (lastVisiblePage, hasNextPage)
func (c *Client) FetchAnimeByYearWithPagination(year int, limit int, page int) ([]model.Movie, int, bool, error) {
	if limit <= 0 {
		limit = 25
	}
	if page <= 0 {
		page = 1
	}

	endpoint := fmt.Sprintf("%s/anime?start_date=%d-01-01&end_date=%d-12-31&order_by=popularity&sort=asc&page=%d&limit=%d",
		BaseURL, year, year, page, limit)
	return c.fetchAndParseWithPagination(endpoint)
}

func (c *Client) fetchAndParse(endpoint string) ([]model.Movie, error) {
	movies, _, _, err := c.fetchAndParseWithPagination(endpoint)
	return movies, err
}

func (c *Client) fetchAndParseWithPagination(endpoint string) ([]model.Movie, int, bool, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, false, err
	}

	ua := "CINEMBROT-AnimeScraper/1.0"
	if c.cfg != nil && c.cfg.ScraperUserAgent != "" {
		ua = c.cfg.ScraperUserAgent
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, false, fmt.Errorf("jikan api request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, false, fmt.Errorf("jikan api returned status %d: %s", resp.StatusCode, string(body))
	}

	var res JikanAnimeResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, 0, false, fmt.Errorf("failed to decode jikan response: %w", err)
	}

	var movies []model.Movie
	for _, a := range res.Data {
		movie := c.convertAnimeToMovie(a)
		movies = append(movies, movie)
	}

	lastPage := res.Pagination.LastVisiblePage
	if lastPage <= 0 {
		lastPage = 1
	}

	return movies, lastPage, res.Pagination.HasNextPage, nil
}

func (c *Client) convertAnimeToMovie(a JikanAnime) model.Movie {
	title := a.Title
	if a.TitleEnglish != "" {
		title = a.TitleEnglish
	}
	// Pastikan judul menggunakan alfabet Latin/QWERTY jika masih ada karakter non-Latin
	if scraper.ContainsNonLatin(title) {
		if a.TitleEnglish != "" && !scraper.ContainsNonLatin(a.TitleEnglish) {
			title = a.TitleEnglish
		} else if !scraper.ContainsNonLatin(a.Title) {
			title = a.Title
		}
	}

	altTitle := a.Title
	if a.TitleJapanese != "" {
		if altTitle != "" && altTitle != a.TitleJapanese {
			altTitle = altTitle + " / " + a.TitleJapanese
		} else {
			altTitle = a.TitleJapanese
		}
	}

	year := a.Year
	if year == 0 {
		year = time.Now().Year()
	}

	slugBase := scraper.Slugify(title)
	if slugBase == "" {
		slugBase = fmt.Sprintf("anime-%d", a.MalID)
	}
	slug := fmt.Sprintf("%s-%d", slugBase, year)

	// Status mapping
	status := "released"
	if strings.Contains(strings.ToLower(a.Status), "airing") {
		status = "ongoing"
	}

	// Poster selection (prioritize WebP, then JPG)
	poster := a.Images.WebP.LargeImageURL
	if poster == "" {
		poster = a.Images.JPG.LargeImageURL
	}
	thumb := a.Images.WebP.SmallImageURL
	if thumb == "" {
		thumb = a.Images.JPG.SmallImageURL
	}

	synopsis := scraper.CleanHTMLToPlainText(a.Synopsis)

	// Genres
	var genres []model.Genre
	for _, g := range a.Genres {
		cleanG := strings.TrimSpace(g.Name)
		if cleanG != "" {
			genres = append(genres, model.Genre{
				Name: cleanG,
				Slug: scraper.Slugify(cleanG),
			})
		}
	}

	// Directors / Studios
	var directors []model.Director
	for _, s := range a.Studios {
		cleanS := strings.TrimSpace(s.Name)
		if cleanS != "" {
			directors = append(directors, model.Director{
				Name: cleanS,
				Slug: scraper.Slugify(cleanS),
			})
		}
	}

	// Subtitle Candidates Links (Indonesian & English)
	searchQuery := url.QueryEscape(title)
	downloadLinks := []model.DownloadLink{
		{
			Provider:   "Subtitle Indonesia (SRT)",
			Quality:    "SRT",
			Resolution: "Sub",
			Format:     "SRT",
			URL:        fmt.Sprintf("https://subdl.com/search/%s", searchQuery),
			IsValid:    true,
			Status:     "ACTIVE",
		},
		{
			Provider:   "Subtitle English (SRT)",
			Quality:    "SRT",
			Resolution: "Sub",
			Format:     "SRT",
			URL:        fmt.Sprintf("https://www.opensubtitles.org/en/search/sublanguageid-eng/moviename-%s", searchQuery),
			IsValid:    true,
			Status:     "ACTIVE",
		},
	}

	return model.Movie{
		Slug:              slug,
		Title:             title,
		OriginalTitle:     a.TitleJapanese,
		AlternativeTitles: altTitle,
		Type:              "anime",
		Status:            status,
		Tagline:           fmt.Sprintf("Anime %s • %d Episodes • Skor %.1f/10", a.Type, a.Episodes, a.Score),
		Synopsis:          synopsis,
		Year:              year,
		DurationFormatted: a.Duration,
		Country:           "Japan",
		Language:          "Japanese",
		AgeRating:         a.Rating,
		Quality:           "HD 1080p",
		IsLegal:           true,
		IsFree:            true,
		Rating:            a.Score,
		IMDbRating:        a.Score,
		VoteCount:         a.ScoredBy,
		PosterURL:         poster,
		PosterThumbURL:    thumb,
		ThumbnailURL:      thumb,
		TrailerURL:        a.Trailer.URL,
		SourceWebsite:     "MyAnimeList / Jikan API",
		SourceURL:         a.URL,
		Genres:            genres,
		Directors:         directors,
		DownloadLinks:     downloadLinks,
	}
}
