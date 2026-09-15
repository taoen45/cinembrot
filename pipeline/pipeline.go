package pipeline

import (
	"fmt"
	"log"
	"strings"
	"time"

	"cinembrot/config"
	"cinembrot/imageprocessor"
	"cinembrot/model"
	"cinembrot/provider/archive"
	"cinembrot/provider/jikan"
	"cinembrot/provider/omdb"
	"cinembrot/provider/openmovies"
	"cinembrot/provider/tmdb"
	"cinembrot/provider/yts"
	"cinembrot/scraper"
	"cinembrot/validator"

	"gorm.io/gorm"
)

type Pipeline struct {
	cfg        *config.Config
	repo       *scraper.Repository
	archiveCli *archive.Client
	jikanCli   *jikan.Client
	tmdbCli    *tmdb.Client
	omdbCli    *omdb.Client
	ytsCli     *yts.Client
}

func NewPipeline(cfg *config.Config, repo *scraper.Repository) *Pipeline {
	return &Pipeline{
		cfg:        cfg,
		repo:       repo,
		archiveCli: archive.NewClient(cfg),
		jikanCli:   jikan.NewClient(cfg),
		tmdbCli:    tmdb.NewClient(cfg),
		omdbCli:    omdb.NewClient(cfg),
		ytsCli:     yts.NewClient(cfg),
	}
}

// IngestOpenMovies inserts curated Creative Commons / Open source films with full 4K downloads
func (p *Pipeline) IngestOpenMovies() (int, error) {
	startTime := time.Now()
	movies := openmovies.GetCuratedOpenMovies()
	savedCount := 0

	for i := range movies {
		movie := &movies[i]
		p.enrichMetadata(movie)

		// Download & Convert Images to WebP (Original & Thumbnail)
		imageprocessor.ProcessMovieImages(movie, p.cfg.ScraperUserAgent)

		// Validate Download Link Health & Validity
		if len(movie.DownloadLinks) > 0 {
			movie.DownloadLinks = validator.ValidateMovieDownloadLinks(movie.DownloadLinks, p.cfg.ScraperUserAgent)
		}

		if err := p.repo.UpsertMovie(movie); err != nil {
			log.Printf("[ERROR] Failed to save open movie '%s': %v\n", movie.Title, err)
			continue
		}
		log.Printf("[SUCCESS] Ingested Open Movie: '%s' (%d) [WebP Ready, Links: %d]\n",
			movie.Title, movie.Year, len(movie.DownloadLinks))
		savedCount++
	}

	_ = p.repo.LogScrape("BlenderOpenMovies", "https://studio.blender.org/films/", "SUCCESS", savedCount, "", time.Since(startTime))
	return savedCount, nil
}

// IngestArchiveFeatureFilms fetches top public domain feature films from Internet Archive and saves to MariaDB
func (p *Pipeline) IngestArchiveFeatureFilms(limit int, page int) (int, error) {
	startTime := time.Now()
	movies, err := p.archiveCli.FetchFeatureFilms(limit, page)
	if err != nil {
		_ = p.repo.LogScrape("Archive.org", "https://archive.org/details/feature_films", "FAILED", 0, err.Error(), time.Since(startTime))
		return 0, err
	}

	savedCount := 0
	for i := range movies {
		movie := &movies[i]
		p.enrichMetadata(movie)

		// Download & Convert Images to WebP (Original & Thumbnail)
		imageprocessor.ProcessMovieImages(movie, p.cfg.ScraperUserAgent)

		// Validate Download Link Health & Validity
		if len(movie.DownloadLinks) > 0 {
			movie.DownloadLinks = validator.ValidateMovieDownloadLinks(movie.DownloadLinks, p.cfg.ScraperUserAgent)
		}

		if err := p.repo.UpsertMovie(movie); err != nil {
			log.Printf("[ERROR] Failed to upsert movie '%s': %v\n", movie.Title, err)
			continue
		}
		log.Printf("[SUCCESS] Saved Archive.org movie: '%s' (%d) [WebP Ready, Links: %d]\n",
			movie.Title, movie.Year, len(movie.DownloadLinks))
		savedCount++
	}

	_ = p.repo.LogScrape("Archive.org", "https://archive.org/details/feature_films", "SUCCESS", savedCount, "", time.Since(startTime))
	return savedCount, nil
}

