package scheduler

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"cinembrot/config"
	"cinembrot/model"
	"cinembrot/pipeline"
	"cinembrot/provider/vorta"
	"cinembrot/scraper"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"gorm.io/gorm"
)

// AutoScraper manages background automated scraping with polite rate limiting
type AutoScraper struct {
	cfg        *config.Config
	db         *gorm.DB
	pipe       *pipeline.Pipeline
	repo       *scraper.Repository
	stopChan   chan struct{}
	dailyLock  sync.Mutex
	isDailyRun bool
}

func NewAutoScraper(cfg *config.Config, db *gorm.DB, pipe *pipeline.Pipeline, repo *scraper.Repository) *AutoScraper {
	return &AutoScraper{
		cfg:      cfg,
		db:       db,
		pipe:     pipe,
		repo:     repo,
		stopChan: make(chan struct{}),
	}
}

// GetSchedulerConfig loads dynamic scheduler configuration from MariaDB SystemSetting table
func GetSchedulerConfig(db *gorm.DB, defaultCfg *config.Config) model.SchedulerConfig {
	cfg := model.SchedulerConfig{
		Enabled:         defaultCfg.AutoScrapeEnabled,
		IntervalMinutes: defaultCfg.AutoScrapeIntervalMinutes,
		StartYear:       defaultCfg.AutoScrapeStartYear,
		EndYear:         defaultCfg.AutoScrapeEndYear,
		PagesPerYear:    defaultCfg.AutoScrapePagesPerYear,
		DelayMs:         defaultCfg.AutoScrapeDelayMs,
	}

	if db != nil {
		var settings []model.SystemSetting
		if err := db.Find(&settings).Error; err == nil {
			for _, s := range settings {
				switch s.Key {
				case "auto_scrape_enabled":
					cfg.Enabled = s.Value == "true" || s.Value == "1" || s.Value == "on"
				case "auto_scrape_interval_minutes":
					if v, err := strconv.Atoi(s.Value); err == nil && v > 0 {
						cfg.IntervalMinutes = v
					}
				case "auto_scrape_start_year":
					if v, err := strconv.Atoi(s.Value); err == nil && v > 0 {
						cfg.StartYear = v
					}
				case "auto_scrape_end_year":
					if v, err := strconv.Atoi(s.Value); err == nil && v > 0 {
						cfg.EndYear = v
					}
				case "auto_scrape_pages_per_year":
					if v, err := strconv.Atoi(s.Value); err == nil && v > 0 {
						cfg.PagesPerYear = v
					}
				case "auto_scrape_delay_ms":
					if v, err := strconv.Atoi(s.Value); err == nil && v >= 0 {
						cfg.DelayMs = v
					}
				}
			}
		}
	}

	if cfg.IntervalMinutes < 5 {
		cfg.IntervalMinutes = 5
	}
	if cfg.PagesPerYear < 1 {
		cfg.PagesPerYear = 1
	}
	if cfg.StartYear == 0 {
		cfg.StartYear = time.Now().Year()
	}
	if cfg.EndYear == 0 {
		cfg.EndYear = 2015
	}

	return cfg
}

// Start launches the background scheduler goroutine
func (s *AutoScraper) Start() {
	log.Printf("\n[SCHEDULER] 🤖 Background Auto-Scraper Scheduler aktif (Menggunakan pengaturan MariaDB & CMS)...\n")

	// 1. Pengawas Jadwal Harian Tetap (Pukul 05:00 & 17:00 WIB)
	s.startDailyFixedScheduler()

	// 2. Pengawas Sinkronisasi Mingguan Platform Drama Pendek (/api/v2/categories)
	s.startWeeklyCategorySyncScheduler()

	// 3. Interval Auto-Scraper umum
	go func() {
		// Run initial check after 10 seconds of startup
		time.Sleep(10 * time.Second)
		s.RunFullCycle()

		for {
			schedCfg := GetSchedulerConfig(s.db, s.cfg)
			interval := time.Duration(schedCfg.IntervalMinutes) * time.Minute
			if interval < 5*time.Minute {
				interval = 5 * time.Minute
			}

			select {
			case <-time.After(interval):
				s.RunFullCycle()
			case <-s.stopChan:
				log.Println("[INFO] Auto-Scraper stopped.")
				return
			}
		}
	}()
}

