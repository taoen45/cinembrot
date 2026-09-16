package pipeline

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"cinembrot/config"
	"cinembrot/imageprocessor"
	"cinembrot/model"
	"cinembrot/provider/archive"
	"cinembrot/provider/embed"
	"cinembrot/provider/jikan"
	"cinembrot/provider/omdb"
	"cinembrot/provider/openmovies"
	"cinembrot/provider/subtitles"
	"cinembrot/provider/tmdb"
	"cinembrot/provider/yts"
	"cinembrot/scraper"
	"cinembrot/translator"
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

// ProcessMovieSynopses ensures the movie has both an Indonesian synopsis and English synopsis.
// If either is missing or if the synopsis is still English, it auto-translates seamlessly.
func ProcessMovieSynopses(movie *model.Movie) {
	if movie == nil {
		return
	}
	raw := strings.TrimSpace(movie.Synopsis)
	if raw == "" && strings.TrimSpace(movie.SynopsisEN) != "" {
		raw = strings.TrimSpace(movie.SynopsisEN)
	}
	if raw == "" {
		return
	}

	// Jika belum ada SynopsisEN atau Synopsis masih sama persis dengan raw (belum ter-bilingual)
	if strings.TrimSpace(movie.SynopsisEN) == "" || movie.Synopsis == movie.SynopsisEN {
		synID, synEN, err := translator.TranslateBilingual(raw)
		if err == nil {
			if synID != "" {
				movie.Synopsis = synID
			}
			if synEN != "" {
				movie.SynopsisEN = synEN
			}
		}
	}
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
			if tmdbMovie.SynopsisEN != "" {
				movie.SynopsisEN = tmdbMovie.SynopsisEN
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

	// 3. Auto-translate sinopsis dwibahasa (Indonesia & English)
	ProcessMovieSynopses(movie)

	// 4. Pastikan DownloadLinks selalu terisi otomatis (YTS download & candidates subtitle ID/EN)
	if len(movie.DownloadLinks) == 0 {
		if (movie.Type == "hollywood" || movie.Type == "movie") && p.ytsCli != nil {
			if ytsMovies, err := p.ytsCli.SearchMovies(movie.Title, 3, 1); err == nil && len(ytsMovies) > 0 {
				for _, ym := range ytsMovies {
					if ym.Year == movie.Year || movie.Year == 0 {
						movie.DownloadLinks = append(movie.DownloadLinks, ym.DownloadLinks...)
						break
					}
				}
			}
		}
		if len(movie.DownloadLinks) == 0 {
			movie.DownloadLinks = subtitles.GenerateSubtitleDownloadLinks(movie.Title, movie.Year)
		}
	}
}

// IngestAnime fetches top popular and/or latest seasonal anime from MyAnimeList/Jikan and TMDb and saves to DB
func (p *Pipeline) IngestAnime(category string, limit int, year ...int) (int, error) {
	startTime := time.Now()

	targetYear := 0
	if len(year) > 0 && year[0] > 0 {
		targetYear = year[0]
	}

	catLower := strings.ToLower(strings.TrimSpace(category))
	var allMovies []model.Movie
	seenTitles := make(map[string]bool)

	addUnique := func(movies []model.Movie) {
		for _, m := range movies {
			key := strings.ToLower(strings.TrimSpace(m.Title))
			if key != "" && !seenTitles[key] {
				seenTitles[key] = true
				allMovies = append(allMovies, m)
			}
		}
	}

	// 1. Scraping Anime berdasarkan Tahun Rilis dengan Auto-Discovery Seluruh Halaman
	if targetYear > 0 {
		if limit <= 0 {
			// Mode Otomatis: Cek berapa page yang ada lalu generate SEMUA page
			log.Printf("[ANIME] 🔍 Menghubungi provider untuk mendeteksi total halaman anime rilis tahun %d...\n", targetYear)
			firstBatch, lastPage, hasNext, err := p.jikanCli.FetchAnimeByYearWithPagination(targetYear, 25, 1)
			if err == nil && len(firstBatch) > 0 {
				if lastPage > 50 {
					lastPage = 50 // batas aman
				}
				log.Printf("[ANIME] 🎌 Terdeteksi total %d halaman anime di MyAnimeList untuk tahun %d. Men-generate SEMUA halaman secara otomatis...\n",
					lastPage, targetYear)
				addUnique(firstBatch)

				currentPage := 2
				for currentPage <= lastPage && (hasNext || currentPage <= lastPage) {
					log.Printf("[ANIME] 🎌 Mengambil halaman %d dari %d...\n", currentPage, lastPage)
					time.Sleep(350 * time.Millisecond)
					batch, lp, hn, fetchErr := p.jikanCli.FetchAnimeByYearWithPagination(targetYear, 25, currentPage)
					if fetchErr != nil || len(batch) == 0 {
						log.Printf("[WARN] Jikan page %d notice: %v\n", currentPage, fetchErr)
						break
					}
					addUnique(batch)
					if lp > lastPage && lp <= 50 {
						lastPage = lp
					}
					hasNext = hn
					currentPage++
				}
			} else {
				// Fallback ke TMDb Discover Anime
				log.Printf("[WARN] Jikan anime tahun %d tidak merespons. Menggunakan fallback TMDb...\n", targetYear)
				firstBatchTMDb, totalPages, tmdbErr := p.tmdbCli.DiscoverAnimeWithTotal(20, 1, targetYear, "popularity.desc")
				if tmdbErr == nil && len(firstBatchTMDb) > 0 {
					if totalPages > 50 {
						totalPages = 50
					}
					log.Printf("[ANIME] 🎌 Terdeteksi total %d halaman anime di TMDb untuk tahun %d. Men-generate SEMUA halaman secara otomatis...\n",
						totalPages, targetYear)
					addUnique(firstBatchTMDb)

					for pg := 2; pg <= totalPages; pg++ {
						log.Printf("[ANIME] 🎌 Mengambil TMDb halaman %d dari %d...\n", pg, totalPages)
						batch, _ := p.tmdbCli.DiscoverAnime(20, pg, targetYear, "popularity.desc")
						if len(batch) == 0 {
							break
						}
						addUnique(batch)
						time.Sleep(250 * time.Millisecond)
					}
				}
			}
		} else {
			// Jika limit ditentukan secara eksplisit, batasi sesuai limit
			log.Printf("[ANIME] 🎌 Mengambil %d anime yang rilis pada tahun %d...\n", limit, targetYear)
			got := 0
			page := 1
			for got < limit && page <= 15 {
				perPage := limit - got
				if perPage > 25 {
					perPage = 25
				}
				batch, err := p.jikanCli.FetchAnimeByYear(targetYear, perPage, page)
				if err != nil || len(batch) == 0 {
					batch, _ = p.tmdbCli.DiscoverAnime(perPage, page, targetYear, "popularity.desc")
				}
				if len(batch) == 0 {
					break
				}
				addUnique(batch)
				got += len(batch)
				page++
				time.Sleep(300 * time.Millisecond)
			}
		}
	} else {
		// Non-year: top atau seasonal
		if limit <= 0 {
			limit = 25
		}
		fetchBatch := func(subCat string, targetCount int) {
			got := 0
			page := 1
			for got < targetCount && page <= 10 {
				perPage := targetCount - got
				if perPage > 25 {
					perPage = 25
				}

				var batch []model.Movie
				var err error

				if subCat == "seasonal" || subCat == "now" || subCat == "latest" {
					batch, err = p.jikanCli.FetchSeasonalAnime(perPage, page)
				} else {
					batch, err = p.jikanCli.FetchTopAnime(perPage, page)
				}

				if err != nil || len(batch) == 0 {
					log.Printf("[WARN] Jikan %s page %d notice: %v. Menggunakan fallback TMDb...\n", subCat, page, err)
					sortOption := "popularity.desc"
					if subCat == "seasonal" || subCat == "now" || subCat == "latest" {
						sortOption = "first_air_date.desc"
					}
					batch, err = p.tmdbCli.DiscoverAnime(perPage, page, 0, sortOption)
				}

				if len(batch) == 0 {
					break
				}

				addUnique(batch)
				got += len(batch)
				page++
				time.Sleep(350 * time.Millisecond)
			}
		}

		if catLower == "all" || catLower == "combo" || catLower == "popular-latest" || limit >= 40 {
			half := limit / 2
			log.Printf("[ANIME] 🎌 Mengambil %d anime terpopuler sepanjang masa...\n", half)
			fetchBatch("top", half)
			log.Printf("[ANIME] 🎌 Mengambil %d anime terbaru & seasonal musim ini...\n", limit-half)
			fetchBatch("seasonal", limit-half)
		} else if catLower == "seasonal" || catLower == "now" || catLower == "latest" {
			fetchBatch("seasonal", limit)
		} else {
			fetchBatch("top", limit)
		}
	}

	if len(allMovies) == 0 {
		_ = p.repo.LogScrape("Anime-Scraper", fmt.Sprintf("category=%s&limit=%d", category, limit), "FAILED", 0, "tidak ada anime yang berhasil diambil", time.Since(startTime))
		return 0, fmt.Errorf("tidak ada anime yang berhasil diambil dari provider")
	}

	log.Printf("[ANIME] 🚀 Mulai memproses %d judul anime unik (enrich metadata + konversi WebP + subtitle)...\n", len(allMovies))

	savedCount := 0
	for i := range allMovies {
		movie := &allMovies[i]
		p.enrichMetadata(movie)

		// Download & Convert Images to WebP (Original & Thumbnail)
		imageprocessor.ProcessMovieImages(movie, p.cfg.ScraperUserAgent)

		// Generate multi-server streaming embed links if empty
		if len(movie.StreamLinks) == 0 {
			movie.StreamLinks = embed.GenerateMultiServerStreams(movie, 0, 1, 1)
		}

		if err := p.repo.UpsertMovie(movie); err != nil {
			log.Printf("[ERROR] Gagal menyimpan anime '%s': %v\n", movie.Title, err)
			continue
		}
		log.Printf("  -> [%d/%d] Saved Anime: '%s' (%d) [WebP: %t, Subtitle Links: %d]\n",
			savedCount+1, len(allMovies), movie.Title, movie.Year, strings.HasPrefix(movie.PosterURL, "/uploads/"), len(movie.DownloadLinks))
		savedCount++
	}

	_ = p.repo.LogScrape("Anime-Scraper", fmt.Sprintf("category=%s&total=%d", category, savedCount), "SUCCESS", savedCount, "", time.Since(startTime))
	return savedCount, nil
}

// IngestAsianDramas fetches top Asian dramas (Korean, Chinese, Japanese, Thai) from TMDb and saves to DB
func (p *Pipeline) IngestAsianDramas(lang string, pages int, year ...int) (int, error) {
	startTime := time.Now()
	targetPages := pages
	savedCount := 0

	// Jika pages <= 0, periksa total_pages di API lalu scrape SEMUA halaman
	if targetPages <= 0 {
		firstBatch, totalPages, err := p.tmdbCli.DiscoverAsianDramasWithTotal(lang, 1, year...)
		if err != nil {
			_ = p.repo.LogScrape("TMDb-TV", fmt.Sprintf("discover/tv?lang=%s&pages=all", lang), "FAILED", 0, err.Error(), time.Since(startTime))
			return 0, err
		}
		if totalPages <= 0 {
			totalPages = 1
		}
		if totalPages > 50 {
			totalPages = 50
		}
		targetPages = totalPages

		yearStr := "Semua Tahun"
		if len(year) > 0 && year[0] > 0 {
			yearStr = fmt.Sprintf("Tahun %d", year[0])
		}
		log.Printf("[DRAMA] 🔍 Terdeteksi total %d halaman untuk Drama '%s' (%s). Men-generate SEMUA halaman secara otomatis...\n",
			targetPages, lang, yearStr)

		// Simpan halaman 1
		for i := range firstBatch {
			drama := &firstBatch[i]
			p.enrichMetadata(drama)
			imageprocessor.ProcessMovieImages(drama, p.cfg.ScraperUserAgent)
			if len(drama.StreamLinks) == 0 {
				drama.StreamLinks = embed.GenerateMultiServerStreams(drama, 0, 1, 1)
			}
			if err := p.repo.UpsertMovie(drama); err != nil {
				continue
			}
			log.Printf("[SUCCESS] Saved Asian Drama: '%s' (%d) [WebP Ready, Links: %d]\n",
				drama.Title, drama.Year, len(drama.DownloadLinks))
			savedCount++
		}

		// Lanjutkan halaman 2 sampai targetPages
		for pg := 2; pg <= targetPages; pg++ {
			log.Printf("[DRAMA] 🎭 Mengambil halaman %d dari %d...\n", pg, targetPages)
			dramas, err := p.tmdbCli.DiscoverAsianDramas(lang, pg, year...)
			if err != nil {
				log.Printf("[WARN] Asian drama page %d notice: %v\n", pg, err)
				break
			}
			for i := range dramas {
				drama := &dramas[i]
				p.enrichMetadata(drama)
				imageprocessor.ProcessMovieImages(drama, p.cfg.ScraperUserAgent)
				if len(drama.StreamLinks) == 0 {
					drama.StreamLinks = embed.GenerateMultiServerStreams(drama, 0, 1, 1)
				}
				if err := p.repo.UpsertMovie(drama); err != nil {
					continue
				}
				log.Printf("[SUCCESS] Saved Asian Drama: '%s' (%d) [WebP Ready, Links: %d]\n",
					drama.Title, drama.Year, len(drama.DownloadLinks))
				savedCount++
			}
			time.Sleep(250 * time.Millisecond)
		}
	} else {
		for pg := 1; pg <= targetPages; pg++ {
			dramas, err := p.tmdbCli.DiscoverAsianDramas(lang, pg, year...)
			if err != nil {
				_ = p.repo.LogScrape("TMDb-TV", fmt.Sprintf("discover/tv?lang=%s&page=%d", lang, pg), "FAILED", savedCount, err.Error(), time.Since(startTime))
				return savedCount, err
			}

			for i := range dramas {
				drama := &dramas[i]
				p.enrichMetadata(drama)
				imageprocessor.ProcessMovieImages(drama, p.cfg.ScraperUserAgent)
				if len(drama.StreamLinks) == 0 {
					drama.StreamLinks = embed.GenerateMultiServerStreams(drama, 0, 1, 1)
				}
				if err := p.repo.UpsertMovie(drama); err != nil {
					log.Printf("[ERROR] Failed to upsert Asian drama '%s': %v\n", drama.Title, err)
					continue
				}
				log.Printf("[SUCCESS] Saved Asian Drama: '%s' (%d) [WebP Ready, Links: %d]\n",
					drama.Title, drama.Year, len(drama.DownloadLinks))
				savedCount++
			}
		}
	}

	_ = p.repo.LogScrape("TMDb-TV", fmt.Sprintf("discover/tv?lang=%s&pages=%d", lang, targetPages), "SUCCESS", savedCount, "", time.Since(startTime))
	return savedCount, nil
}

// IngestHollywoodMovies fetches top blockbuster Hollywood / Box Office movies from TMDb and saves to DB
func (p *Pipeline) IngestHollywoodMovies(category string, pages int, year ...int) (int, error) {
	startTime := time.Now()
	targetPages := pages
	savedCount := 0

	// Jika pages <= 0, periksa total_pages di API lalu scrape SEMUA halaman
	if targetPages <= 0 {
		firstBatch, totalPages, err := p.tmdbCli.DiscoverHollywoodMoviesWithTotal(category, 1, year...)
		if err != nil {
			_ = p.repo.LogScrape("TMDb-Hollywood", fmt.Sprintf("category=%s&pages=all", category), "FAILED", 0, err.Error(), time.Since(startTime))
			return 0, err
		}
		if totalPages <= 0 {
			totalPages = 1
		}
		if totalPages > 50 {
			totalPages = 50
		}
		targetPages = totalPages

		yearStr := "Semua Tahun"
		if len(year) > 0 && year[0] > 0 {
			yearStr = fmt.Sprintf("Tahun %d", year[0])
		}
		log.Printf("[HOLLYWOOD] 🔍 Terdeteksi total %d halaman untuk kategori '%s' (%s). Men-generate SEMUA halaman secara otomatis...\n",
			targetPages, category, yearStr)

		// Simpan halaman 1
		for i := range firstBatch {
			movie := &firstBatch[i]
			p.enrichMetadata(movie)
			imageprocessor.ProcessMovieImages(movie, p.cfg.ScraperUserAgent)
			if len(movie.StreamLinks) == 0 {
				movie.StreamLinks = embed.GenerateMultiServerStreams(movie, 0, 1, 1)
			}
			if err := p.repo.UpsertMovie(movie); err != nil {
				continue
			}
			log.Printf("[SUCCESS] Saved Hollywood Movie: '%s' (%d) [WebP Ready, Links: %d, Rating: %.1f]\n",
				movie.Title, movie.Year, len(movie.DownloadLinks), movie.Rating)
			savedCount++
		}

		// Lanjutkan halaman 2 sampai targetPages
		for pg := 2; pg <= targetPages; pg++ {
			log.Printf("[HOLLYWOOD] 🎬 Mengambil halaman %d dari %d...\n", pg, targetPages)
			movies, err := p.tmdbCli.DiscoverHollywoodMovies(category, pg, year...)
			if err != nil {
				log.Printf("[WARN] Hollywood page %d notice: %v\n", pg, err)
				break
			}
			for i := range movies {
				movie := &movies[i]
				p.enrichMetadata(movie)
				imageprocessor.ProcessMovieImages(movie, p.cfg.ScraperUserAgent)
				if len(movie.StreamLinks) == 0 {
					movie.StreamLinks = embed.GenerateMultiServerStreams(movie, 0, 1, 1)
				}
				if err := p.repo.UpsertMovie(movie); err != nil {
					continue
				}
				log.Printf("[SUCCESS] Saved Hollywood Movie: '%s' (%d) [WebP Ready, Links: %d, Rating: %.1f]\n",
					movie.Title, movie.Year, len(movie.DownloadLinks), movie.Rating)
				savedCount++
			}
			time.Sleep(250 * time.Millisecond)
		}
	} else {
		for pg := 1; pg <= targetPages; pg++ {
			movies, err := p.tmdbCli.DiscoverHollywoodMovies(category, pg, year...)
			if err != nil {
				_ = p.repo.LogScrape("TMDb-Hollywood", fmt.Sprintf("category=%s&page=%d", category, pg), "FAILED", savedCount, err.Error(), time.Since(startTime))
				return savedCount, err
			}

			for i := range movies {
				movie := &movies[i]
				p.enrichMetadata(movie)
				imageprocessor.ProcessMovieImages(movie, p.cfg.ScraperUserAgent)
				if len(movie.StreamLinks) == 0 {
					movie.StreamLinks = embed.GenerateMultiServerStreams(movie, 0, 1, 1)
				}
				if err := p.repo.UpsertMovie(movie); err != nil {
					log.Printf("[ERROR] Failed to upsert Hollywood movie '%s': %v\n", movie.Title, err)
					continue
				}
				log.Printf("[SUCCESS] Saved Hollywood Movie: '%s' (%d) [WebP Ready, Links: %d, Rating: %.1f]\n",
					movie.Title, movie.Year, len(movie.DownloadLinks), movie.Rating)
				savedCount++
			}
		}
	}

	_ = p.repo.LogScrape("TMDb-Hollywood", fmt.Sprintf("category=%s&pages=%d", category, targetPages), "SUCCESS", savedCount, "", time.Since(startTime))
	return savedCount, nil
}

// PopulateMissingStreamLinks updates existing movies/anime/drama in database that lack streaming links
func (p *Pipeline) PopulateMissingStreamLinks(db *gorm.DB) (int, error) {
	var movies []model.Movie
	if err := db.Preload("StreamLinks").Find(&movies).Error; err != nil {
		return 0, err
	}

	updatedCount := 0
	for i := range movies {
		movie := &movies[i]
		if len(movie.StreamLinks) == 0 {
			newStreams := embed.GenerateMultiServerStreams(movie, 0, 1, 1)
			if len(newStreams) > 0 {
				for _, st := range newStreams {
					st.MovieID = movie.ID
					_ = db.Create(&st).Error
				}
				updatedCount++
				log.Printf("  -> [%d] Ditambahkan %d server streaming untuk: '%s' (%s)\n",
					updatedCount, len(newStreams), movie.Title, movie.Type)
			}
		}
	}
	return updatedCount, nil
}

// FixNonLatinTitlesAndSynopses scans all records in MariaDB, converts non-Latin titles (Kanji/Hanzi/Kana)
// into official English QWERTY titles, updates slugs, and populates missing synopses from TMDb
func (p *Pipeline) FixNonLatinTitlesAndSynopses(db *gorm.DB) (int, error) {
	var movies []model.Movie
	if err := db.Find(&movies).Error; err != nil {
		return 0, err
	}

	log.Printf("[CLEANUP] Memeriksa %d judul di database untuk konversi judul Kanji -> English (QWERTY) dan pengisian sinopsis kosong...\n", len(movies))
	updatedCount := 0

	for i := range movies {
		movie := &movies[i]
		isNonLatin := scraper.ContainsNonLatin(movie.Title)
		isSynopsisEmpty := strings.TrimSpace(movie.Synopsis) == "" || strings.Contains(strings.ToLower(movie.Synopsis), "belum tersedia")

		if !isNonLatin && !isSynopsisEmpty {
			continue
		}

		updates := make(map[string]interface{})
		needUpdate := false

		// 1. Dapatkan TMDb ID dari SourceURL jika tersedia
		var tmdbID int
		if strings.Contains(movie.SourceURL, "themoviedb.org") {
			parts := strings.Split(movie.SourceURL, "/")
			for idx, part := range parts {
				if (part == "tv" || part == "movie") && idx+1 < len(parts) {
					if id, err := strconv.Atoi(parts[idx+1]); err == nil && id > 0 {
						tmdbID = id
						break
					}
				}
			}
		}

		isTV := movie.Type == "anime" || movie.Type == "drama_pendek" || strings.Contains(movie.SourceURL, "/tv/")

		var freshMovie *model.Movie
		var err error

		if tmdbID > 0 {
			if isTV {
				freshMovie, err = p.tmdbCli.GetTVDetails(tmdbID)
			} else {
				freshMovie, err = p.tmdbCli.GetMovieDetails(tmdbID)
			}
		}

		// Fallback jika belum berhasil: cari via SearchTV atau SearchMovie
		if freshMovie == nil || err != nil {
			searchQuery := movie.OriginalTitle
			if searchQuery == "" || scraper.ContainsNonLatin(searchQuery) {
				searchQuery = movie.Title
			}
			if isTV {
				if searchRes, sErr := p.tmdbCli.SearchTV(searchQuery); sErr == nil && len(searchRes.Results) > 0 {
					freshMovie, _ = p.tmdbCli.GetTVDetails(searchRes.Results[0].ID)
				}
			} else {
				if searchRes, sErr := p.tmdbCli.SearchMovie(searchQuery, movie.Year); sErr == nil && len(searchRes.Results) > 0 {
					freshMovie, _ = p.tmdbCli.GetMovieDetails(searchRes.Results[0].ID)
				}
			}
		}

		if freshMovie != nil {
			if isNonLatin && freshMovie.Title != "" && !scraper.ContainsNonLatin(freshMovie.Title) {
				updates["title"] = freshMovie.Title
				if freshMovie.Slug != "" {
					updates["slug"] = freshMovie.Slug
				}
				updates["alternative_titles"] = movie.Title
				if movie.OriginalTitle == "" {
					updates["original_title"] = movie.Title
				}
				needUpdate = true
			}

			if isSynopsisEmpty && freshMovie.Synopsis != "" {
				updates["synopsis"] = freshMovie.Synopsis
				needUpdate = true
			}
		} else if isSynopsisEmpty {
			enSynopsis := p.tmdbCli.GetEnglishSynopsis(movie.SourceURL, movie.Title, movie.Type)
			if enSynopsis != "" {
				updates["synopsis"] = enSynopsis
				needUpdate = true
			}
		}

		if needUpdate {
			if err := db.Model(movie).Updates(updates).Error; err == nil {
				updatedCount++
				log.Printf("  -> [FIX %d] '%s' -> Title: '%v', Slug: '%v', Sinopsis: %d karakter\n",
					updatedCount, movie.Title, updates["title"], updates["slug"], len(fmt.Sprintf("%v", updates["synopsis"])))
			}
		}
		time.Sleep(80 * time.Millisecond) // Rate limiting
	}

	return updatedCount, nil
}

// TranslateAllExistingSynopses sweeps all movies in the database and ensures both
// Indonesian (synopsis) and English (synopsis_en) versions exist.
func TranslateAllExistingSynopses(db *gorm.DB) (int, error) {
	var movies []model.Movie
	// Ambil film yang synopsis_en nya kosong atau synopsis nya belum diterjemahkan
	err := db.Where("synopsis <> '' AND (synopsis_en IS NULL OR synopsis_en = '' OR synopsis = synopsis_en)").
		Order("id desc").
		Find(&movies).Error
	if err != nil {
		return 0, err
	}

	if len(movies) == 0 {
		log.Println("[TRANSLATE] Semua sinopsis film di database sudah memiliki versi dwibahasa (ID & EN) lengkap!")
		return 0, nil
	}

	log.Printf("[TRANSLATE] 🌐 Ditemukan %d judul film yang membutuhkan sinkronisasi sinopsis dwibahasa...\n", len(movies))
	updatedCount := 0

	for i := range movies {
		movie := &movies[i]
		raw := strings.TrimSpace(movie.Synopsis)
		if raw == "" && strings.TrimSpace(movie.SynopsisEN) != "" {
			raw = strings.TrimSpace(movie.SynopsisEN)
		}
		if raw == "" {
			continue
		}

		synID, synEN, err := translator.TranslateBilingual(raw)
		if err != nil || synID == "" {
			continue
		}

		updates := map[string]interface{}{
			"synopsis":    synID,
			"synopsis_en": synEN,
		}

		if err := db.Model(movie).Updates(updates).Error; err == nil {
			updatedCount++
			log.Printf("  -> [%d/%d] 🌐 Sukses Terjemahkan '%s' (%d) [ID: %d char, EN: %d char]\n",
				updatedCount, len(movies), movie.Title, movie.Year, len(synID), len(synEN))
		}

		time.Sleep(100 * time.Millisecond) // Rate limiting ramah
	}

	log.Printf("[TRANSLATE] ✅ Selesai! Sebanyak %d sinopsis film berhasil disinkronkan dwibahasa (ID & EN).\n", updatedCount)
	return updatedCount, nil
}

// PopulateAllExistingDownloadLinks sweeps all movies in the database that don't have download links
// and generates direct torrent/mirror/subtitle download links for them.
func PopulateAllExistingDownloadLinks(db *gorm.DB, ytsCli *yts.Client) (int, error) {
	var movies []model.Movie
	// Ambil film yang belum memiliki download link
	err := db.Preload("DownloadLinks").
		Order("id desc").
		Find(&movies).Error
	if err != nil {
		return 0, err
	}

	log.Printf("[POPULATE-DOWNLOADS] 📥 Memeriksa kelengkapan URL link download untuk %d judul film di database...\n", len(movies))
	populatedCount := 0

	for i := range movies {
		movie := &movies[i]
		if len(movie.DownloadLinks) > 0 {
			continue
		}

		var newLinks []model.DownloadLink

		// Jika film Hollywood / Bioskop: coba cari torrent dari YTS
		if (movie.Type == "hollywood" || movie.Type == "movie") && ytsCli != nil {
			if ytsMovies, err := ytsCli.SearchMovies(movie.Title, 3, 1); err == nil && len(ytsMovies) > 0 {
				for _, ym := range ytsMovies {
					if ym.Year == movie.Year || movie.Year == 0 {
						for _, dl := range ym.DownloadLinks {
							dl.MovieID = movie.ID
							newLinks = append(newLinks, dl)
						}
						break
					}
				}
			}
		}

		// Jika belum ada, tambahkan link unduhan subtitle dwibahasa resmi
		if len(newLinks) == 0 {
			subLinks := subtitles.GenerateSubtitleDownloadLinks(movie.Title, movie.Year)
			for _, sl := range subLinks {
				sl.MovieID = movie.ID
				newLinks = append(newLinks, sl)
			}
		}

		if len(newLinks) > 0 {
			for j := range newLinks {
				db.Create(&newLinks[j])
			}
			populatedCount++
			log.Printf("  -> [%d] 📥 Sukses Menambahkan %d Link Download untuk '%s' (%d)\n",
				populatedCount, len(newLinks), movie.Title, movie.Year)
		}

		time.Sleep(50 * time.Millisecond) // Rate limiting
	}

	log.Printf("[POPULATE-DOWNLOADS] ✅ Selesai! Berhasil melengkapi URL link download untuk %d film di database.\n", populatedCount)
	return populatedCount, nil
}

