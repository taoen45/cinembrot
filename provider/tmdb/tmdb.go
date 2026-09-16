package tmdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"cinembrot/config"
	"cinembrot/model"
	"cinembrot/provider/subtitles"
	"cinembrot/scraper"
)

const BaseURL = "https://api.themoviedb.org/3"
const ImageBaseURL = "https://image.tmdb.org/t/p/original"
const ThumbnailBaseURL = "https://image.tmdb.org/t/p/w500"

// FreekeysPool contains TMDb API key pool from rickylawson/freekeys
var FreekeysPool = []string{
	"fb7bb23f03b6994dafc674c074d01761",
	"e55425032d3d0f371fc776f302e7c09b",
	"8301a21598f8b45668d5711a814f01f6",
	"8cf43ad9c085135b9479ad5cf6bbcbda",
	"da63548086e399ffc910fbc08526df05",
	"13e53ff644a8bd4ba37b3e1044ad24f3",
	"269890f657dddf4635473cf4cf456576",
	"a2f888b27315e62e471b2d587048f32e",
	"8476a7ab80ad76f0936744df0430e67c",
	"5622cafbfe8f8cfe358a29c53e19bba0",
	"ae4bd1b6fce2a5648671bfc171d15ba4",
	"257654f35e3dff105574f97fb4b97035",
	"2f4038e83265214a0dcd6ec2eb3276f5",
	"9e43f45f94705cc8e1d5a0400d19a7b7",
	"af6887753365e14160254ac7f4345dd2",
	"06f10fc8741a672af455421c239a1ffc",
	"09ad8ace66eec34302943272db0e8d2c",
}

type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.ScraperTimeoutSec) * time.Second,
		},
	}
}

// GetAPIKey returns user configured key or rotates from freekeys pool
func (c *Client) GetAPIKey() string {
	if c.cfg.TMDBAPIKey != "" {
		return c.cfg.TMDBAPIKey
	}
	// Fallback to first active key from freekeys pool
	return FreekeysPool[0]
}

// SearchMovieResponse structure from TMDb
type SearchMovieResponse struct {
	Page         int `json:"page"`
	TotalResults int `json:"total_results"`
	TotalPages   int `json:"total_pages"`
	Results      []struct {
		ID               int     `json:"id"`
		Title            string  `json:"title"`
		OriginalTitle    string  `json:"original_title"`
		Overview         string  `json:"overview"`
		ReleaseDate      string  `json:"release_date"`
		PosterPath       string  `json:"poster_path"`
		BackdropPath     string  `json:"backdrop_path"`
		VoteAverage      float64 `json:"vote_average"`
		VoteCount        int     `json:"vote_count"`
		Popularity       float64 `json:"popularity"`
		OriginalLanguage string  `json:"original_language"`
	} `json:"results"`
}

