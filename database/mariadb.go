package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cinembrot/auth"
	"cinembrot/config"
	"cinembrot/model"
	"cinembrot/scraper"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB initializes MariaDB connection and auto-migrates all tables
func InitDB(cfg *config.Config) (*gorm.DB, error) {
	// Step 1: Ensure database exists for local development
	if cfg.DBHost == "localhost" || cfg.DBHost == "127.0.0.1" {
		if err := ensureDatabaseExists(cfg); err != nil {
			log.Printf("[INFO] Notice checking local database creation: %v\n", err)
		}
	}

	// Step 2: Connect to the specific database
	gormDB, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MariaDB: %w", err)
	}

	// Step 3: Setup connection pooling
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get generic database object: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Step 4: Auto-migrate schemas
	log.Println("[INFO] Running auto-migration for database tables...")
	err = gormDB.AutoMigrate(
		&model.Movie{},
		&model.Genre{},
		&model.Director{},
		&model.Actor{},
		&model.Episode{},
		&model.DownloadLink{},
		&model.StreamLink{},
		&model.Schedule{},
		&model.ScrapeLog{},
		&model.Comment{},
		&model.User{},
		&model.ScrapeSource{},
		&model.SystemSetting{},
		&model.TorrentTask{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to auto-migrate tables: %w", err)
	}

	// Seed default admin user if none exists
	seedAdminUser(gormDB, cfg)

	// Seed default website sources in database
	seedDefaultSources(gormDB)

	// Seed default system & scheduler settings
	seedDefaultSettings(gormDB)

	// Sanitize any existing movie synopses containing HTML tags
	SanitizeExistingSynopses(gormDB)

	log.Printf("[SUCCESS] Connected to MariaDB '%s' on %s:%s and migrations completed.\n",
		cfg.DBName, cfg.DBHost, cfg.DBPort)

	DB = gormDB
	return gormDB, nil
}

func seedAdminUser(db *gorm.DB, cfg *config.Config) {
	adminUser := cfg.AdminDefaultUser
	if adminUser == "" {
		adminUser = "admin"
	}
	adminPass := cfg.AdminDefaultPass
	if adminPass == "" {
		adminPass = "CINEMBROT123"
	}

	var user model.User
	if err := db.Where("username = ?", adminUser).First(&user).Error; err != nil {
		user = model.User{
			Username:     adminUser,
			PasswordHash: auth.HashPassword(adminPass),
			FullName:     "Administrator",
			Role:         "admin",
			IsActive:     true,
		}
		_ = db.Create(&user)
		log.Printf("[CMS] 🔑 Akun Admin Default dibuat: Username '%s' | Password '%s'\n", adminUser, adminPass)
	} else if !auth.CheckPasswordHash(adminPass, user.PasswordHash) {
		// Update password hash if needed
		db.Model(&user).Update("password_hash", auth.HashPassword(adminPass))
		log.Printf("[CMS] 🔑 Password Admin Default diperbarui: '%s'\n", adminUser)
	}
}