// Stop stops the scheduler
func (s *AutoScraper) Stop() {
	close(s.stopChan)
}

// RunFullCycle executes a gentle, multi-source movie scraping sweep
func (s *AutoScraper) RunFullCycle() {
	schedCfg := GetSchedulerConfig(s.db, s.cfg)
	if !schedCfg.Enabled {
		log.Println("[AUTO-SCRAPER] ⏸️ Jadwal scraper otomatis dinonaktifkan (OFF) di panel CMS. Siklus otomatis dilewati.")
		return
	}

	startTime := time.Now()
	log.Println("\n==============================================================")
	log.Printf(" [AUTO-SCRAPER] 🚀 Memulai siklus scraping otomatis (Tahun: %d - %d, Interval: %d menit, %d hal/tahun)\n",
		schedCfg.StartYear, schedCfg.EndYear, schedCfg.IntervalMinutes, schedCfg.PagesPerYear)
	log.Println("==============================================================")

	totalIngested := 0

	// 0. Ambil daftar sumber yang AKTIF dari tabel scrape_sources di MariaDB
	var activeSources []model.ScrapeSource
	s.db.Where("is_active = ?", true).Find(&activeSources)

	isSourceActive := func(code string) bool {
		for _, as := range activeSources {
			if as.Code == code {
				return true
			}
		}
		return false
	}

	// 1. Sync Creative Commons Open Movies (jika aktif di DB)
	if isSourceActive("blender") {
		log.Println("[AUTO-SCRAPER] 🎬 Memeriksa film Open Source / Creative Commons (Blender)...")
		openCount, err := s.pipe.IngestOpenMovies()
		if err == nil {
			totalIngested += openCount
		}
		s.politeSleep()
	}

	// 2. Discover movies across years (from StartYear down to EndYear)
	startYear := schedCfg.StartYear
	endYear := schedCfg.EndYear
	pagesPerYear := schedCfg.PagesPerYear

	if startYear < endYear {
		startYear, endYear = endYear, startYear
	}

	for year := startYear; year >= endYear; year-- {
		// Periksa kembali status ON/OFF di setiap tahun agar responsive jika user mematikan jadwal di tengah jalan
		currCfg := GetSchedulerConfig(s.db, s.cfg)
		if !currCfg.Enabled {
			log.Println("[AUTO-SCRAPER] ⏸️ Auto-scraper dimatikan dari CMS saat siklus berjalan. Menghentikan penelusuran tahun.")
			break
		}

		log.Printf("[AUTO-SCRAPER] 📅 Menelusuri film rilis tahun %d (Maks %d halaman)...\n", year, pagesPerYear)

		for page := 1; page <= pagesPerYear; page++ {
			// A. Ingest TMDb Popular movies (HANYA jika aktif di DB)
			if isSourceActive("tmdb") {
				count, err := s.pipe.IngestByYear(year, 1, "tmdb")
				if err == nil {
					totalIngested += count
				}
				s.politeSleep()
			}

			// B. Ingest YTS Torrents (HANYA jika aktif di DB)
			if isSourceActive("yts") {
				ytsCount, err := s.pipe.IngestByYear(year, 1, "yts")
				if err == nil {
					totalIngested += ytsCount
				}
				s.politeSleep()
			}
		}

		// C. Archive.org feature films (HANYA jika aktif di DB)
		if isSourceActive("archive") && (year%5 == 0 || year == startYear) {
			archCount, _ := s.pipe.IngestByYear(year, 1, "archive")
			totalIngested += archCount
			s.politeSleep()
		}
	}

	// 3. Log cycle summary
	duration := time.Since(startTime)
	log.Printf("\n[AUTO-SCRAPER] ✅ Siklus scraping selesai dalam %v. Total %d film tersimpan/diperbarui.\n",
		duration.Round(time.Second), totalIngested)

	_ = s.repo.LogScrape("AutoScraper-FullCycle", fmt.Sprintf("years=%d-%d", startYear, endYear),
		"SUCCESS", totalIngested, "", duration)
}