// SearchAndIngestTMDb searches and stores movie data directly from TMDb REST API
func (p *Pipeline) SearchAndIngestTMDb(query string, year int) (*model.Movie, error) {
	searchRes, err := p.tmdbCli.SearchMovie(query, year)
	if err != nil {
		return nil, fmt.Errorf("TMDb search failed: %w", err)
	}

	if len(searchRes.Results) == 0 {
		return nil, fmt.Errorf("no movie found matching '%s' on TMDb", query)
	}

	tmdbID := searchRes.Results[0].ID
	movie, err := p.tmdbCli.GetMovieDetails(tmdbID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch TMDb movie details: %w", err)
	}

	// Enrich with OMDb ratings
	_ = p.omdbCli.EnrichMovie(movie)

	// Download & Convert Images to WebP (Original & Thumbnail)
	imageprocessor.ProcessMovieImages(movie, p.cfg.ScraperUserAgent)

	// Validate Download Link Health & Validity
	if len(movie.DownloadLinks) > 0 {
		movie.DownloadLinks = validator.ValidateMovieDownloadLinks(movie.DownloadLinks, p.cfg.ScraperUserAgent)
	}

	if err := p.repo.UpsertMovie(movie); err != nil {
		return nil, fmt.Errorf("failed to save TMDb movie to MariaDB: %w", err)
	}

	log.Printf("[SUCCESS] Ingested TMDb movie: '%s' (TMDb ID: %d) [WebP Ready] into MariaDB!\n", movie.Title, tmdbID)
	return movie, nil
}

// IngestByYear discovers and scrapes movies for a specific release year
func (p *Pipeline) IngestByYear(year int, pages int, source string) (int, error) {
	startTime := time.Now()
	totalSaved := 0

	if pages <= 0 {
		pages = 1
	}

	// 1. Scraping from TMDb Discover API
	if source == "tmdb" || source == "all" {
		for page := 1; page <= pages; page++ {
			pageStart := time.Now()
			pageSaved := 0
			log.Printf("[INFO] Scraping TMDb Discover year %d (Page %d/%d)...\n", year, page, pages)
			movies, err := p.tmdbCli.DiscoverMoviesByYear(year, page)
			if err != nil {
				log.Printf("[WARN] Failed to fetch TMDb page %d for year %d: %v\n", page, year, err)
				continue
			}

			for i := range movies {
				movie := &movies[i]
				_ = p.omdbCli.EnrichMovie(movie)

				// Download & Convert Images to WebP (Original & Thumbnail)
				imageprocessor.ProcessMovieImages(movie, p.cfg.ScraperUserAgent)

				// Validate Download Link Health & Validity
				if len(movie.DownloadLinks) > 0 {
					movie.DownloadLinks = validator.ValidateMovieDownloadLinks(movie.DownloadLinks, p.cfg.ScraperUserAgent)
				}

				if err := p.repo.UpsertMovie(movie); err == nil {
					log.Printf("  -> [TMDb %d] Saved: '%s' (Rating: %.1f, WebP: %t)\n",
						year, movie.Title, movie.Rating, strings.HasPrefix(movie.PosterURL, "/uploads/"))
					totalSaved++
					pageSaved++
				}
				time.Sleep(time.Duration(p.cfg.ScraperDelayMs) * time.Millisecond)
			}

			if pages > 1 {
				_ = p.repo.LogScrape(fmt.Sprintf("YearScraper-%d-tmdb", year),
					fmt.Sprintf("year=%d&page=%d/%d", year, page, pages), "SUCCESS", pageSaved, "", time.Since(pageStart))
			}
		}
	}

	// 2. Scraping legal movies from Archive.org
	if source == "archive" || source == "all" {
		for page := 1; page <= pages; page++ {
			pageStart := time.Now()
			pageSaved := 0
			log.Printf("[INFO] Scraping Archive.org feature films year %d (Page %d/%d)...\n", year, page, pages)
			movies, err := p.archiveCli.FetchFeatureFilmsByYear(year, 15, page)
			if err != nil {
				log.Printf("[WARN] Failed to fetch Archive.org page %d for year %d: %v\n", page, year, err)
				continue
			}

			for i := range movies {
				movie := &movies[i]
				p.enrichMetadata(movie)

				// Download & Convert Images to WebP (Original & Thumbnail)
				imageprocessor.ProcessMovieImages(movie, p.cfg.ScraperUserAgent)

				// Validate Download Link Health & Validity
				if len(movie.DownloadLinks) > 0 {
					movie.DownloadLinks = validator.ValidateMovieDownloadLinks(movie.DownloadLinks, p.cfg.ScraperUserAgent)
				}

				if err := p.repo.UpsertMovie(movie); err == nil {
					log.Printf("  -> [Archive %d] Saved: '%s' (Downloads: %d, WebP: %t)\n",
						year, movie.Title, movie.Views, strings.HasPrefix(movie.PosterURL, "/uploads/"))
					totalSaved++
					pageSaved++
				}
				time.Sleep(time.Duration(p.cfg.ScraperDelayMs) * time.Millisecond)
			}

			if pages > 1 {
				_ = p.repo.LogScrape(fmt.Sprintf("YearScraper-%d-archive", year),
					fmt.Sprintf("year=%d&page=%d/%d", year, page, pages), "SUCCESS", pageSaved, "", time.Since(pageStart))
			}
		}
	}

	var lastErr error

	// 3. Scraping movies & torrent downloads from YTS REST API
	if source == "yts" || source == "all" {
		for page := 1; page <= pages; page++ {
			pageStart := time.Now()
			pageSaved := 0
			log.Printf("[INFO] Scraping YTS Movies year %d (Page %d/%d)...\n", year, page, pages)
			movies, err := p.ytsCli.FetchMoviesByYear(year, 20, page)
			if err != nil {
				log.Printf("[WARN] Failed to fetch YTS page %d for year %d: %v\n", page, year, err)
				lastErr = err
				continue
			}

			for i := range movies {
				movie := &movies[i]
				p.enrichMetadata(movie)

				// Download & Convert Images to WebP (Original & Thumbnail)
				imageprocessor.ProcessMovieImages(movie, p.cfg.ScraperUserAgent)

				if err := p.repo.UpsertMovie(movie); err == nil {
					log.Printf("  -> [YTS %d] Saved: '%s' (Rating: %.1f, Torrents: %d, WebP: %t)\n",
						year, movie.Title, movie.Rating, len(movie.DownloadLinks), strings.HasPrefix(movie.PosterURL, "/uploads/"))
					totalSaved++
					pageSaved++
				}
				time.Sleep(time.Duration(p.cfg.ScraperDelayMs) * time.Millisecond)
			}

			if pages > 1 {
				_ = p.repo.LogScrape(fmt.Sprintf("YearScraper-%d-yts", year),
					fmt.Sprintf("year=%d&page=%d/%d", year, page, pages), "SUCCESS", pageSaved, "", time.Since(pageStart))
			}
		}
	}

	status := "SUCCESS"
	errMsg := ""
	if totalSaved == 0 && lastErr != nil {
		status = "FAILED"
		errMsg = lastErr.Error()
	}

	_ = p.repo.LogScrape(fmt.Sprintf("YearScraper-%d-%s", year, source),
		fmt.Sprintf("year=%d&pages=%d", year, pages), status, totalSaved, errMsg, time.Since(startTime))

	return totalSaved, lastErr
}