// seedDefaultSources initializes website scraping sources in MariaDB
func seedDefaultSources(db *gorm.DB) {
	defaultSources := []model.ScrapeSource{
		{
			Name:            "The Movie Database (TMDb)",
			Code:            "tmdb",
			BaseURL:         "https://api.themoviedb.org/3",
			Type:            "api",
			Category:        "Metadata & Popular",
			Description:     "Penyedia katalog metadata film, rating, sinopsis, poster resolusi tinggi, sutradara, dan cast aktor.",
			IsActive:        true,
			RateLimitPerSec: 5,
		},
		{
			Name:            "Internet Archive (Feature Films)",
			Code:            "archive",
			BaseURL:         "https://archive.org/details/feature_films",
			Type:            "api",
			Category:        "Public Domain & Legal Downloads",
			Description:     "Arsip publik film bioskop klasik, public domain, dan open license dengan link download video langsung.",
			IsActive:        true,
			RateLimitPerSec: 3,
		},
		{
			Name:            "Blender Open Studio",
			Code:            "blender",
			BaseURL:         "https://studio.blender.org/films/",
			Type:            "html_scrape",
			Category:        "Creative Commons Open Movies",
			Description:     "Film animasi open source berkualitas 4K Creative Commons (Sintel, Tears of Steel, Big Buck Bunny, Spring, Charge).",
			IsActive:        true,
			RateLimitPerSec: 2,
		},
		{
			Name:            "Public Domain Movies Hub",
			Code:            "publicdomain",
			BaseURL:         "https://publicdomainmovies.info/",
			Type:            "html_scrape",
			Category:        "Public Domain Catalog",
			Description:     "Direktori kurasi film-film berlisensi domain publik bebas hak cipta komersial.",
			IsActive:        true,
			RateLimitPerSec: 2,
		},
		{
			Name:            "YTS Movies (YIFY Torrents)",
			Code:            "yts",
			BaseURL:         "https://yts.lt/api/v2",
			Type:            "api",
			Category:        "Torrent & Commercial Releases",
			Description:     "Penyedia REST API resmi film dengan link download file torrent dan magnet link resolusi 720p, 1080p, dan 4K.",
			IsActive:        true,
			RateLimitPerSec: 3,
		},
	}

	for _, s := range defaultSources {
		var existing model.ScrapeSource
		if err := db.Where("code = ?", s.Code).First(&existing).Error; err != nil {
			_ = db.Create(&s)
			log.Printf("[CMS DB] 🌐 Sumber website ditambahkan ke DB: '%s' (%s)\n", s.Name, s.Code)
		} else if s.Code == "yts" && existing.BaseURL == "https://yts.mx/api/v2" {
			db.Model(&existing).Update("base_url", "https://yts.lt/api/v2")
			log.Printf("[CMS DB] 🌐 Updated YTS BaseURL to active mirror: 'https://yts.lt/api/v2'\n")
		}
	}
}

