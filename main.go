package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cinembrot/config"
	"cinembrot/database"
	"cinembrot/model"
	"cinembrot/pipeline"
	"cinembrot/provider/yts"
	"cinembrot/scheduler"
	"cinembrot/scraper"
	"cinembrot/server"
	"cinembrot/validator"

	"gorm.io/gorm"
)

func main() {
	serveWeb := flag.Bool("serve", false, "Start CINEMBROT Web Server on http://localhost:8080 (with background auto-scraper)")
	autoScrapeOnly := flag.Bool("auto-scrape", false, "Run full automated polite scraping cycle across all years")
	cronDaemon := flag.Bool("daemon", false, "Run standalone background scraping scheduler without web server")
	scrapeURL := flag.String("url", "", "Scrape custom website URL using Colly engine")
	fetchArchive := flag.Bool("archive", false, "Ingest legal public domain feature films from Internet Archive API")
	archiveLimit := flag.Int("archive-limit", 10, "Number of films to fetch from Internet Archive")
	archivePage := flag.Int("archive-page", 1, "Page number for Internet Archive search")
	fetchOpenMovies := flag.Bool("openmovies", false, "Ingest Creative Commons / Blender Open Movies")
	tmdbQuery := flag.String("tmdb", "", "Search & ingest rich metadata from TMDb REST API")
	tmdbYear := flag.Int("year", 0, "Optional release year filter for single TMDb search")
	byYear := flag.Int("by-year", 0, "Automatically scrape and discover top movies by release year (e.g. 2024, 2023, 1999)")
	pages := flag.Int("pages", 1, "Number of pages to scrape for year discovery (1 page = 20 movies)")
	source := flag.String("source", "tmdb", "Source for year discovery: 'tmdb', 'archive', 'yts', or 'all'")
	syncAll := flag.Bool("sync", false, "Run full pipeline (Open Movies + Internet Archive)")
	convertImages := flag.Bool("convert-images", false, "Download and convert existing movie images in database to local WebP")
	checkLinks := flag.Bool("check-links", false, "Scan and validate all download file links in database for broken/dead URLs")

	// CLI Scraper Khusus Anime & Drama Asia
	scrapeAnime := flag.Bool("scrape-anime", false, "Scrape anime resmi dari MyAnimeList via Jikan API & TMDb (WebP + Subtitle)")
	animeCat := flag.String("anime-cat", "top", "Kategori anime: 'top' (terpopuler) atau 'seasonal' (musim ini/on-going)")
	animeLimit := flag.Int("anime-limit", 0, "Jumlah judul anime yang diambil (0 = otomatis ikuti seluruh halaman)")
	animePages := flag.Int("anime-pages", 0, "Jumlah halaman anime yang diambil (0 = otomatis scrape SEMUA halaman yang ada)")

	scrapeDrama := flag.Bool("scrape-drama", false, "Scrape serial drama Asia dari TMDb TV API (WebP + Subtitle)")
	dramaLang := flag.String("drama-lang", "ko", "Bahasa drama: 'ko' (Korea), 'zh' (China), 'ja' (Jepang), 'th' (Thailand), atau 'all'")
	dramaPages := flag.Int("drama-pages", 0, "Jumlah halaman drama yang diambil (0 = otomatis scrape SEMUA halaman yang ada)")

	// CLI Scraper Khusus Film Hollywood / Box Office
	scrapeHollywood := flag.Bool("scrape-hollywood", false, "Scrape film Hollywood / Box Office resmi dari TMDb API (WebP + Subtitle + Multi-Server Stream)")
	hollywoodCat := flag.String("hollywood-cat", "boxoffice", "Kategori Hollywood: 'boxoffice' (pendapatan tertinggi), 'popular' (terpopuler), 'top_rated' (rating tertinggi), 'now_playing' (rilis bioskop terbaru)")
	hollywoodPages := flag.Int("hollywood-pages", 0, "Jumlah halaman Hollywood yang diambil (0 = otomatis scrape SEMUA halaman yang ada)")

	// CLI Sinkronisasi Multi-Server Streaming Video
	populateStreams := flag.Bool("populate-streams", false, "Isi dan perbarui server streaming embed (VidSrc, AutoEmbed, 2Embed, VidLink) untuk semua judul di database")

	// CLI Perbaikan Judul Non-Latin & Sinopsis Kosong
	fixTitles := flag.Bool("fix-titles", false, "Perbaiki judul-judul kanji/non-Latin ke English QWERTY dan lengkapi sinopsis kosong di database")

	// CLI Sinkronisasi Terjemahan Sinopsis Dwibahasa (ID & EN)
	translateSynopsis := flag.Bool("translate-synopsis", false, "Terjemahkan seluruh sinopsis film di database ke Bahasa Indonesia (ID) & English (EN)")

	// CLI Sinkronisasi Link Download File / Subtitle
	populateDownloads := flag.Bool("populate-downloads", false, "Isi dan perbarui URL link download secara massal untuk semua film di database")

	// CLI Scheduler Harian Lengkap (05:00 & 17:00 WIB)
	dailyScrape := flag.Bool("daily-scrape", false, "Jalankan 1 siklus lengkap scraper harian (Anime On-Going, Drama Asia, Hollywood, TMDb, YTS) untuk tahun sekarang")

	flag.Parse()

	fmt.Println("================================================================")
	fmt.Println("             CINEMBROT - GOLANG MOVIE ENGINE & WEB               ")
	fmt.Println("================================================================")

	// 1. Load Configurations
	cfg := config.LoadConfig()
	fmt.Printf("[CONFIG] Target MariaDB : %s@tcp(%s:%s)/%s\n", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)

	// 2. Initialize MariaDB Connection & Auto-Migrate Schemas
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("[FATAL] MariaDB Connection Failed: %v\n", err)
	}

	// 3. Initialize Repositories, Pipelines, and Auto-Scraper
	repo := scraper.NewRepository(db)
	engine := scraper.NewEngine(cfg, repo)
	pipe := pipeline.NewPipeline(cfg, repo)
	sampleScraper := scraper.NewSampleMovieScraper(engine)
	autoScraper := scheduler.NewAutoScraper(cfg, db, pipe, repo)

	// 4. Handle Dedicated Auto-Scraping Flags
	if *autoScrapeOnly {
		fmt.Println("\n[ACTION] Running one full polite auto-scraping cycle across years...")
		autoScraper.RunFullCycle()
		printDatabaseStats(db)
		return
	}

	if *cronDaemon {
		fmt.Println("\n[ACTION] Running standalone background scheduler daemon (Ctrl+C to stop)...")
		autoScraper.Start()
		// Wait for terminate signal
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		fmt.Println("\n[INFO] Scheduler shutting down...")
		return
	}

	// 5. Handle CLI Action Flags
	if *scrapeURL != "" {
		fmt.Printf("\n[ACTION] Starting HTML Scraper for: %s\n", *scrapeURL)
		if err := sampleScraper.ScrapeURL(*scrapeURL); err != nil {
			log.Printf("[WARN] Scraper notice: %v\n", err)
		}
		printDatabaseStats(db)
		return
	}

	if *byYear > 0 {
		fmt.Printf("\n[ACTION] Automated Year-Based Scraping for Year: %d (Pages: %d, Source: %s)...\n",
			*byYear, *pages, *source)
		count, err := pipe.IngestByYear(*byYear, *pages, *source)
		if err != nil {
			log.Printf("[ERROR] Year scraping error: %v\n", err)
		} else {
			fmt.Printf("\n[SUCCESS] Successfully scraped and saved %d movies for year %d into MariaDB!\n", count, *byYear)
		}
		printDatabaseStats(db)
		return
	}

	if *tmdbQuery != "" {
		fmt.Printf("\n[ACTION] Querying TMDb REST API for: '%s' (Year: %d)...\n", *tmdbQuery, *tmdbYear)
		movie, err := pipe.SearchAndIngestTMDb(*tmdbQuery, *tmdbYear)
		if err != nil {
			log.Printf("[ERROR] TMDb Ingestion failed: %v\n", err)
		} else {
			fmt.Printf("[SUCCESS] Ingested '%s' (%d) | Rating: %.1f | Genres: %d | Actors: %d\n",
				movie.Title, movie.Year, movie.Rating, len(movie.Genres), len(movie.Actors))
		}
		printDatabaseStats(db)
		return
	}

	if *fetchOpenMovies || *syncAll {
		fmt.Println("\n[ACTION] Ingesting Creative Commons / Blender Open Movies...")
		count, err := pipe.IngestOpenMovies()
		if err == nil {
			fmt.Printf("[SUCCESS] Ingested %d Creative Commons films!\n", count)
		}
	}

	if *fetchArchive || *syncAll {
		fmt.Printf("\n[ACTION] Ingesting %d Feature Films from Internet Archive (Page %d)...\n", *archiveLimit, *archivePage)
		count, err := pipe.IngestArchiveFeatureFilms(*archiveLimit, *archivePage)
		if err == nil {
			fmt.Printf("[SUCCESS] Ingested %d feature films from Internet Archive!\n", count)
		}
	}

	if *convertImages {
		fmt.Println("\n[ACTION] Mengonversi gambar film di database ke format WebP lokal (Original + Thumbnail)...")
		count := pipeline.ConvertExistingImagesToWebP(db, cfg.ScraperUserAgent, 500)
		fmt.Printf("\n[SUCCESS] Berhasil mengonversi %d film ke WebP lokal!\n", count)
		printDatabaseStats(db)
		return
	}

	if *checkLinks {
		fmt.Println("\n[ACTION] Memulai validasi dan pengecekan kesehatan link download file di database...")
		total, valid, dead := validator.ValidateAllDatabaseLinks(db, cfg.ScraperUserAgent, 0)
		fmt.Printf("\n[RESULT] Selesai! Total Diperiksa: %d | Aktif (Valid): %d | Rusak/Mati (Dead): %d\n", total, valid, dead)
		printDatabaseStats(db)
		return
	}

	if *scrapeAnime {
		yearInfo := ""
		if *tmdbYear > 0 {
			yearInfo = fmt.Sprintf(", Tahun: %d", *tmdbYear)
		}
		pageInfo := "Semua Halaman (Otomatis)"
		effectiveLimit := *animeLimit
		if *animePages > 0 {
			pageInfo = fmt.Sprintf("%d Halaman", *animePages)
			if effectiveLimit <= 0 {
				effectiveLimit = *animePages * 25
			}
		} else if effectiveLimit > 0 {
			pageInfo = fmt.Sprintf("Limit %d judul", effectiveLimit)
		}
		fmt.Printf("\n[ACTION] 🎌 Memulai scraping Anime resmi (Kategori: %s, Halaman: %s%s)...\n", *animeCat, pageInfo, yearInfo)
		count, err := pipe.IngestAnime(*animeCat, effectiveLimit, *tmdbYear)
		if err != nil {
			log.Printf("[ERROR] Scraping Anime gagal: %v\n", err)
		} else {
			fmt.Printf("\n[SUCCESS] Berhasil scrape dan simpan %d anime (WebP + Subtitle) ke MariaDB!\n", count)
		}
		printDatabaseStats(db)
		return
	}

	if *scrapeDrama {
		yearInfo := ""
		if *tmdbYear > 0 {
			yearInfo = fmt.Sprintf(", Tahun: %d", *tmdbYear)
		}
		pageInfo := "Semua Halaman (Otomatis)"
		if *dramaPages > 0 {
			pageInfo = fmt.Sprintf("%d Halaman", *dramaPages)
		}
		fmt.Printf("\n[ACTION] 🎭 Memulai scraping Drama Asia resmi TMDb TV (Bahasa: %s, Halaman: %s%s)...\n", *dramaLang, pageInfo, yearInfo)
		count, err := pipe.IngestAsianDramas(*dramaLang, *dramaPages, *tmdbYear)
		if err != nil {
			log.Printf("[ERROR] Scraping Drama Asia gagal: %v\n", err)
		} else {
			fmt.Printf("\n[SUCCESS] Berhasil scrape dan simpan %d drama asia (WebP + Subtitle) ke MariaDB!\n", count)
		}
		printDatabaseStats(db)
		return
	}

	if *scrapeHollywood {
		yearInfo := ""
		if *tmdbYear > 0 {
			yearInfo = fmt.Sprintf(", Tahun: %d", *tmdbYear)
		}
		pageInfo := "Semua Halaman (Otomatis)"
		if *hollywoodPages > 0 {
			pageInfo = fmt.Sprintf("%d Halaman", *hollywoodPages)
		}
		fmt.Printf("\n[ACTION] 🎬 Memulai scraping Film Hollywood / Box Office resmi TMDb (Kategori: %s, Halaman: %s%s)...\n", *hollywoodCat, pageInfo, yearInfo)
		count, err := pipe.IngestHollywoodMovies(*hollywoodCat, *hollywoodPages, *tmdbYear)
		if err != nil {
			log.Printf("[ERROR] Scraping Hollywood gagal: %v\n", err)
		} else {
			fmt.Printf("\n[SUCCESS] Berhasil scrape dan simpan %d film Hollywood / Box Office (WebP + Subtitle + Streaming) ke MariaDB!\n", count)
		}
		printDatabaseStats(db)
		return
	}

	if *populateStreams {
		fmt.Println("\n[ACTION] 🎬 Memulai sinkronisasi server streaming embed (VidSrc, AutoEmbed, 2Embed, VidLink) untuk semua judul di database...")
		count, err := pipe.PopulateMissingStreamLinks(db)
		if err != nil {
			log.Printf("[ERROR] Sinkronisasi stream link gagal: %v\n", err)
		} else {
			fmt.Printf("\n[SUCCESS] Berhasil menambahkan server streaming untuk %d judul film/anime/drama di MariaDB!\n", count)
		}
		printDatabaseStats(db)
		return
	}

	if *fixTitles {
		fmt.Println("\n[ACTION] 🛠️ Memulai perbaikan judul non-Latin (Kanji/CJK) ke English QWERTY dan pengisian sinopsis kosong...")
		count, err := pipe.FixNonLatinTitlesAndSynopses(db)
		if err != nil {
			log.Printf("[ERROR] Perbaikan judul & sinopsis gagal: %v\n", err)
		} else {
			fmt.Printf("\n[SUCCESS] Berhasil memperbarui %d judul & sinopsis film/anime/drama di MariaDB!\n", count)
		}
		printDatabaseStats(db)
		return
	}

	if *translateSynopsis {
		fmt.Println("\n[ACTION] 🌐 Memulai sinkronisasi terjemahan sinopsis dwibahasa (Indonesia & English)...")
		count, err := pipeline.TranslateAllExistingSynopses(db)
		if err != nil {
			log.Printf("[ERROR] Sinkronisasi terjemahan sinopsis gagal: %v\n", err)
		} else {
			fmt.Printf("\n[SUCCESS] Berhasil menerjemahkan dan menyinkronkan %d sinopsis film di MariaDB!\n", count)
		}
		printDatabaseStats(db)
		return
	}

	if *populateDownloads {
		fmt.Println("\n[ACTION] 📥 Memulai sinkronisasi dan pengisian URL link download secara massal...")
		ytsCli := yts.NewClient(cfg)
		count, err := pipeline.PopulateAllExistingDownloadLinks(db, ytsCli)
		if err != nil {
			log.Printf("[ERROR] Sinkronisasi link download gagal: %v\n", err)
		} else {
			fmt.Printf("\n[SUCCESS] Berhasil melengkapi link download untuk %d judul film di MariaDB!\n", count)
		}
		printDatabaseStats(db)
		return
	}

	if *dailyScrape {
		targetYear := *tmdbYear
		if targetYear <= 0 {
			loc, _ := time.LoadLocation("Asia/Jakarta")
			if loc != nil {
				targetYear = time.Now().In(loc).Year()
			} else {
				targetYear = time.Now().Year()
			}
		}
		fmt.Printf("\n[ACTION] ⏰ Menjalankan siklus scraper harian lengkap untuk tahun %d...\n", targetYear)
		count, err := autoScraper.RunDailyCatchupCycle(targetYear)
		if err != nil {
			log.Printf("[ERROR] Siklus scraper harian gagal: %v\n", err)
		} else {
			fmt.Printf("\n[SUCCESS] Berhasil! Total %d konten baru/diperbarui untuk tahun %d.\n", count, targetYear)
		}
		printDatabaseStats(db)
		return
	}

	// 6. Start Web Server with background Auto-Scraper enabled
	if *serveWeb || (!*fetchArchive && !*fetchOpenMovies && !*syncAll && *tmdbQuery == "" && *scrapeURL == "" && *byYear == 0 && !*convertImages && !*checkLinks && !*scrapeAnime && !*scrapeDrama && !*scrapeHollywood && !*populateStreams && !*fixTitles && !*translateSynopsis && !*populateDownloads && !*dailyScrape) {
		// Launch background polite auto-scraper
		autoScraper.Start()

		// Background routine: Convert any existing movie images to local WebP & validate links
		go func() {
			time.Sleep(3 * time.Second)
			pipeline.ConvertExistingImagesToWebP(db, cfg.ScraperUserAgent, 200)
			validator.ValidateAllDatabaseLinks(db, cfg.ScraperUserAgent, 100)
		}()

		webServer := server.NewServer(cfg, db)
		if err := webServer.Start(); err != nil {
			log.Fatalf("[FATAL] Web server error: %v\n", err)
		}
		return
	}

	printDatabaseStats(db)

	fmt.Println("\n[INFO] Perintah CLI (Terminal) yang dapat digunakan:")
	fmt.Println("  go run . -serve                       # 🚀 JALANKAN WEB SERVER & AUTO-SCRAPER BACKGROUND")
	fmt.Println("  go run . -scrape-hollywood -year 2026 # 🎬 Scrape SEMUA halaman film Hollywood/Box Office tahun 2026")
	fmt.Println("  go run . -scrape-anime -year 2026     # 🎌 Scrape SEMUA halaman anime rilis tahun 2026")
	fmt.Println("  go run . -scrape-drama -year 2026     # 🎭 Scrape SEMUA halaman drama Asia rilis tahun 2026")
	fmt.Println("  go run . -populate-downloads          # 📥 Isi dan perbarui URL link download massal di database")
	fmt.Println("  go run . -fix-titles                  # 🛠️ Perbaiki judul Kanji ke English QWERTY & isi sinopsis kosong")
	fmt.Println("  go run . -translate-synopsis          # 🌐 Sinkronisasi terjemahan sinopsis dwibahasa (ID & EN)")
	fmt.Println("  go run . -populate-streams            # 🎬 Isi/perbarui server streaming embed untuk semua judul")
	fmt.Println("  go run . -check-links                # 🔍 Pengecekan & validasi link download file di database")
	fmt.Println("  go run . -convert-images             # 🖼️ Download & konversi semua poster/backdrop di DB ke WebP lokal")
	fmt.Println("  go run . -auto-scrape                # 🤖 Jalankan 1 siklus scraping ramah server semua tahun")
	fmt.Println("  go run . -daemon                     # ⏳ Jalankan scraper otomatis di background (cron)")
	fmt.Println("  go run . -by-year 2024               # Scrape film-film rilis tahun 2024")
	fmt.Println("  go run . -archive                    # Ambil film legal dari Internet Archive API")
	fmt.Println("  go run . -tmdb \"Inception\"           # Ambil metadata HD dari TMDb API")
	_ = os.Stdout.Sync()
}