// MovieDetailResponse represents full movie details from TMDb with appended credits & videos
type MovieDetailResponse struct {
	ID               int      `json:"id"`
	Title            string   `json:"title"`
	OriginalTitle    string   `json:"original_title"`
	Tagline          string   `json:"tagline"`
	Overview         string   `json:"overview"`
	ReleaseDate      string   `json:"release_date"`
	Runtime          int      `json:"runtime"`
	Status           string   `json:"status"`
	VoteAverage      float64  `json:"vote_average"`
	VoteCount        int      `json:"vote_count"`
	Popularity       float64  `json:"popularity"`
	PosterPath       string   `json:"poster_path"`
	BackdropPath     string   `json:"backdrop_path"`
	OriginalLanguage string   `json:"original_language"`
	Budget           int64    `json:"budget"`
	Revenue          int64    `json:"revenue"`
	Genres           []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"genres"`
	ProductionCountries []struct {
		Iso31661 string `json:"iso_3166_1"`
		Name     string `json:"name"`
	} `json:"production_countries"`
	SpokenLanguages []struct {
		EnglishName string `json:"english_name"`
		Name        string `json:"name"`
	} `json:"spoken_languages"`
	Credits struct {
		Cast []struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Character   string `json:"character"`
			ProfilePath string `json:"profile_path"`
			Order       int    `json:"order"`
		} `json:"cast"`
		Crew []struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Job         string `json:"job"`
			Department  string `json:"department"`
			ProfilePath string `json:"profile_path"`
		} `json:"crew"`
	} `json:"credits"`
	Videos struct {
		Results []struct {
			Key      string `json:"key"`
			Site     string `json:"site"`
			Type     string `json:"type"`
			Official bool   `json:"official"`
		} `json:"results"`
	} `json:"videos"`
	Translations TMDbTranslations `json:"translations"`
}

type TMDbTranslations struct {
	Translations []struct {
		Iso31661    string `json:"iso_3166_1"`
		Iso6391     string `json:"iso_639_1"`
		Name        string `json:"name"`
		EnglishName string `json:"english_name"`
		Data        struct {
			Name     string `json:"name"`
			Title    string `json:"title"`
			Overview string `json:"overview"`
			Tagline  string `json:"tagline"`
		} `json:"data"`
	} `json:"translations"`
}

// extractEnglishInfo extracts English title/name and overview from TMDb translations
func extractEnglishInfo(trans TMDbTranslations) (enTitle string, enOverview string) {
	for _, t := range trans.Translations {
		if t.Iso6391 == "en" {
			name := strings.TrimSpace(t.Data.Name)
			if name == "" {
				name = strings.TrimSpace(t.Data.Title)
			}
			if name != "" && !scraper.ContainsNonLatin(name) {
				enTitle = name
			}
			if strings.TrimSpace(t.Data.Overview) != "" {
				enOverview = strings.TrimSpace(t.Data.Overview)
			}
			if enTitle != "" && enOverview != "" {
				break
			}
		}
	}
	return
}

// SearchMovie searches TMDb for a movie title
func (c *Client) SearchMovie(title string, year int) (*SearchMovieResponse, error) {
	apiKey := c.GetAPIKey()

	queryURL := fmt.Sprintf("%s/search/movie?api_key=%s&query=%s&language=%s",
		BaseURL, apiKey, url.QueryEscape(title), c.cfg.TMDBLanguage)

	if year > 0 {
		queryURL += fmt.Sprintf("&year=%d", year)
	}

	req, err := http.NewRequest("GET", queryURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDb API returned status %d", resp.StatusCode)
	}

	var result SearchMovieResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SearchTV searches TMDb for a TV series / anime / drama title
func (c *Client) SearchTV(title string) (*SearchTVResponse, error) {
	apiKey := c.GetAPIKey()

	queryURL := fmt.Sprintf("%s/search/tv?api_key=%s&query=%s&language=%s",
		BaseURL, apiKey, url.QueryEscape(title), c.cfg.TMDBLanguage)

	req, err := http.NewRequest("GET", queryURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDb Search TV API returned status %d", resp.StatusCode)
	}

	var result SearchTVResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetMovieDetails fetches full rich movie metadata by TMDb ID
func (c *Client) GetMovieDetails(tmdbID int) (*model.Movie, error) {
	apiKey := c.GetAPIKey()

	detailURL := fmt.Sprintf("%s/movie/%d?api_key=%s&language=%s&append_to_response=credits,videos,translations",
		BaseURL, tmdbID, apiKey, c.cfg.TMDBLanguage)

	req, err := http.NewRequest("GET", detailURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDb API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var res MovieDetailResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}

	// Format release date and year
	var releaseDate *time.Time
	var year int
	if res.ReleaseDate != "" {
		if t, err := time.Parse("2006-01-02", res.ReleaseDate); err == nil {
			releaseDate = &t
			year = t.Year()
		}
	}

	// Format duration
	durationFormatted := ""
	if res.Runtime > 0 {
		hours := res.Runtime / 60
		mins := res.Runtime % 60
		if hours > 0 {
			durationFormatted = fmt.Sprintf("%dh %dm", hours, mins)
		} else {
			durationFormatted = fmt.Sprintf("%d min", mins)
		}
	}

	// Format genres
	var genres []model.Genre
	for _, g := range res.Genres {
		genres = append(genres, model.Genre{
			Name: g.Name,
			Slug: scraper.Slugify(g.Name),
		})
	}

	// Format Directors & Cast
	var directors []model.Director
	for _, crew := range res.Credits.Crew {
		if crew.Job == "Director" {
			photoURL := ""
			if crew.ProfilePath != "" {
				photoURL = ThumbnailBaseURL + crew.ProfilePath
			}
			directors = append(directors, model.Director{
				Name:     crew.Name,
				Slug:     scraper.Slugify(crew.Name),
				PhotoURL: photoURL,
			})
		}
	}

	var actors []model.Actor
	for i, cast := range res.Credits.Cast {
		if i >= 10 { // Top 10 main cast members
			break
		}
		photoURL := ""
		if cast.ProfilePath != "" {
			photoURL = ThumbnailBaseURL + cast.ProfilePath
		}
		actors = append(actors, model.Actor{
			Name:          cast.Name,
			Slug:          scraper.Slugify(cast.Name),
			CharacterName: cast.Character,
			PhotoURL:      photoURL,
		})
	}

	// Find trailer URL from YouTube
	trailerURL := ""
	for _, v := range res.Videos.Results {
		if v.Site == "YouTube" && (v.Type == "Trailer" || v.Type == "Teaser") {
			trailerURL = fmt.Sprintf("https://www.youtube.com/watch?v=%s", v.Key)
			if v.Official {
				break
			}
		}
	}

	// Country & Language
	country := ""
	if len(res.ProductionCountries) > 0 {
		country = res.ProductionCountries[0].Name
	}
	language := ""
	if len(res.SpokenLanguages) > 0 {
		language = res.SpokenLanguages[0].EnglishName
	}

	posterURL := ""
	if res.PosterPath != "" {
		posterURL = ImageBaseURL + res.PosterPath
	}
	backdropURL := ""
	if res.BackdropPath != "" {
		backdropURL = ImageBaseURL + res.BackdropPath
	}

	title := res.Title
	originalTitle := res.OriginalTitle
	alternativeTitles := ""
	overview := res.Overview

	enTitle, enOverview := extractEnglishInfo(res.Translations)

	if scraper.ContainsNonLatin(title) {
		if enTitle != "" {
			alternativeTitles = title
			title = enTitle
		} else if !scraper.ContainsNonLatin(originalTitle) && originalTitle != "" {
			alternativeTitles = title
			title = originalTitle
		}
	} else if originalTitle != "" && originalTitle != title {
		alternativeTitles = originalTitle
	}

	if strings.TrimSpace(overview) == "" && enOverview != "" {
		overview = enOverview
	}

	slugBase := scraper.Slugify(title)
	if slugBase == "" {
		slugBase = fmt.Sprintf("movie-%d", res.ID)
	}

	movie := &model.Movie{
		Title:             title,
		OriginalTitle:     originalTitle,
		AlternativeTitles: alternativeTitles,
		Slug:              fmt.Sprintf("%s-%d", slugBase, year),
		Type:              "movie",
		Status:            strings.ToLower(res.Status),
		Tagline:           res.Tagline,
		Synopsis:          overview,
		ReleaseDate:       releaseDate,
		Year:              year,
		DurationMinutes:   res.Runtime,
		DurationFormatted: durationFormatted,
		Country:           country,
		Language:          language,
		Quality:           "4K UHD / HD",
		IsLegal:           true,
		IsFree:            false,
		LicenseType:       "Commercial / Copyrighted",
		LicenseName:       "All Rights Reserved (Commercial Copyright)",
		LicenseURL:        "https://www.themoviedb.org/terms-of-use",
		TMDbRating:        res.VoteAverage,
		Rating:            res.VoteAverage,
		VoteCount:         res.VoteCount,
		Popularity:        res.Popularity,
		PosterURL:         posterURL,
		BackdropURL:       backdropURL,
		ThumbnailURL:      ThumbnailBaseURL + res.PosterPath,
		TrailerURL:        trailerURL,
		SourceWebsite:     "themoviedb.org",
		SourceURL:         fmt.Sprintf("https://www.themoviedb.org/movie/%d", res.ID),
		Genres:            genres,
		Directors:         directors,
		Actors:            actors,
		RawMetadata:       string(body),
	}

	// Subtitle Candidates & Download Links (Indonesian & English)
	movie.DownloadLinks = subtitles.GenerateSubtitleDownloadLinks(title, year)

	return movie, nil
}

// FetchPopularMovies retrieves top popular movies from TMDb
func (c *Client) FetchPopularMovies(page int) ([]model.Movie, error) {
	apiKey := c.GetAPIKey()

	if page <= 0 {
		page = 1
	}

	popularURL := fmt.Sprintf("%s/movie/popular?api_key=%s&language=%s&page=%d",
		BaseURL, apiKey, c.cfg.TMDBLanguage, page)

	req, err := http.NewRequest("GET", popularURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDb API returned status %d", resp.StatusCode)
	}

	var searchRes SearchMovieResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchRes); err != nil {
		return nil, err
	}

	var movies []model.Movie
	for _, item := range searchRes.Results {
		movie, err := c.GetMovieDetails(item.ID)
		if err != nil {
			continue
		}
		movies = append(movies, *movie)
	}

	return movies, nil
}

// DiscoverMoviesByYear retrieves top movies released in a specific year
func (c *Client) DiscoverMoviesByYear(year int, page int) ([]model.Movie, error) {
	apiKey := c.GetAPIKey()

	if page <= 0 {
		page = 1
	}

	discoverURL := fmt.Sprintf("%s/discover/movie?api_key=%s&primary_release_year=%d&sort_by=popularity.desc&page=%d&language=%s",
		BaseURL, apiKey, year, page, c.cfg.TMDBLanguage)

	req, err := http.NewRequest("GET", discoverURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDb API returned status %d", resp.StatusCode)
	}

	var searchRes SearchMovieResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchRes); err != nil {
		return nil, err
	}

	var movies []model.Movie
	for _, item := range searchRes.Results {
		movie, err := c.GetMovieDetails(item.ID)
		if err != nil {
			continue
		}
		movies = append(movies, *movie)
	}

	return movies, nil
}

// SearchTVResponse structure from TMDb Discover TV
type SearchTVResponse struct {
	Page         int `json:"page"`
	TotalResults int `json:"total_results"`
	TotalPages   int `json:"total_pages"`
	Results      []struct {
		ID               int      `json:"id"`
		Name             string   `json:"name"`
		OriginalName     string   `json:"original_name"`
		OriginalLanguage string   `json:"original_language"`
		Overview         string   `json:"overview"`
		PosterPath       string   `json:"poster_path"`
		BackdropPath     string   `json:"backdrop_path"`
		FirstAirDate     string   `json:"first_air_date"`
		VoteAverage      float64  `json:"vote_average"`
		VoteCount        int      `json:"vote_count"`
		Popularity       float64  `json:"popularity"`
	} `json:"results"`
}

// TVDetailResponse represents full response from TMDb TV details API
type TVDetailResponse struct {
	ID               int      `json:"id"`
	Name             string   `json:"name"`
	OriginalName     string   `json:"original_name"`
	Overview         string   `json:"overview"`
	Tagline          string   `json:"tagline"`
	FirstAirDate     string   `json:"first_air_date"`
	PosterPath       string   `json:"poster_path"`
	BackdropPath     string   `json:"backdrop_path"`
	VoteAverage      float64  `json:"vote_average"`
	VoteCount        int      `json:"vote_count"`
	Popularity       float64  `json:"popularity"`
	Status           string   `json:"status"`
	NumberOfEpisodes int      `json:"number_of_episodes"`
	NumberOfSeasons  int      `json:"number_of_seasons"`
	EpisodeRunTime   []int    `json:"episode_run_time"`
	OriginCountry    []string `json:"origin_country"`
	OriginalLanguage string   `json:"original_language"`
	Genres           []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"genres"`
	Credits struct {
		Cast []struct {
			Name        string `json:"name"`
			Character   string `json:"character"`
			ProfilePath string `json:"profile_path"`
			Order       int    `json:"order"`
		} `json:"cast"`
		Crew []struct {
			Name        string `json:"name"`
			Job         string `json:"job"`
			ProfilePath string `json:"profile_path"`
		} `json:"crew"`
	} `json:"credits"`
	Videos struct {
		Results []struct {
			Key  string `json:"key"`
			Site string `json:"site"`
			Type string `json:"type"`
		} `json:"results"`
	} `json:"videos"`
	Translations TMDbTranslations `json:"translations"`
}