// politeSleep adds a polite, randomized delay so we never overwhelm target servers
func (s *AutoScraper) politeSleep() {
	baseMs := s.cfg.AutoScrapeDelayMs
	if baseMs <= 0 {
		baseMs = 2000
	}
	// Add 0-1000ms jitter to simulate natural human requests
	jitter := rand.Intn(1000)
	sleepDuration := time.Duration(baseMs+jitter) * time.Millisecond
	time.Sleep(sleepDuration)
}

// startDailyFixedScheduler monitors the clock and triggers RunDailyCatchupCycle at 05:00 WIB and 17:00 WIB daily
func (s *AutoScraper) startDailyFixedScheduler() {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}

	go func() {
		log.Printf("[DAILY-SCHEDULER ⏰] Pengawas jadwal harian aktif (Target: 05:00 WIB & 17:00 WIB)...\n")
		for {
			now := time.Now().In(loc)

			// Waktu target hari ini
			t05 := time.Date(now.Year(), now.Month(), now.Day(), 5, 0, 0, 0, loc)
			t17 := time.Date(now.Year(), now.Month(), now.Day(), 17, 0, 0, 0, loc)

			var nextRun time.Time
			if now.Before(t05) {
				nextRun = t05
			} else if now.Before(t17) {
				nextRun = t17
			} else {
				// Sudah lewat jam 17:00, target berikutnya adalah jam 05:00 besok
				nextRun = t05.Add(24 * time.Hour)
			}

			waitDur := time.Until(nextRun)
			log.Printf("[DAILY-SCHEDULER ⏰] ⏳ Eksekusi berikutnya dijadwalkan pada: %s (%v lagi)\n",
				nextRun.Format("2006-01-02 15:04:05 MST"), waitDur.Round(time.Minute))

			select {
			case <-time.After(waitDur):
				go func() {
					currentYear := time.Now().In(loc).Year()
					log.Printf("[DAILY-SCHEDULER ⏰] 🔔 Waktu jadwal harian (%s WIB) tercapai! Memulai scraping tahun %d...\n",
						time.Now().In(loc).Format("15:04"), currentYear)
					_, _ = s.RunDailyCatchupCycle(currentYear)
				}()
				// Jeda sejenak 2 menit agar loop tidak mendeteksi menit yang sama
				time.Sleep(2 * time.Minute)
			case <-s.stopChan:
				log.Println("[DAILY-SCHEDULER] Pengawas jadwal harian dihentikan.")
				return
			}
		}
	}()
}

