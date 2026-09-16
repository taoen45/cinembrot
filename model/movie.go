package model

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// Movie represents comprehensive cinema/movie information
type Movie struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	Slug               string         `gorm:"uniqueIndex;size:255;not null" json:"slug"`
	Title              string         `gorm:"size:255;not null;index" json:"title"`
	OriginalTitle      string         `gorm:"size:255" json:"original_title,omitempty"`
	AlternativeTitles  string         `gorm:"size:500" json:"alternative_titles,omitempty"`
	Type               string         `gorm:"size:50;default:'movie';index" json:"type"`     // movie, series, anime, documentary, tv_show
	Status             string         `gorm:"size:50;default:'released';index" json:"status"` // ongoing, completed, released, upcoming
	Tagline            string         `gorm:"size:500" json:"tagline,omitempty"`
	Synopsis           string         `gorm:"type:longtext" json:"synopsis,omitempty"` // Default Indonesian Synopsis
	SynopsisEN         string         `gorm:"type:longtext" json:"synopsis_en,omitempty"` // English Synopsis
	ReleaseDate        *time.Time     `json:"release_date,omitempty"`
	Year               int            `gorm:"index" json:"year,omitempty"`
	DurationMinutes    int            `json:"duration_minutes,omitempty"`
	DurationFormatted  string         `gorm:"size:50" json:"duration_formatted,omitempty"` // e.g. "2h 15m" / "135 min"
	Country            string         `gorm:"size:100;index" json:"country,omitempty"`
	Language           string         `gorm:"size:100" json:"language,omitempty"`
	AgeRating          string         `gorm:"size:50" json:"age_rating,omitempty"` // PG-13, R, 13+, 18+, SU
	Quality            string         `gorm:"size:50" json:"quality,omitempty"`    // HD, CAM, WEB-DL, BluRay, 4K UHD
	
	// Licensing, Legality & Free Access
	IsLegal            bool           `gorm:"default:true;index" json:"is_legal"`               // Menandakan film legal didistribusikan
	IsFree             bool           `gorm:"default:false;index" json:"is_free"`              // Gratis ditonton / didownload
	LicenseType        string         `gorm:"size:100;index" json:"license_type,omitempty"`    // Public Domain, Creative Commons, Commercial/Copyrighted, dll
	LicenseName        string         `gorm:"size:255" json:"license_name,omitempty"`           // e.g. "Creative Commons Attribution 3.0 (CC BY 3.0)", "Public Domain Mark 1.0"
	LicenseURL         string         `gorm:"size:500" json:"license_url,omitempty"`            // e.g. "https://creativecommons.org/licenses/by/3.0/"

	// Ratings
	IMDbRating         float64        `json:"imdb_rating,omitempty"`
	IMDbVotes          int            `json:"imdb_votes,omitempty"`
	TMDbRating         float64        `json:"tmdb_rating,omitempty"`
	Rating             float64        `gorm:"index" json:"rating,omitempty"`
	VoteCount          int            `json:"vote_count,omitempty"`
	Popularity         float64        `json:"popularity,omitempty"`
	Views              int64          `gorm:"default:0" json:"views"`

	// Media Assets (WebP Original & WebP Thumbnails)
	PosterURL          string         `gorm:"size:1000" json:"poster_url,omitempty"`
	PosterThumbURL     string         `gorm:"size:1000" json:"poster_thumb_url,omitempty"`
	BackdropURL        string         `gorm:"size:1000" json:"backdrop_url,omitempty"`
	BackdropThumbURL   string         `gorm:"size:1000" json:"backdrop_thumb_url,omitempty"`
	ThumbnailURL       string         `gorm:"size:1000" json:"thumbnail_url,omitempty"`
	TrailerURL         string         `gorm:"size:1000" json:"trailer_url,omitempty"`

	// Source tracking & CMS Manual Override Lock
	SourceWebsite      string         `gorm:"size:100;index" json:"source_website"`
	SourceURL          string         `gorm:"uniqueIndex;size:500;not null" json:"source_url"`
	IsManualEdit       bool           `gorm:"default:false;index" json:"is_manual_edit"` // True if edited by admin in CMS (scraper will not overwrite)

	// Relationships
	Genres             []Genre        `gorm:"many2many:movie_genres;" json:"genres,omitempty"`
	Directors          []Director     `gorm:"many2many:movie_directors;" json:"directors,omitempty"`
	Actors             []Actor        `gorm:"many2many:movie_actors;" json:"actors,omitempty"`
	Episodes           []Episode      `gorm:"foreignKey:MovieID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"episodes,omitempty"`
	DownloadLinks      []DownloadLink `gorm:"foreignKey:MovieID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"download_links,omitempty"`
	StreamLinks        []StreamLink   `gorm:"foreignKey:MovieID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"stream_links,omitempty"`
	Schedules          []Schedule     `gorm:"foreignKey:MovieID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"schedules,omitempty"`
	Comments           []Comment      `gorm:"foreignKey:MovieID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"comments,omitempty"`

	// Raw extra metadata for custom scrapers (JSON format)
	RawMetadata        string         `gorm:"type:longtext" json:"raw_metadata,omitempty"`

	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