// GetTVDetails fetches rich TV drama metadata by TMDb TV ID
func (c *Client) GetTVDetails(tvID int) (*model.Movie, error) {
	apiKey := c.GetAPIKey()

	detailURL := fmt.Sprintf("%s/tv/%d?api_key=%s&language=%s&append_to_response=credits,videos,translations",
		BaseURL, tvID, apiKey, c.cfg.TMDBLanguage)

	req, err := http.NewRequest("GET", detailURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDb TV API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var res TVDetailResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}

	// Format release date and year
	var releaseDate *time.Time
	var year int
	if res.FirstAirDate != "" {
		if t, err := time.Parse("2006-01-02", res.FirstAirDate); err == nil {
			releaseDate = &t
			year = t.Year()
		}
	}
	if year == 0 {
		year = time.Now().Year()
	}

	// Format duration
	durationFormatted := ""
	runtime := 0
	if len(res.EpisodeRunTime) > 0 {
		runtime = res.EpisodeRunTime[0]
		durationFormatted = fmt.Sprintf("%d min/ep", runtime)
	}
	if res.NumberOfEpisodes > 0 {
		if durationFormatted != "" {
			durationFormatted = fmt.Sprintf("%s (%d eps)", durationFormatted, res.NumberOfEpisodes)
		} else {
			durationFormatted = fmt.Sprintf("%d Episodes", res.NumberOfEpisodes)
		}
	}

	// Format genres
	var genres []model.Genre
	for _, g := range res.Genres {
		genres = append(genres, model.Genre{
			Name: g.Name,
			Slug: scraper.Slugify(g.Name),
		})
	}

	// Format Directors & Cast
	var directors []model.Director
	for _, crew := range res.Credits.Crew {
		if crew.Job == "Director" || crew.Job == "Executive Producer" {
			photoURL := ""
			if crew.ProfilePath != "" {
				photoURL = ThumbnailBaseURL + crew.ProfilePath
			}
			directors = append(directors, model.Director{
				Name:     crew.Name,
				Slug:     scraper.Slugify(crew.Name),
				PhotoURL: photoURL,
			})
			if len(directors) >= 3 {
				break
			}
		}
	}

	var actors []model.Actor
	for _, cast := range res.Credits.Cast {
		photoURL := ""
		if cast.ProfilePath != "" {
			photoURL = ThumbnailBaseURL + cast.ProfilePath
		}
		actors = append(actors, model.Actor{
			Name:          cast.Name,
			Slug:          scraper.Slugify(cast.Name),
			CharacterName: cast.Character,
			PhotoURL:      photoURL,
		})
		if len(actors) >= 12 {
			break
		}
	}

	trailerURL := ""
	for _, v := range res.Videos.Results {
		if v.Site == "YouTube" && (v.Type == "Trailer" || v.Type == "Teaser") {
			trailerURL = fmt.Sprintf("https://www.youtube.com/watch?v=%s", v.Key)
			break
		}
	}

	country := ""
	if len(res.OriginCountry) > 0 {
		country = res.OriginCountry[0]
	}
	language := res.OriginalLanguage

	posterURL := ""
	if res.PosterPath != "" {
		posterURL = ImageBaseURL + res.PosterPath
	}
	backdropURL := ""
	if res.BackdropPath != "" {
		backdropURL = ImageBaseURL + res.BackdropPath
	}

	// Status mapping
	status := "released"
	if strings.Contains(strings.ToLower(res.Status), "returning") || strings.Contains(strings.ToLower(res.Status), "in production") {
		status = "ongoing"
	}

	title := res.Name
	originalTitle := res.OriginalName
	alternativeTitles := ""
	overview := scraper.CleanHTMLToPlainText(res.Overview)

	enTitle, enOverview := extractEnglishInfo(res.Translations)

	if scraper.ContainsNonLatin(title) {
		if enTitle != "" {
			alternativeTitles = title
			title = enTitle
		} else if !scraper.ContainsNonLatin(originalTitle) && originalTitle != "" {
			alternativeTitles = title
			title = originalTitle
		}
	} else if originalTitle != "" && originalTitle != title {
		alternativeTitles = originalTitle
	}

	if strings.TrimSpace(overview) == "" && enOverview != "" {
		overview = scraper.CleanHTMLToPlainText(enOverview)
	}
	if strings.TrimSpace(overview) == "" {
		overview = c.GetEnglishSynopsis(fmt.Sprintf("https://www.themoviedb.org/tv/%d", res.ID), title, "tv")
	}

	slugBase := scraper.Slugify(title)
	if slugBase == "" {
		slugBase = fmt.Sprintf("tv-%d", res.ID)
	}

	// Subtitle Candidates (Indonesian & English) using English / Latin title
	downloadLinks := subtitles.GenerateSubtitleDownloadLinks(title, year)

	movie := &model.Movie{
		Title:             title,
		OriginalTitle:     originalTitle,
		AlternativeTitles: alternativeTitles,
		Slug:              fmt.Sprintf("%s-%d", slugBase, year),
		Type:              "drama_pendek",
		Status:            status,
		Tagline:           res.Tagline,
		Synopsis:          overview,
		ReleaseDate:       releaseDate,
		Year:              year,
		DurationMinutes:   runtime,
		DurationFormatted: durationFormatted,
		Country:           country,
		Language:          language,
		Quality:           "HD 1080p",
		IsLegal:           true,
		IsFree:            true,
		LicenseType:       "Commercial / Promotional",
		LicenseName:       "Promotional Metadata (TMDb TV API)",
		LicenseURL:        "https://www.themoviedb.org/terms-of-use",
		TMDbRating:        res.VoteAverage,
		Rating:            res.VoteAverage,
		VoteCount:         res.VoteCount,
		Popularity:        res.Popularity,
		PosterURL:         posterURL,
		BackdropURL:       backdropURL,
		ThumbnailURL:      ThumbnailBaseURL + res.PosterPath,
		TrailerURL:        trailerURL,
		SourceWebsite:     "themoviedb.org (TV)",
		SourceURL:         fmt.Sprintf("https://www.themoviedb.org/tv/%d", res.ID),
		Genres:            genres,
		Directors:         directors,
		Actors:            actors,
		DownloadLinks:     downloadLinks,
		RawMetadata:       string(body),
	}

	return movie, nil
}