// ConvertExistingImagesToWebP processes and converts movies already in MariaDB that have remote HTTP image URLs
func ConvertExistingImagesToWebP(db *gorm.DB, userAgent string, limit int) int {
	var movies []model.Movie
	db.Where("poster_url LIKE 'http%' OR backdrop_url LIKE 'http%'").Limit(limit).Find(&movies)

	if len(movies) == 0 {
		return 0
	}

	log.Printf("[WEBP] Mengonversi %d gambar film yang ada di database ke WebP lokal (Original + Thumbnail)...\n", len(movies))
	converted := 0

	for i := range movies {
		movie := &movies[i]
		imageprocessor.ProcessMovieImages(movie, userAgent)

		// Update database record
		db.Model(movie).Updates(map[string]interface{}{
			"poster_url":         movie.PosterURL,
			"poster_thumb_url":   movie.PosterThumbURL,
			"backdrop_url":       movie.BackdropURL,
			"backdrop_thumb_url": movie.BackdropThumbURL,
			"thumbnail_url":      movie.ThumbnailURL,
		})
		converted++
		log.Printf("  -> [WebP %d/%d] Selesai: '%s' -> Poster: %s, Thumb: %s\n",
			converted, len(movies), movie.Title, movie.PosterURL, movie.PosterThumbURL)
	}

	return converted
}