func seedDefaultSettings(db *gorm.DB) {
	defaults := []model.SystemSetting{
		{Key: "auto_scrape_enabled", Value: "true", Description: "Saklar ON/OFF Auto-Scraper Latar Belakang"},
		{Key: "auto_scrape_interval_minutes", Value: "30", Description: "Interval waktu scraping otomatis (menit)"},
		{Key: "auto_scrape_start_year", Value: "2026", Description: "Tahun awal penelusuran katalog film"},
		{Key: "auto_scrape_end_year", Value: "2015", Description: "Tahun akhir penelusuran katalog film"},
		{Key: "auto_scrape_pages_per_year", Value: "1", Description: "Jumlah halaman yang discraping per tahun (1 halaman = 20 film)"},
		{Key: "auto_scrape_delay_ms", Value: "500", Description: "Jeda waktu ramah server antar permintaan film (ms)"},
		{Key: "download_movie_path", Value: "public/download/movie", Description: "Direktori penyimpanan file unduhan film dan hardsub subtitle"},
		{Key: "show_torrent_public", Value: "false", Description: "Tampilkan link torrent mentah di halaman publik film (true/false)"},
		{Key: "ads_enabled", Value: "true", Description: "Saklar Master ON/OFF Iklan di Website"},
		{Key: "adsterra_popunder_code", Value: `<script src="https://ripenhopperwitty.com/54/e2/d0/54e2d041392f6fe60241fb7998b44e22.js"></script>`, Description: "Kode Script Iklan Adsterra Popunder"},
		{Key: "adsterra_socialbar_code", Value: `<script src="https://ripenhopperwitty.com/e8/02/6c/e8026c7abe894a6b7c4f480910f52fc3.js"></script>`, Description: "Kode Script Iklan Adsterra Social Bar"},
		{Key: "adsterra_banner_728_code", Value: `<script>atOptions = {'key' : '85fa49a7c78f9d85d42fce646c7003ab','format' : 'iframe','height' : 90,'width' : 728,'params' : {}};</script><script src="https://ripenhopperwitty.com/85fa49a7c78f9d85d42fce646c7003ab/invoke.js"></script>`, Description: "Kode Script Iklan Adsterra Top Banner 728x90"},
		{Key: "adsterra_banner_468_code", Value: `<script>atOptions = {'key' : 'eaee3d9752226db4fc8a95711e98bd76','format' : 'iframe','height' : 60,'width' : 468,'params' : {}};</script><script src="https://ripenhopperwitty.com/eaee3d9752226db4fc8a95711e98bd76/invoke.js"></script>`, Description: "Kode Script Iklan Adsterra Banner 468x60"},
		{Key: "adsterra_banner_300_code", Value: `<script>atOptions = {'key' : '4a071f6c776fe2fe0e203dab2f8c93f7','format' : 'iframe','height' : 250,'width' : 300,'params' : {}};</script><script src="https://ripenhopperwitty.com/4a071f6c776fe2fe0e203dab2f8c93f7/invoke.js"></script>`, Description: "Kode Script Iklan Adsterra Medium Rectangle 300x250"},
		{Key: "adsterra_banner_160x600_code", Value: `<script>atOptions = {'key' : '0cc038844da89345b824c5c39dc1b8ad','format' : 'iframe','height' : 600,'width' : 160,'params' : {}};</script><script src="https://ripenhopperwitty.com/0cc038844da89345b824c5c39dc1b8ad/invoke.js"></script>`, Description: "Kode Script Iklan Adsterra Left Skyscraper 160x600"},
		{Key: "adsterra_banner_160x300_code", Value: `<script>atOptions = {'key' : '65bb105572567bac7c71464679347faf','format' : 'iframe','height' : 300,'width' : 160,'params' : {}};</script><script src="https://ripenhopperwitty.com/65bb105572567bac7c71464679347faf/invoke.js"></script>`, Description: "Kode Script Iklan Adsterra Right Skyscraper 160x300"},
		{Key: "adsterra_banner_320_code", Value: `<script>atOptions = {'key' : 'aed924266db38d0d234d16ebd3df5cad','format' : 'iframe','height' : 50,'width' : 320,'params' : {}};</script><script src="https://ripenhopperwitty.com/aed924266db38d0d234d16ebd3df5cad/invoke.js"></script>`, Description: "Kode Script Iklan Adsterra Mobile Sticky Banner 320x50"},
		{Key: "adsterra_native_banner_code", Value: `<script async="async" data-cfasync="false" src="https://ripenhopperwitty.com/4436b99755730d92de41c04a72905573/invoke.js"></script><div id="container-4436b99755730d92de41c04a72905573"></div>`, Description: "Kode Script Iklan Adsterra Native Banner Rekomendasi"},
		{Key: "adsterra_smartlink_url", Value: `https://ripenhopperwitty.com/dshsjtw4mm?key=b8e97cccf7d6fdfe0a74963ec89f2ae2`, Description: "URL Direct Adsterra Smartlink (Download / Streaming)"},
		{Key: "translate_enabled", Value: "true", Description: "Saklar ON/OFF Fitur Multi-Bahasa / Translate"},
		{Key: "google_translate_enabled", Value: "true", Description: "Saklar ON/OFF Mesin Google Website Translator"},
		{Key: "default_language", Value: "id", Description: "Bahasa Default Website (id / en)"},
		{Key: "comments_enabled", Value: "true", Description: "Saklar ON/OFF Kolom Komentar Penonton"},
		{Key: "site_name", Value: "CINEMBROT", Description: "Nama Brand Website"},
		{Key: "site_tagline", Value: "Platform Streaming & Download Film Gratis Legal", Description: "Tagline atau Slogan Website"},
		{Key: "turnstile_enabled", Value: "true", Description: "Saklar ON/OFF Cloudflare Turnstile CAPTCHA saat Login Admin"},
		{Key: "turnstile_site_key", Value: "0x4AAAAAAE4tnwYcBvU-hrsC", Description: "Cloudflare Turnstile Site Key"},
		{Key: "turnstile_secret_key", Value: "0x4AAAAAAE4tn3-kKYj8JmsodJoqg4Sk40g", Description: "Cloudflare Turnstile Secret Key"},
	}

	for _, s := range defaults {
		var existing model.SystemSetting
		if err := db.Where("`key` = ?", s.Key).First(&existing).Error; err != nil {
			_ = db.Create(&s)
		}
	}
}