// RunDailyCatchupCycle executes a full catch-up scrape across all categories for the target year (2026/current year):
// 1. Anime (Seasonal on-going & Top) untuk judul dan episode baru
// 2. Asian Dramas (Korea, China, Jepang, Thailand) untuk judul dan episode baru
// 3. Hollywood Movies (Now Playing bioskop & Populer) untuk film baru
// 4. TMDb & YTS general movies untuk film baru
func (s *AutoScraper) RunDailyCatchupCycle(targetYear int) (int, error) {
	s.dailyLock.Lock()
	if s.isDailyRun {
		s.dailyLock.Unlock()
		log.Println("[DAILY-SCHEDULER ⚠️] Siklus harian sedang berjalan saat ini. Panggilan baru diabaikan.")
		return 0, nil
	}
	s.isDailyRun = true
	s.dailyLock.Unlock()

	defer func() {
		s.dailyLock.Lock()
		s.isDailyRun = false
		s.dailyLock.Unlock()
	}()

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}

	if targetYear <= 0 {
		targetYear = time.Now().In(loc).Year()
	}

	startTime := time.Now()
	log.Println("\n==========================================================================")
	log.Printf(" [DAILY-SCHEDULER ⏰] 🚀 Memulai siklus scraping harian (Pukul 05:00/17:00 WIB)\n")
	log.Printf("  Target Tahun: %d (Film Baru, Anime On-Going, Drama Asia & Episode Baru)\n", targetYear)
	log.Println("==========================================================================")

	totalIngested := 0

	// 1. ANIME: Seasonal On-Going & Top Anime Tahun Sekarang (Mendeteksi Anime & Episode Baru)
	log.Printf("[DAILY-SCHEDULER] 🎌 [1/4] Scraping Anime Musim Ini / On-Going & Populer Tahun %d...\n", targetYear)
	if count, err := s.pipe.IngestAnime("seasonal", 0, targetYear); err == nil {
		totalIngested += count
		log.Printf("[DAILY-SCHEDULER]  -> Berhasil ingest %d anime seasonal/on-going.\n", count)
	} else {
		log.Printf("[DAILY-SCHEDULER]  ⚠️ Ingest anime seasonal notice: %v\n", err)
	}
	s.politeSleep()

	if count, err := s.pipe.IngestAnime("top", 50, targetYear); err == nil {
		totalIngested += count
		log.Printf("[DAILY-SCHEDULER]  -> Berhasil ingest %d anime top tahun %d.\n", count, targetYear)
	} else {
		log.Printf("[DAILY-SCHEDULER]  ⚠️ Ingest anime top notice: %v\n", err)
	}
	s.politeSleep()

	// 2. DRAMA ASIA: Serial Drama Korea, China, Jepang, Thailand Tahun Sekarang (Mendeteksi Drama & Episode Baru)
	log.Printf("[DAILY-SCHEDULER] 🎭 [2/4] Scraping Serial Drama Asia (Korea, China, Jepang, Thailand) Tahun %d...\n", targetYear)
	dramaLangs := []struct {
		code string
		name string
	}{
		{"ko", "Korea (K-Drama)"},
		{"zh", "China (C-Drama)"},
		{"ja", "Jepang (J-Drama)"},
		{"th", "Thailand (Thai Drama)"},
	}
	for _, dl := range dramaLangs {
		log.Printf("[DAILY-SCHEDULER]  -> Memindai Drama %s (Tahun %d)...\n", dl.name, targetYear)
		if count, err := s.pipe.IngestAsianDramas(dl.code, 3, targetYear); err == nil {
			totalIngested += count
			log.Printf("[DAILY-SCHEDULER]  -> Berhasil ingest %d serial drama %s.\n", count, dl.name)
		} else {
			log.Printf("[DAILY-SCHEDULER]  ⚠️ Ingest drama %s notice: %v\n", dl.name, err)
		}
		s.politeSleep()
	}

	// 3. HOLLYWOOD / BOX OFFICE: Film Rilis Bioskop Terbaru (Now Playing) & Populer Tahun Sekarang
	log.Printf("[DAILY-SCHEDULER] 🎬 [3/4] Scraping Film Hollywood / Box Office Bioskop Terbaru Tahun %d...\n", targetYear)
	if count, err := s.pipe.IngestHollywoodMovies("now_playing", 3, targetYear); err == nil {
		totalIngested += count
		log.Printf("[DAILY-SCHEDULER]  -> Berhasil ingest %d film Hollywood Now Playing bioskop.\n", count)
	} else {
		log.Printf("[DAILY-SCHEDULER]  ⚠️ Ingest Hollywood now_playing notice: %v\n", err)
	}
	s.politeSleep()

	if count, err := s.pipe.IngestHollywoodMovies("popular", 3, targetYear); err == nil {
		totalIngested += count
		log.Printf("[DAILY-SCHEDULER]  -> Berhasil ingest %d film Hollywood Popular.\n", count)
	} else {
		log.Printf("[DAILY-SCHEDULER]  ⚠️ Ingest Hollywood popular notice: %v\n", err)
	}
	s.politeSleep()

	// 4. TMDB & YTS: Katalog Film Umum Tahun Sekarang
	log.Printf("[DAILY-SCHEDULER] 🍿 [4/4] Scraping Katalog Film TMDb & YTS Tahun %d...\n", targetYear)
	if count, err := s.pipe.IngestByYear(targetYear, 3, "tmdb"); err == nil {
		totalIngested += count
		log.Printf("[DAILY-SCHEDULER]  -> Berhasil ingest %d film TMDb.\n", count)
	} else {
		log.Printf("[DAILY-SCHEDULER]  ⚠️ Ingest TMDb by-year notice: %v\n", err)
	}
	s.politeSleep()

	if count, err := s.pipe.IngestByYear(targetYear, 3, "yts"); err == nil {
		totalIngested += count
		log.Printf("[DAILY-SCHEDULER]  -> Berhasil ingest %d film YTS.\n", count)
	} else {
		log.Printf("[DAILY-SCHEDULER]  ⚠️ Ingest YTS by-year notice: %v\n", err)
	}

	duration := time.Since(startTime)
	log.Printf("\n[DAILY-SCHEDULER] ✅ SIKLUS HARIAN SELESAI dalam %v. Total %d konten (Film/Anime/Drama/Episode Baru) diproses.\n",
		duration.Round(time.Second), totalIngested)

	_ = s.repo.LogScrape("DailyScheduler-05:00/17:00", fmt.Sprintf("year=%d (anime,drama,hollywood,tmdb,yts)", targetYear),
		"SUCCESS", totalIngested, "", duration)

	return totalIngested, nil
}