// DiscoverAsianDramas retrieves top Asian dramas (Korean, Chinese, Japanese, Thai) with optional year filter
func (c *Client) DiscoverAsianDramas(lang string, page int, year ...int) ([]model.Movie, error) {
	apiKey := c.GetAPIKey()

	if page <= 0 {
		page = 1
	}

	langParam := "ko|zh|ja|th"
	if lang != "" && lang != "all" {
		langParam = lang
	}

	yearFilter := ""
	if len(year) > 0 && year[0] > 0 {
		yearFilter = fmt.Sprintf("&first_air_date_year=%d", year[0])
	}

	discoverURL := fmt.Sprintf("%s/discover/tv?api_key=%s&with_original_language=%s&sort_by=popularity.desc&page=%d&language=%s%s",
		BaseURL, apiKey, langParam, page, c.cfg.TMDBLanguage, yearFilter)

	req, err := http.NewRequest("GET", discoverURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDb Discover TV API returned status %d", resp.StatusCode)
	}

	var searchRes SearchTVResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchRes); err != nil {
		return nil, err
	}

	var dramas []model.Movie
	for _, item := range searchRes.Results {
		drama, err := c.GetTVDetails(item.ID)
		if err != nil {
			continue
		}
		dramas = append(dramas, *drama)
	}

	return dramas, nil
}