// GetAllSettings returns all settings as a map
func GetAllSettings(db *gorm.DB) map[string]string {
	var settings []model.SystemSetting
	res := make(map[string]string)
	if err := db.Find(&settings).Error; err == nil {
		for _, s := range settings {
			res[s.Key] = s.Value
		}
	}
	return res
}

// GetSetting gets a single setting or returns defaultVal
func GetSetting(db *gorm.DB, key, defaultVal string) string {
	var s model.SystemSetting
	if err := db.Where("`key` = ?", key).First(&s).Error; err == nil && s.Value != "" {
		return s.Value
	}
	return defaultVal
}

// SaveSetting creates or updates a setting
func SaveSetting(db *gorm.DB, key, value, desc string) error {
	var s model.SystemSetting
	if err := db.Where("`key` = ?", key).First(&s).Error; err == nil {
		return db.Model(&s).Updates(map[string]interface{}{
			"value":       value,
			"description": desc,
			"updated_at":  time.Now(),
		}).Error
	}
	return db.Create(&model.SystemSetting{
		Key:         key,
		Value:       value,
		Description: desc,
		UpdatedAt:   time.Now(),
	}).Error
}

// GetDownloadMoviePath returns the absolute download path configured in system_settings or default
func GetDownloadMoviePath(db *gorm.DB) string {
	var s model.SystemSetting
	val := filepath.Join("public", "download", "movie")
	if err := db.Where("`key` = ?", "download_movie_path").First(&s).Error; err == nil && strings.TrimSpace(s.Value) != "" {
		val = strings.TrimSpace(s.Value)
	}

	if !filepath.IsAbs(val) {
		if abs, err := filepath.Abs(val); err == nil {
			val = abs
		}
	}
	_ = os.MkdirAll(val, 0755)
	return val
}

// GetShowTorrentPublic returns whether raw torrent download links should be shown on public movie detail pages (default: false)
func GetShowTorrentPublic(db *gorm.DB) bool {
	var s model.SystemSetting
	if err := db.Where("`key` = ?", "show_torrent_public").First(&s).Error; err == nil {
		val := strings.ToLower(strings.TrimSpace(s.Value))
		return val == "true" || val == "1" || val == "on"
	}
	return false
}

// SanitizeExistingSynopses cleans up any existing synopses in the database that still contain HTML tags
func SanitizeExistingSynopses(db *gorm.DB) {
	var movies []model.Movie
	db.Where("synopsis LIKE ? OR synopsis LIKE ? OR synopsis LIKE ?", "%<p>%", "%<a %", "%</div>%").Find(&movies)
	for _, m := range movies {
		clean := scraper.CleanHTMLToPlainText(m.Synopsis)
		if clean != m.Synopsis {
			db.Model(&model.Movie{}).Where("id = ?", m.ID).Update("synopsis", clean)
		}
	}
}

// ensureDatabaseExists connects without DB name to check or create target database
func ensureDatabaseExists(cfg *config.Config) error {
	rawDB, err := sql.Open("mysql", cfg.ServerDSN())
	if err != nil {
		return err
	}
	defer rawDB.Close()

	if err := rawDB.Ping(); err != nil {
		return fmt.Errorf("cannot ping MariaDB server: %w", err)
	}

	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", cfg.DBName)
	_, err = rawDB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to execute create database query: %w", err)
	}

	return nil
}