func printDatabaseStats(db *gorm.DB) {
	var movieCount, freeCount, legalCount, genreCount, directorCount, actorCount, downloadCount, streamCount, logCount int64
	db.Model(&model.Movie{}).Count(&movieCount)
	db.Model(&model.Movie{}).Where("is_free = ?", true).Count(&freeCount)
	db.Model(&model.Movie{}).Where("is_legal = ?", true).Count(&legalCount)
	db.Model(&model.Genre{}).Count(&genreCount)
	db.Model(&model.Director{}).Count(&directorCount)
	db.Model(&model.Actor{}).Count(&actorCount)
	db.Model(&model.DownloadLink{}).Count(&downloadCount)
	db.Model(&model.StreamLink{}).Count(&streamCount)
	db.Model(&model.ScrapeLog{}).Count(&logCount)

	fmt.Println("\n======================== DATABASE STATS ========================")
	fmt.Printf("Total Film (Movies)       : %d\n", movieCount)
	fmt.Printf("  ├── Legal / Sah         : %d film\n", legalCount)
	fmt.Printf("  ├── Gratis (Open/Public): %d film (Free to watch/download)\n", freeCount)
	fmt.Printf("  └── Berlisensi Komersil : %d film (Copyrighted / TMDb)\n", movieCount-freeCount)
	fmt.Printf("Total Genres              : %d\n", genreCount)
	fmt.Printf("Total Directors           : %d\n", directorCount)
	fmt.Printf("Total Actors              : %d\n", actorCount)
	fmt.Printf("Total Download Links      : %d\n", downloadCount)
	fmt.Printf("Total Stream Players      : %d\n", streamCount)
	fmt.Printf("Total Scrape Logs         : %d\n", logCount)
	fmt.Println("================================================================")
}