// DiscoverAsianDramasWithTotal retrieves top Asian dramas and returns the total available pages from TMDb
func (c *Client) DiscoverAsianDramasWithTotal(lang string, page int, year ...int) ([]model.Movie, int, error) {
	apiKey := c.GetAPIKey()

	if page <= 0 {
		page = 1
	}

	langParam := "ko|zh|ja|th"
	if lang != "" && lang != "all" {
		langParam = lang
	}

	yearFilter := ""
	if len(year) > 0 && year[0] > 0 {
		yearFilter = fmt.Sprintf("&first_air_date_year=%d", year[0])
	}

	discoverURL := fmt.Sprintf("%s/discover/tv?api_key=%s&with_original_language=%s&sort_by=popularity.desc&page=%d&language=%s%s",
		BaseURL, apiKey, langParam, page, c.cfg.TMDBLanguage, yearFilter)

	req, err := http.NewRequest("GET", discoverURL, nil)
	if err != nil {
		return nil, 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("TMDb Discover TV API returned status %d", resp.StatusCode)
	}

	var searchRes SearchTVResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchRes); err != nil {
		return nil, 0, err
	}

	var dramas []model.Movie
	for _, item := range searchRes.Results {
		drama, err := c.GetTVDetails(item.ID)
		if err != nil {
			continue
		}
		dramas = append(dramas, *drama)
	}

	return dramas, searchRes.TotalPages, nil
}

