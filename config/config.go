package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configurations
type Config struct {
	// Web Server Configuration
	ServerPort string

	// MariaDB Database Configuration
	DBHost string
	DBPort string
	DBUser string
	DBPass string
	DBName string

	// Scraper Engine & Rate Limiter Settings (Polite Mode)
	ScraperUserAgent  string
	ScraperMaxThreads int
	ScraperDelayMs    int
	ScraperTimeoutSec int

	// Automatic Background Scraping Settings
	AutoScrapeEnabled         bool
	AutoScrapeIntervalMinutes int
	AutoScrapeDelayMs         int
	AutoScrapeStartYear       int
	AutoScrapeEndYear         int
	AutoScrapePagesPerYear    int

	// External APIs for Rich Metadata
	TMDBAPIKey   string
	TMDBLanguage string
	OMDBAPIKey   string

	// CMS Admin Configuration
	AdminDefaultUser  string
	AdminDefaultPass  string

	// Feature Toggles (Comments, Reviews, Ads)
	EnableComments    bool
	EnableAds         bool
	AdsterraPopunder  string
	AdsterraSocialBar string
	AdsterraBanner728 string
	AdsterraBanner300 string
	AdsterraNative    string

	// PPD Video Hosters (DoodStream & Streamtape)
	DoodStreamAPIKey string
	StreamtapeLogin  string
	StreamtapeKey    string
	AutoHosterUpload bool
}

// LoadConfig loads configuration from .env and environment variables
func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("[INFO] No .env file found or error reading it, using environment variables or defaults")
	}

	return &Config{
		ServerPort:        getEnv("PORT", "8080"),
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnv("DB_PORT", "3306"),
		DBUser:            getEnv("DB_USER", "root"),
		DBPass:            getEnv("DB_PASS", "gogogogo"),
		DBName:            getEnv("DB_NAME", "CINEMBROT"),
		ScraperUserAgent:  getEnv("SCRAPER_USER_AGENT", "CINEMBROTMovieBot/1.0 (+https://github.com/CINEMBROT/bot; contact: admin@CINEMBROT.local) Mozilla/5.0"),
		ScraperMaxThreads: getEnvAsInt("SCRAPER_MAX_CONCURRENCY", 1),
		ScraperDelayMs:    getEnvAsInt("SCRAPER_DELAY_MS", 2000),
		ScraperTimeoutSec: getEnvAsInt("SCRAPER_TIMEOUT_SEC", 30),

		// Auto Scrape Scheduler Defaults (Polite: delay 2.5s between movies, runs every 6 hours)
		AutoScrapeEnabled:         getEnvAsBool("AUTO_SCRAPE_ENABLED", true),
		AutoScrapeIntervalMinutes: getEnvAsInt("AUTO_SCRAPE_INTERVAL_MINUTES", 360), // Every 6 hours
		AutoScrapeDelayMs:         getEnvAsInt("AUTO_SCRAPE_DELAY_MS", 2500),        // 2.5s polite delay per movie
		AutoScrapeStartYear:       getEnvAsInt("AUTO_SCRAPE_START_YEAR", 2024),
		AutoScrapeEndYear:         getEnvAsInt("AUTO_SCRAPE_END_YEAR", 1990),
		AutoScrapePagesPerYear:    getEnvAsInt("AUTO_SCRAPE_PAGES_PER_YEAR", 2),

		TMDBAPIKey:   getEnv("TMDB_API_KEY", "fb7bb23f03b6994dafc674c074d01761"),
		TMDBLanguage: getEnv("TMDB_LANGUAGE", "id-ID"),
		OMDBAPIKey:   getEnv("OMDB_API_KEY", "4b447405"),

		AdminDefaultUser:  getEnv("ADMIN_USER", "admin"),
		AdminDefaultPass:  getEnv("ADMIN_PASS", "cinembrot123"),

		EnableComments:    getEnvAsBool("ENABLE_COMMENTS", true),
		EnableAds:         getEnvAsBool("ENABLE_ADS", true),
		AdsterraPopunder:  getEnv("ADSTERRA_POPUNDER_KEY", ""),
		AdsterraSocialBar: getEnv("ADSTERRA_SOCIAL_BAR_KEY", ""),
		AdsterraBanner728: getEnv("ADSTERRA_BANNER_728_KEY", ""),
		AdsterraBanner300: getEnv("ADSTERRA_BANNER_300_KEY", ""),
		AdsterraNative:    getEnv("ADSTERRA_NATIVE_KEY", ""),

		// Video Hosters Defaults
		DoodStreamAPIKey: getEnv("DOODSTREAM_API_KEY", "579834sz4bib8d70d8741z"),
		StreamtapeLogin:  getEnv("STREAMTAPE_LOGIN", "5325cb880d00ecec2186"),
		StreamtapeKey:    getEnv("STREAMTAPE_KEY", ""),
		AutoHosterUpload: getEnvAsBool("AUTO_HOSTER_UPLOAD", true),
	}
}

// DSN returns the MySQL/MariaDB data source name with database selected
func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s",
		c.DBUser, c.DBPass, c.DBHost, c.DBPort, c.DBName)
}

// ServerDSN returns the MySQL/MariaDB data source name without database selected
func (c *Config) ServerDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s",
		c.DBUser, c.DBPass, c.DBHost, c.DBPort)
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvAsBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			return boolVal
		}
	}
	return defaultVal
}