// Genre represents movie categories
type Genre struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"uniqueIndex;size:100;not null" json:"name"`
	Slug      string         `gorm:"uniqueIndex;size:100;not null" json:"slug"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Director represents movie directors
type Director struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:255;not null;index" json:"name"`
	Slug      string         `gorm:"size:255;index" json:"slug,omitempty"`
	PhotoURL  string         `gorm:"size:1000" json:"photo_url,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Actor represents movie cast members
type Actor struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Name          string         `gorm:"size:255;not null;index" json:"name"`
	Slug          string         `gorm:"size:255;index" json:"slug,omitempty"`
	CharacterName string         `gorm:"size:255" json:"character_name,omitempty"`
	PhotoURL      string         `gorm:"size:1000" json:"photo_url,omitempty"`
	PhotoThumbURL string         `gorm:"size:1000" json:"photo_thumb_url,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// Episode represents episodes in a TV/Drama/Anime series
type Episode struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	MovieID       uint           `gorm:"not null;index" json:"movie_id"`
	SeasonNumber  int            `gorm:"default:1;index" json:"season_number"`
	EpisodeNumber int            `gorm:"not null;index" json:"episode_number"`
	Title         string         `gorm:"size:255" json:"title,omitempty"`
	Slug          string         `gorm:"size:255" json:"slug,omitempty"`
	Synopsis      string         `gorm:"type:text" json:"synopsis,omitempty"`
	Duration      string         `gorm:"size:50" json:"duration,omitempty"`
	ReleaseDate   *time.Time     `json:"release_date,omitempty"`
	StreamURL     string         `gorm:"size:1000" json:"stream_url,omitempty"`
	SourceURL     string         `gorm:"size:500" json:"source_url,omitempty"`
	
	DownloadLinks []DownloadLink `gorm:"foreignKey:EpisodeID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"download_links,omitempty"`
	StreamLinks   []StreamLink   `gorm:"foreignKey:EpisodeID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"stream_links,omitempty"`

	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// DownloadLink represents downloadable links for movies or episodes
type DownloadLink struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	MovieID        uint           `gorm:"not null;index" json:"movie_id"`
	EpisodeID      *uint          `gorm:"index" json:"episode_id,omitempty"`
	Provider       string         `gorm:"size:100;index" json:"provider"`        // Google Drive, Mega, Mediafire, Archive.org, Blender, etc.
	Quality        string         `gorm:"size:50" json:"quality,omitempty"`       // 360p, 480p, 720p, 1080p, 4K
	Resolution     string         `gorm:"size:50" json:"resolution,omitempty"`    // HD, FHD, UHD
	FileSize       string         `gorm:"size:50" json:"file_size,omitempty"`     // 500MB, 1.2GB
	Format         string         `gorm:"size:50" json:"format,omitempty"`        // MP4, MKV, AVI
	URL            string         `gorm:"size:1000;not null" json:"url"`
	Password       string         `gorm:"size:100" json:"password,omitempty"`
	
	// Link Validation & Health Checking
	IsValid        bool           `gorm:"default:true;index" json:"is_valid"`
	Status         string         `gorm:"size:50;default:'ACTIVE';index" json:"status"` // ACTIVE, DEAD, CHECKING
	HTTPStatus     int            `json:"http_status,omitempty"`
	ResponseTimeMs int64          `json:"response_time_ms,omitempty"`
	LastCheckedAt  *time.Time     `json:"last_checked_at,omitempty"`

	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// IsTorrent checks if a download link is a raw torrent or magnet link