// DiscoverAnime retrieves anime series from TMDb (with_genres=16, with_original_language=ja) with custom sort_by and year
func (c *Client) DiscoverAnime(limit int, page int, year int, sortBy ...string) ([]model.Movie, error) {
	animes, _, err := c.DiscoverAnimeWithTotal(limit, page, year, sortBy...)
	return animes, err
}

// DiscoverAnimeWithTotal retrieves anime series and returns total available pages
func (c *Client) DiscoverAnimeWithTotal(limit int, page int, year int, sortBy ...string) ([]model.Movie, int, error) {
	apiKey := c.GetAPIKey()

	if page <= 0 {
		page = 1
	}

	sort := "popularity.desc"
	if len(sortBy) > 0 && sortBy[0] != "" {
		sort = sortBy[0]
	}

	yearFilter := ""
	if year > 0 {
		yearFilter = fmt.Sprintf("&first_air_date_year=%d", year)
	}

	discoverURL := fmt.Sprintf("%s/discover/tv?api_key=%s&with_genres=16&with_original_language=ja&sort_by=%s&page=%d&language=%s%s",
		BaseURL, apiKey, sort, page, c.cfg.TMDBLanguage, yearFilter)

	req, err := http.NewRequest("GET", discoverURL, nil)
	if err != nil {
		return nil, 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("TMDb Discover Anime API returned status %d", resp.StatusCode)
	}

	var searchRes SearchTVResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchRes); err != nil {
		return nil, 0, err
	}

	var animes []model.Movie
	for _, item := range searchRes.Results {
		anime, err := c.GetTVDetails(item.ID)
		if err != nil {
			continue
		}
		anime.Type = "anime"
		animes = append(animes, *anime)
		if limit > 0 && len(animes) >= limit {
			break
		}
	}

	return animes, searchRes.TotalPages, nil
}