// startWeeklyCategorySyncScheduler menjalankan sinkronisasi platform /api/v2/categories sekali seminggu (setiap Senin 03:00 WIB)
// serta melakukan pengecekan awal saat startup jika tabel drama_categories masih kosong.
func (s *AutoScraper) startWeeklyCategorySyncScheduler() {
	go func() {
		// Pengecekan awal saat startup: jika tabel masih kosong, sync segera
		time.Sleep(15 * time.Second)
		var count int64
		if s.db != nil {
			s.db.Model(&model.DramaCategory{}).Count(&count)
			if count == 0 {
				log.Println("[WEEKLY-SYNC] 🚀 Tabel drama_categories kosong, menjalankan initial sync /api/v2/categories...")
				if _, err := SyncDramaCategories(s.db); err != nil {
					log.Printf("[WEEKLY-SYNC] Initial sync error: %v\n", err)
				}
			}
		}

		for {
			loc, err := time.LoadLocation("Asia/Jakarta")
			if err != nil {
				loc = time.FixedZone("WIB", 7*3600)
			}
			now := time.Now().In(loc)

			// Target: Setiap hari Senin pukul 03:00 WIB
			target := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, loc)
			daysUntilMonday := (int(time.Monday) - int(now.Weekday()) + 7) % 7
			if daysUntilMonday == 0 && now.After(target) {
				daysUntilMonday = 7
			}
			target = target.AddDate(0, 0, daysUntilMonday)

			sleepDuration := target.Sub(now)
			log.Printf("[WEEKLY-SYNC] 📅 Jadwal sinkronisasi mingguan platform drama berikutnya: %s (%v lagi)\n",
				target.Format("2006-01-02 15:04:05 MST"), sleepDuration.Round(time.Minute))

			select {
			case <-time.After(sleepDuration):
				log.Println("[WEEKLY-SYNC] 🔄 Menjalankan sinkronisasi mingguan platform drama dari /api/v2/categories...")
				if updated, err := SyncDramaCategories(s.db); err != nil {
					log.Printf("[WEEKLY-SYNC] ⚠️ Gagal sinkronisasi mingguan platform: %v\n", err)
				} else {
					log.Printf("[WEEKLY-SYNC] ✅ Berhasil memperbarui %d platform drama di database.\n", updated)
				}

				// Sync katalog discovery mingguan (sopan, 1 halaman per platform, jeda santai 10 detik antar platform)
				log.Println("[WEEKLY-SYNC] 🍿 Menjalankan sinkronisasi mingguan katalog Discovery drama pendek...")
				SyncDramaDiscoveryCatalog(s.db)
			case <-s.stopChan:
				return
			}
		}
	}()
}