func (d DownloadLink) IsTorrent() bool {
	u := strings.ToLower(strings.TrimSpace(d.URL))
	p := strings.ToLower(strings.TrimSpace(d.Provider))
	return strings.HasPrefix(u, "magnet:") || strings.HasSuffix(u, ".torrent") ||
		strings.Contains(u, "/torrent/download/") ||
		strings.Contains(p, "torrent") || strings.Contains(p, "magnet") || strings.Contains(p, "yts")
}

// IsMatang checks if a download link is a processed video file (Hardsub Indonesia / Direct Video MP4)
func (d DownloadLink) IsMatang() bool {
	return !d.IsTorrent()
}

// MatangDownloads returns only the completed / direct / hardsubbed download links
func (m Movie) MatangDownloads() []DownloadLink {
	var list []DownloadLink
	for _, dl := range m.DownloadLinks {
		if dl.IsMatang() {
			list = append(list, dl)
		}
	}
	return list
}

// TorrentDownloads returns only raw torrent and magnet links
func (m Movie) TorrentDownloads() []DownloadLink {
	var list []DownloadLink
	for _, dl := range m.DownloadLinks {
		if dl.IsTorrent() {
			list = append(list, dl)
		}
	}
	return list
}

// StreamLink represents direct streaming / embed player links
type StreamLink struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	MovieID        uint           `gorm:"not null;index" json:"movie_id"`
	EpisodeID      *uint          `gorm:"index" json:"episode_id,omitempty"`
	Provider       string         `gorm:"size:100;index" json:"provider"`        // Streamtape, Hydrax, Doodstream, Vidcloud, Mixdrop, etc.
	ServerName     string         `gorm:"size:100" json:"server_name,omitempty"` // Server 1, Fast Server, VIP
	Quality        string         `gorm:"size:50" json:"quality,omitempty"`       // 720p, 1080p, Auto
	EmbedURL       string         `gorm:"size:1000;not null" json:"embed_url"`
	DirectURL      string         `gorm:"size:1000" json:"direct_url,omitempty"`
	
	// Validation
	IsValid        bool           `gorm:"default:true;index" json:"is_valid"`
	Status         string         `gorm:"size:50;default:'ACTIVE';index" json:"status"` // ACTIVE, DEAD
	LastCheckedAt  *time.Time     `json:"last_checked_at,omitempty"`

	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// Schedule represents cinema schedule/showtimes
type Schedule struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	MovieID     uint           `gorm:"not null;index" json:"movie_id"`
	CinemaChain string         `gorm:"size:100;index" json:"cinema_chain"`    // Cinema XXI, CGV, Cinepolis, FLIX, etc.
	CinemaName  string         `gorm:"size:255;not null;index" json:"cinema_name"`
	City        string         `gorm:"size:100;index" json:"city"`
	Address     string         `gorm:"size:500" json:"address,omitempty"`
	HallType    string         `gorm:"size:100" json:"hall_type,omitempty"`    // Regular, 2D, 3D, IMAX, Premiere, Velvet, 4DX
	ShowDate    string         `gorm:"size:20;index" json:"show_date"`         // YYYY-MM-DD
	Showtimes   string         `gorm:"type:text;not null" json:"showtimes"`    // "12:00, 14:30, 17:00, 19:30" or JSON
	Price       string         `gorm:"size:100" json:"price,omitempty"`        // Rp 40.000 - Rp 50.000
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// ScrapeLog keeps track of scraper execution history and health
type ScrapeLog struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	SourceWebsite  string         `gorm:"size:100;index" json:"source_website"`
	TargetURL      string         `gorm:"size:500;not null" json:"target_url"`
	Status         string         `gorm:"size:50;index" json:"status"` // SUCCESS, FAILED, RUNNING, PARTIAL
	ItemsScraped   int            `json:"items_scraped"`
	ErrorMessage   string         `gorm:"type:text" json:"error_message,omitempty"`
	ExecutionTimeMs int64         `json:"execution_time_ms"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// Comment represents user comments and reviews for a movie
type Comment struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	MovieID        uint           `gorm:"not null;index" json:"movie_id"`
	AuthorName     string         `gorm:"size:150;not null" json:"author_name"`
	AuthorEmail    string         `gorm:"size:255" json:"author_email,omitempty"`
	Content        string         `gorm:"type:text;not null" json:"content"`
	Rating         float64        `gorm:"default:10" json:"rating,omitempty"` // User rating (e.g. 1 - 10)
	IsApproved     bool           `gorm:"default:true;index" json:"is_approved"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// User represents CMS Admin & Editor accounts