// DiscoverHollywoodMovies retrieves top blockbuster Hollywood / Box Office movies from TMDb
func (c *Client) DiscoverHollywoodMovies(category string, page int, year ...int) ([]model.Movie, error) {
	movies, _, err := c.DiscoverHollywoodMoviesWithTotal(category, page, year...)
	return movies, err
}

// DiscoverHollywoodMoviesWithTotal retrieves Hollywood movies and returns total available pages
func (c *Client) DiscoverHollywoodMoviesWithTotal(category string, page int, year ...int) ([]model.Movie, int, error) {
	apiKey := c.GetAPIKey()

	if page <= 0 {
		page = 1
	}

	sortParam := "revenue.desc"
	minVotes := 100
	switch category {
	case "popular":
		sortParam = "popularity.desc"
		minVotes = 50
	case "top_rated":
		sortParam = "vote_average.desc"
		minVotes = 300
	case "now_playing":
		sortParam = "popularity.desc"
		minVotes = 20
	default:
		// "boxoffice"
		sortParam = "revenue.desc"
		minVotes = 100
	}

	yearFilter := ""
	if len(year) > 0 && year[0] > 0 {
		yearFilter = fmt.Sprintf("&primary_release_year=%d", year[0])
	}

	discoverURL := fmt.Sprintf("%s/discover/movie?api_key=%s&with_original_language=en&sort_by=%s&vote_count.gte=%d&page=%d&language=%s%s",
		BaseURL, apiKey, sortParam, minVotes, page, c.cfg.TMDBLanguage, yearFilter)

	req, err := http.NewRequest("GET", discoverURL, nil)
	if err != nil {
		return nil, 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("TMDb Discover Movie API returned status %d", resp.StatusCode)
	}

	var searchRes SearchMovieResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchRes); err != nil {
		return nil, 0, err
	}

	var movies []model.Movie
	for _, item := range searchRes.Results {
		movie, err := c.GetMovieDetails(item.ID)
		if err != nil {
			continue
		}
		movie.Type = "hollywood"
		movies = append(movies, *movie)
	}

	return movies, searchRes.TotalPages, nil
}