// enrichMetadata tries to query TMDb and OMDb to enhance metadata automatically
func (p *Pipeline) enrichMetadata(movie *model.Movie) {
	// 1. Try TMDb enrichment
	if searchRes, err := p.tmdbCli.SearchMovie(movie.Title, movie.Year); err == nil && len(searchRes.Results) > 0 {
		tmdbMovie, err := p.tmdbCli.GetMovieDetails(searchRes.Results[0].ID)
		if err == nil && tmdbMovie != nil {
			if tmdbMovie.PosterURL != "" {
				movie.PosterURL = tmdbMovie.PosterURL
			}
			if tmdbMovie.BackdropURL != "" {
				movie.BackdropURL = tmdbMovie.BackdropURL
			}
			if tmdbMovie.TrailerURL != "" {
				movie.TrailerURL = tmdbMovie.TrailerURL
			}
			if tmdbMovie.Synopsis != "" && len(tmdbMovie.Synopsis) > len(movie.Synopsis) {
				movie.Synopsis = tmdbMovie.Synopsis
			}
			if len(tmdbMovie.Genres) > 0 {
				movie.Genres = tmdbMovie.Genres
			}
			if len(tmdbMovie.Actors) > 0 {
				movie.Actors = tmdbMovie.Actors
			}
			if len(tmdbMovie.Directors) > 0 {
				movie.Directors = tmdbMovie.Directors
			}
			movie.TMDbRating = tmdbMovie.TMDbRating
		}
	}

	// 2. Try OMDb enrichment
	_ = p.omdbCli.EnrichMovie(movie)
}

// IngestAnime fetches top or seasonal anime from MyAnimeList via Jikan API and saves to DB
func (p *Pipeline) IngestAnime(category string, limit int) (int, error) {
	startTime := time.Now()
	if limit <= 0 {
		limit = 15
	}

	var movies []model.Movie
	var err error

	if category == "seasonal" || category == "now" {
		movies, err = p.jikanCli.FetchSeasonalAnime(limit, 1)
	} else {
		movies, err = p.jikanCli.FetchTopAnime(limit, 1)
	}

	if err != nil {
		_ = p.repo.LogScrape("Jikan-MyAnimeList", "https://api.jikan.moe/v4/", "FAILED", 0, err.Error(), time.Since(startTime))
		return 0, err
	}

	savedCount := 0
	for i := range movies {
		movie := &movies[i]
		p.enrichMetadata(movie)

		// Download & Convert Images to WebP (Original & Thumbnail)
		imageprocessor.ProcessMovieImages(movie, p.cfg.ScraperUserAgent)

		if err := p.repo.UpsertMovie(movie); err != nil {
			log.Printf("[ERROR] Failed to upsert anime '%s': %v\n", movie.Title, err)
			continue
		}
		log.Printf("[SUCCESS] Saved Anime: '%s' (%d) [WebP Ready, Links: %d]\n",
			movie.Title, movie.Year, len(movie.DownloadLinks))
		savedCount++
	}

	_ = p.repo.LogScrape("Jikan-MyAnimeList", fmt.Sprintf("category=%s&limit=%d", category, limit), "SUCCESS", savedCount, "", time.Since(startTime))
	return savedCount, nil
}

// IngestAsianDramas fetches top Asian dramas (Korean, Chinese, Japanese, Thai) from TMDb and saves to DB
func (p *Pipeline) IngestAsianDramas(lang string, page int) (int, error) {
	startTime := time.Now()
	if page <= 0 {
		page = 1
	}

	dramas, err := p.tmdbCli.DiscoverAsianDramas(lang, page)
	if err != nil {
		_ = p.repo.LogScrape("TMDb-TV", fmt.Sprintf("discover/tv?lang=%s", lang), "FAILED", 0, err.Error(), time.Since(startTime))
		return 0, err
	}

	savedCount := 0
	for i := range dramas {
		drama := &dramas[i]
		p.enrichMetadata(drama)

		// Download & Convert Images to WebP (Original & Thumbnail)
		imageprocessor.ProcessMovieImages(drama, p.cfg.ScraperUserAgent)

		if err := p.repo.UpsertMovie(drama); err != nil {
			log.Printf("[ERROR] Failed to upsert Asian drama '%s': %v\n", drama.Title, err)
			continue
		}
		log.Printf("[SUCCESS] Saved Asian Drama: '%s' (%d) [WebP Ready, Links: %d]\n",
			drama.Title, drama.Year, len(drama.DownloadLinks))
		savedCount++
	}

	_ = p.repo.LogScrape("TMDb-TV", fmt.Sprintf("discover/tv?lang=%s&page=%d", lang, page), "SUCCESS", savedCount, "", time.Since(startTime))
	return savedCount, nil
}