// SyncDramaCategories mengambil data platform resmi dari /api/v2/categories dan menyimpannya ke MariaDB
func SyncDramaCategories(db *gorm.DB) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}

	client := vorta.NewClient()
	cats, err := client.GetCategories()
	if err != nil {
		return 0, fmt.Errorf("gagal memanggil /api/v2/categories: %w", err)
	}

	if len(cats) == 0 {
		return 0, nil
	}

	updatedCount := 0
	// Pastikan direktori lokal untuk logo ada
	localLogoDir := filepath.Join("public", "img", "platforms")
	_ = os.MkdirAll(localLogoDir, 0755)

	for _, c := range cats {
		// Download atau pastikan logo tersimpan lokal untuk mencegah blokir hotlink / CORS
		savedLogoPath := c.LogoURL
		if c.LogoURL != "" {
			fileName := fmt.Sprintf("%s.png", c.Name)
			localFilePath := filepath.Join(localLogoDir, fileName)
			webPath := fmt.Sprintf("/img/platforms/%s", fileName)

			// Download logo jika belum ada di lokal
			if _, statErr := os.Stat(localFilePath); os.IsNotExist(statErr) {
				req, _ := http.NewRequest("GET", c.LogoURL, nil)
				req.Header.Set("User-Agent", "Cinembrot/1.0")
				client := &http.Client{Timeout: 10 * time.Second}
				if resp, err := client.Do(req); err == nil && resp.StatusCode == http.StatusOK {
					defer resp.Body.Close()
					if out, err := os.Create(localFilePath); err == nil {
						_, _ = io.Copy(out, resp.Body)
						out.Close()
						savedLogoPath = webPath
					}
				}
			} else {
				savedLogoPath = webPath
			}
		}

		var existing model.DramaCategory
		err := db.Where("name = ?", c.Name).First(&existing).Error
		if err != nil {
			// Insert record baru
			newCat := model.DramaCategory{
				ExternalID:  c.ID,
				Name:        c.Name,
				DisplayName: c.DisplayName,
				LogoURL:     savedLogoPath,
				Order:       c.Order,
				IsActive:    c.IsActive,
				Is18Plus:    c.Is18Plus,
			}
			if err := db.Create(&newCat).Error; err == nil {
				updatedCount++
			}
		} else {
			// Update data yang ada
			existing.ExternalID = c.ID
			existing.DisplayName = c.DisplayName
			existing.LogoURL = savedLogoPath
			existing.Order = c.Order
			existing.IsActive = c.IsActive
			existing.Is18Plus = c.Is18Plus
			if err := db.Save(&existing).Error; err == nil {
				updatedCount++
			}
		}
	}

	log.Printf("[VORTA-CATEGORIES] Berhasil menyimpan %d platform resmi ke database MariaDB.\n", updatedCount)
	return updatedCount, nil
}