var (
	synopsisCache   = make(map[string]string)
	synopsisCacheMu sync.RWMutex
)

// GetEnglishSynopsis retrieves English synopsis/overview from TMDb by sourceURL or title
func (c *Client) GetEnglishSynopsis(sourceURL, title, mediaType string) string {
	cacheKey := sourceURL
	if cacheKey == "" {
		cacheKey = title + ":" + mediaType
	}

	synopsisCacheMu.RLock()
	if cached, ok := synopsisCache[cacheKey]; ok {
		synopsisCacheMu.RUnlock()
		return cached
	}
	synopsisCacheMu.RUnlock()

	apiKey := c.GetAPIKey()
	var tmdbID int
	isTV := mediaType == "anime" || mediaType == "drama_pendek" || mediaType == "series" || strings.Contains(sourceURL, "/tv/")

	// Try extracting ID from sourceURL: e.g. https://www.themoviedb.org/tv/99516 or https://www.themoviedb.org/movie/1234
	if strings.Contains(sourceURL, "themoviedb.org") {
		parts := strings.Split(sourceURL, "/")
		for i, part := range parts {
			if (part == "tv" || part == "movie") && i+1 < len(parts) {
				if id, err := strconv.Atoi(parts[i+1]); err == nil && id > 0 {
					tmdbID = id
					if part == "tv" {
						isTV = true
					} else {
						isTV = false
					}
					break
				}
			}
		}
	}

	// Fetch detail with language=en-US
	if tmdbID > 0 {
		var endpoint string
		if isTV {
			endpoint = fmt.Sprintf("%s/tv/%d?api_key=%s&language=en-US", BaseURL, tmdbID, apiKey)
		} else {
			endpoint = fmt.Sprintf("%s/movie/%d?api_key=%s&language=en-US", BaseURL, tmdbID, apiKey)
		}

		resp, err := c.httpClient.Get(endpoint)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var detail struct {
				Overview string `json:"overview"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&detail); err == nil && strings.TrimSpace(detail.Overview) != "" {
				res := scraper.CleanHTMLToPlainText(detail.Overview)
				synopsisCacheMu.Lock()
				synopsisCache[cacheKey] = res
				synopsisCacheMu.Unlock()
				return res
			}
		}
	}

	// Fallback: search by title
	if title != "" {
		var searchURL string
		if isTV {
			searchURL = fmt.Sprintf("%s/search/tv?api_key=%s&query=%s&language=en-US", BaseURL, apiKey, url.QueryEscape(title))
		} else {
			searchURL = fmt.Sprintf("%s/search/movie?api_key=%s&query=%s&language=en-US", BaseURL, apiKey, url.QueryEscape(title))
		}
		resp, err := c.httpClient.Get(searchURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var searchRes struct {
				Results []struct {
					Overview string `json:"overview"`
				} `json:"results"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&searchRes); err == nil && len(searchRes.Results) > 0 {
				if overview := strings.TrimSpace(searchRes.Results[0].Overview); overview != "" {
					res := scraper.CleanHTMLToPlainText(overview)
					synopsisCacheMu.Lock()
					synopsisCache[cacheKey] = res
					synopsisCacheMu.Unlock()
					return res
				}
			}
		}
	}

	return ""
}