type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"uniqueIndex;size:100;not null" json:"username"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	FullName     string         `gorm:"size:150" json:"full_name"`
	Role         string         `gorm:"size:50;default:'admin'" json:"role"` // admin, editor
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	LastLoginAt  *time.Time     `json:"last_login_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// ScrapeSource represents website scraping sources stored in MariaDB
type ScrapeSource struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	Name            string         `gorm:"size:150;not null" json:"name"`            // e.g. "The Movie Database (TMDb)"
	Code            string         `gorm:"uniqueIndex;size:50;not null" json:"code"` // e.g. "tmdb", "archive", "blender", "filmapik"
	BaseURL         string         `gorm:"size:500;not null" json:"base_url"`        // e.g. "https://api.themoviedb.org/3"
	APIKey          string         `gorm:"size:255" json:"api_key,omitempty"`        // Optional API key / token
	Type            string         `gorm:"size:50;default:'api'" json:"type"`        // "api", "html_scrape", "rss_feed", "json_feed"
	Category        string         `gorm:"size:100;default:'General'" json:"category"`// "Metadata", "Public Domain", "Creative Commons", "Third-Party"
	Description     string         `gorm:"type:text" json:"description,omitempty"`
	IsActive        bool           `gorm:"default:true;index" json:"is_active"`
	RateLimitPerSec int            `gorm:"default:5" json:"rate_limit_per_sec"`
	TotalScraped    int64          `gorm:"default:0" json:"total_scraped"`
	LastScrapedAt   *time.Time     `json:"last_scraped_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// SystemSetting represents dynamic system and scheduler configurations stored in MariaDB
type SystemSetting struct {
	Key         string    `gorm:"primaryKey;size:100" json:"key"`
	Value       string    `gorm:"type:text;not null" json:"value"`
	Description string    `gorm:"size:255" json:"description,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SchedulerConfig holds the structured settings for the auto-scraper
type SchedulerConfig struct {
	Enabled         bool
	IntervalMinutes int
	StartYear       int
	EndYear         int
	PagesPerYear    int
	DelayMs         int
}

// SubtitleOption represents one subtitle candidate for a movie review
type SubtitleOption struct {
	ID        string `json:"id"`
	Title     string `json:"title"`    // e.g. "Subtitle ID #1 - YTS Official (Bong Joon-ho)"
	Source    string `json:"source"`   // e.g. "YIFY / YTS", "SubDL", "OpenSubtitles", "Upload Manual"
	Language  string `json:"language"` // e.g. "Indonesian"
	SRTPath   string `json:"srt_path"` // Local filesystem path to .srt
	VTTPath   string `json:"vtt_path"` // Web accessible URL path to .vtt
	IsDefault bool   `json:"is_default"`
}

// TorrentTask tracks the background download, subtitle selection, and hardsub pipeline
type TorrentTask struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	MovieID            uint           `gorm:"index;not null" json:"movie_id"`
	MovieTitle         string         `gorm:"size:255;not null" json:"movie_title"`
	MovieSlug          string         `gorm:"size:255;not null" json:"movie_slug"`
	MoviePoster        string         `gorm:"size:500" json:"movie_poster"`
	TorrentURL         string         `gorm:"type:text;not null" json:"torrent_url"`
	Quality            string         `gorm:"size:100" json:"quality"`
	Status             string         `gorm:"size:50;default:'PENDING';index" json:"status"` // PENDING, DOWNLOADING, DOWNLOADED, HARDSUBBING, COMPLETED, FAILED, CANCELLED
	ProgressPercent    float64        `gorm:"default:0" json:"progress_percent"`
	DownloadedBytes    int64          `gorm:"default:0" json:"downloaded_bytes"`
	TotalBytes         int64          `gorm:"default:0" json:"total_bytes"`
	DownloadSpeedMBs   float64        `gorm:"column:download_speed_mbs;default:0" json:"download_speed_mbs"`
	PeersCount         int            `gorm:"default:0" json:"peers_count"`
	VideoFilePath      string         `gorm:"type:text" json:"video_file_path"`
	VideoWebURL        string         `gorm:"type:text" json:"video_web_url"`
	AvailableSubtitles string         `gorm:"type:text" json:"available_subtitles"` // JSON array of SubtitleOption
	SelectedSubtitleID string         `gorm:"size:100" json:"selected_subtitle_id"`
	HardsubFilePath    string         `gorm:"type:text" json:"hardsub_file_path"`
	HardsubWebURL      string         `gorm:"type:text" json:"hardsub_web_url"`
	ErrorMessage       string         `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}