// SyncDramaDiscoveryCatalog mengambil katalog drama rilis terkini secara berkala 1x seminggu
// Sangat sopan: hanya 1 halaman per platform aktif, dengan jeda santai 10 detik antar platform (Anti-Brute Force).
func SyncDramaDiscoveryCatalog(db *gorm.DB) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}

	var categories []model.DramaCategory
	if err := db.Where("is_active = ?", true).Order("`order` asc").Find(&categories).Error; err != nil {
		return 0, err
	}

	client := vorta.NewClient()
	totalSaved := 0

	for _, cat := range categories {
		log.Printf("[VORTA-DISCOVER] 📥 Mengambil sampel discovery mingguan untuk platform: %s (id & en)...\n", cat.DisplayName)

		for _, lang := range []string{"id", "en"} {
			resp, err := client.GetDiscover(cat.Name, lang, 1)
			if err != nil {
				log.Printf("[VORTA-DISCOVER] ⚠️ Notice fetch %s (%s): %v\n", cat.Name, lang, err)
				time.Sleep(5 * time.Second)
				continue
			}

			for _, item := range resp.Data {
				saved, err := SaveDramaItemToDatabase(db, item, cat.Name, lang)
				if err == nil && saved {
					totalSaved++
				}
			}

			// Jeda santai 5 detik antar bahasa
			time.Sleep(5 * time.Second)
		}

		// Jeda santai 10 detik antar platform untuk menjaga keamanan API rate limit
		time.Sleep(10 * time.Second)
	}

	log.Printf("[VORTA-DISCOVER] ✅ Selesai! %d judul drama berhasil disinkronkan ke MariaDB.\n", totalSaved)
	return totalSaved, nil
}

// SaveDramaItemToDatabase menyimpan metadata drama pendek ke tabel movies
func SaveDramaItemToDatabase(db *gorm.DB, item vorta.DramaItem, platform, lang string) (bool, error) {
	if db == nil || item.ID == "" || item.Title == "" {
		return false, nil
	}

	cleanTitle := strings.TrimSpace(item.Title)
	slug := generateDramaSlug(cleanTitle)

	var existing model.Movie
	// Cek apakah drama dengan slug atau source_url sama sudah ada di database
	sourceURL := fmt.Sprintf("vorta://%s/%s", platform, item.ID)
	err := db.Where("source_url = ? OR slug = ?", sourceURL, slug).First(&existing).Error
	if err == nil {
		// Update cover atau episode jika berubah
		if existing.PosterURL == "" && item.Cover != "" {
			existing.PosterURL = item.Cover
			existing.ThumbnailURL = item.Cover
			_ = db.Save(&existing)
		}
		return false, nil
	}

	totalChapters := 1
	if ch, err := strconv.Atoi(strings.TrimSpace(item.Chapters)); err == nil && ch > 0 {
		totalChapters = ch
	}

	country := "Internasional"
	switch strings.ToLower(platform) {
	case "reelshort":
		country = "Amerika Serikat"
	case "mydrama":
		country = "Internasional"
	case "meloshort", "shortmax", "dramabox", "goodshort", "heatshort":
		country = "China"
	}

	langLabel := "Indonesia"
	if strings.EqualFold(lang, "en") {
		langLabel = "English"
	}

	movie := model.Movie{
		Title:             cleanTitle,
		OriginalTitle:     cleanTitle,
		Slug:              slug,
		Type:              "drama_pendek",
		Year:              time.Now().Year(),
		Status:            "publish",
		Country:           country,
		Language:          langLabel,
		PosterURL:         item.Cover,
		ThumbnailURL:      item.Cover,
		DurationFormatted: fmt.Sprintf("%d Episodes", totalChapters),
		SourceWebsite:     platform,
		SourceURL:         sourceURL,
		RawMetadata:       fmt.Sprintf("vorta:%s:%s", platform, item.ID),
		Views:             100,
		Rating:            8.5,
		Synopsis:          fmt.Sprintf("Serial drama pendek vertikal resmi dari platform %s. Ikuti keseruan jalan ceritanya dengan plot cepat dan penuh intrik.", strings.ToUpper(platform[:1])+platform[1:]),
	}

	if err := db.Create(&movie).Error; err != nil {
		return false, err
	}

	return true, nil
}

func generateDramaSlug(title string) string {
	slug := strings.ToLower(title)
	var sb strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			sb.WriteRune('-')
		}
	}
	clean := strings.Trim(sb.String(), "-")
	for strings.Contains(clean, "--") {
		clean = strings.ReplaceAll(clean, "--", "-")
	}
	if clean == "" {
		clean = fmt.Sprintf("drama-%d", time.Now().UnixNano()%100000)
	}
	return fmt.Sprintf("%s-%d", clean, time.Now().Year())
}
