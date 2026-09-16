package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"cinembrot/auth"
	"cinembrot/database"
	"cinembrot/model"
	"cinembrot/pipeline"
	"cinembrot/scheduler"
	"cinembrot/scraper"
	"cinembrot/validator"
)

type AdminPageData struct {
	Title          string
	ActiveMenu     string
	User           *model.User
	ErrorMsg       string
	SuccessMsg     string
	Stats          map[string]int64
	FilterStats    map[string]int64
	Movies         []model.Movie
	Movie          *model.Movie
	Comments       []model.Comment
	Users          []model.User
	Genres         []model.Genre
	Logs           []model.ScrapeLog
	Sources        []model.ScrapeSource
	Source         *model.ScrapeSource
	Years          []int
	SearchQuery    string
	CurrentFilter  string
	CurrentYear    int
	CurrentLicense string
	CurrentSource  string
	CurrentPage    int
	TotalPages      int
	TotalCount      int64
	SchedulerConfig model.SchedulerConfig
	Tasks           []model.TorrentTask
	Task            *model.TorrentTask
	Subtitles       []model.SubtitleOption
	DownloadPath      string
	ShowTorrentPublic bool
	// TypeContext membedakan konteks halaman: "anime", "drama_pendek", atau "" (semua film)
	TypeContext       string
	Settings          map[string]string
}

// HandleAdminLogin displays and processes the login form
func (s *Server) HandleAdminLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// If already logged in, redirect to dashboard
		if user := s.GetLoggedInUser(r); user != nil {
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}

		settings := database.GetAllSettings(s.db)

		data := AdminPageData{
			Title:    "Login CMS Admin - CINEMBROT",
			Settings: settings,
		}
		if r.URL.Query().Get("error") == "invalid" {
			data.ErrorMsg = "Username atau password salah!"
		} else if r.URL.Query().Get("error") == "turnstile" {
			data.ErrorMsg = "Verifikasi Cloudflare Turnstile gagal atau belum dicentang! Silakan coba lagi."
		}
		s.RenderHTML(w, "admin_login.html", "", data)
		return
	}

	// POST Login
	_ = r.ParseForm()
	username := strings.TrimSpace(r.FormValue("username"))
	password := strings.TrimSpace(r.FormValue("password"))
	redirectTo := r.FormValue("redirect")
	if redirectTo == "" {
		redirectTo = "/admin"
	}

	// 1. Verifikasi Cloudflare Turnstile jika diaktifkan
	turnstileEnabled := database.GetSetting(s.db, "turnstile_enabled", "true") == "true"
	turnstileSecretKey := database.GetSetting(s.db, "turnstile_secret_key", "0x4AAAAAAAE4maSz0Gah94IJ72HChDjUsDyc")

	if turnstileEnabled && turnstileSecretKey != "" {
		turnstileToken := r.FormValue("cf-turnstile-response")
		clientIP := r.Header.Get("CF-Connecting-IP")
		if clientIP == "" {
			clientIP = strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
		}
		if clientIP == "" {
			clientIP = r.RemoteAddr
		}
		valid, err := VerifyTurnstile(turnstileSecretKey, turnstileToken, clientIP)
		if err != nil || !valid {
			log.Printf("[ADMIN AUTH] ⚠️ Verifikasi Cloudflare Turnstile gagal untuk IP %s: %v\n", clientIP, err)
			http.Redirect(w, r, "/admin/login?error=turnstile", http.StatusSeeOther)
			return
		}
	}

	var user model.User
	err := s.db.Where("username = ? AND is_active = ?", username, true).First(&user).Error
	if err != nil {
		// Fallback check against default config if no user exists in DB
		if username == s.cfg.AdminDefaultUser && password == s.cfg.AdminDefaultPass {
			token := auth.GenerateSessionToken(username)
			http.SetCookie(w, &http.Cookie{
				Name:     auth.CookieSessionName,
				Value:    token,
				Path:     "/",
				HttpOnly: true,
				MaxAge:   int(auth.SessionDuration.Seconds()),
			})
			http.Redirect(w, r, redirectTo, http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/admin/login?error=invalid", http.StatusSeeOther)
		return
	}

	// Check password hash (or fallback match for default admin credentials)
	if !auth.CheckPasswordHash(password, user.PasswordHash) {
		pLower := strings.ToLower(strings.TrimSpace(password))
		if username == s.cfg.AdminDefaultUser && (pLower == "cinembrot123" || pLower == "cinebrot123" || password == s.cfg.AdminDefaultPass) {
			// Update hash to current salt format
			s.db.Model(&user).Update("password_hash", auth.HashPassword(password))
		} else {
			http.Redirect(w, r, "/admin/login?error=invalid", http.StatusSeeOther)
			return
		}
	}

	// Update last login
	now := time.Now()
	s.db.Model(&user).Update("last_login_at", &now)

	token := auth.GenerateSessionToken(user.Username)
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieSessionName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(auth.SessionDuration.Seconds()),
	})

	http.Redirect(w, r, redirectTo, http.StatusSeeOther)
}

// HandleAdminLogout logs out the user
func (s *Server) HandleAdminLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieSessionName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

// HandleAdminDashboard renders CMS overview & stats
func (s *Server) HandleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	user := s.GetLoggedInUser(r)

	stats := make(map[string]int64)
	var totalMovies, freeMovies, totalDownloads, activeDownloads, totalStreams, totalComments, totalUsers, totalSources, activeSources int64
	var totalAnime, totalDramaPendek int64

	s.db.Model(&model.Movie{}).Count(&totalMovies)
	s.db.Model(&model.Movie{}).Where("is_free = ?", true).Count(&freeMovies)
	s.db.Model(&model.DownloadLink{}).Count(&totalDownloads)
	s.db.Model(&model.DownloadLink{}).Where("is_valid = ?", true).Count(&activeDownloads)
	s.db.Model(&model.StreamLink{}).Count(&totalStreams)
	s.db.Model(&model.Comment{}).Count(&totalComments)
	s.db.Model(&model.User{}).Count(&totalUsers)
	s.db.Model(&model.ScrapeSource{}).Count(&totalSources)
	s.db.Model(&model.ScrapeSource{}).Where("is_active = ?", true).Count(&activeSources)
	s.db.Model(&model.Movie{}).Where("type = ?", "anime").Count(&totalAnime)
	s.db.Model(&model.Movie{}).Where("type = ?", "drama_pendek").Count(&totalDramaPendek)

	stats["total_movies"] = totalMovies
	stats["free_movies"] = freeMovies
	stats["total_downloads"] = totalDownloads
	stats["active_downloads"] = activeDownloads
	stats["total_streams"] = totalStreams
	stats["total_comments"] = totalComments
	stats["total_users"] = totalUsers
	stats["total_sources"] = totalSources
	stats["active_sources"] = activeSources
	stats["total_anime"] = totalAnime
	stats["total_drama_pendek"] = totalDramaPendek

	var recentMovies []model.Movie
	s.db.Preload("Genres").Order("id desc").Limit(6).Find(&recentMovies)

	var recentComments []model.Comment
	s.db.Order("id desc").Limit(5).Find(&recentComments)

	data := AdminPageData{
		Title:      "Dashboard CMS - CINEMBROT",
		ActiveMenu: "dashboard",
		User:       user,
		Stats:      stats,
		Movies:     recentMovies,
		Comments:   recentComments,
	}

	s.RenderHTML(w, "admin_dashboard.html", "admin_layout.html", data)
}

// HandleAdminHollywood menampilkan daftar konten bertipe 'hollywood' di CMS
func (s *Server) HandleAdminHollywood(w http.ResponseWriter, r *http.Request) {
	user := s.GetLoggedInUser(r)
	queryStr := strings.TrimSpace(r.URL.Query().Get("q"))
	filterStr := strings.TrimSpace(r.URL.Query().Get("filter"))
	yearStr := strings.TrimSpace(r.URL.Query().Get("year"))
	pageStr := r.URL.Query().Get("p")

	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	pageSize := 20
	offset := (page - 1) * pageSize

	hCond := "type = 'hollywood' OR (type = 'movie' AND (country LIKE '%United States%' OR country LIKE '%USA%' OR country LIKE '%UK%' OR country LIKE '%Amerika%' OR language LIKE '%English%' OR language = 'en'))"

	query := s.db.Model(&model.Movie{}).Preload("Genres").Preload("DownloadLinks").Preload("StreamLinks").
		Where(hCond)

	if queryStr != "" {
		query = query.Where("title LIKE ? OR original_title LIKE ? OR slug LIKE ?",
			"%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%")
	}

	currentYear := 0
	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil && y > 0 {
			query = query.Where("year = ?", y)
			currentYear = y
		}
	}

	// Status Filters
	switch filterStr {
	case "no_download":
		query = query.Where("NOT EXISTS (SELECT 1 FROM download_links WHERE download_links.movie_id = movies.id AND download_links.deleted_at IS NULL)")
	case "has_download":
		query = query.Where("EXISTS (SELECT 1 FROM download_links WHERE download_links.movie_id = movies.id AND download_links.deleted_at IS NULL)")
	case "no_synopsis":
		query = query.Where("synopsis IS NULL OR synopsis = '' OR TRIM(synopsis) = ''")
	case "has_synopsis":
		query = query.Where("synopsis IS NOT NULL AND synopsis <> ''")
	case "no_stream":
		query = query.Where("NOT EXISTS (SELECT 1 FROM stream_links WHERE stream_links.movie_id = movies.id AND stream_links.deleted_at IS NULL)")
	case "manual_edit":
		query = query.Where("is_manual_edit = ?", true)
	}

	var totalCount int64
	query.Count(&totalCount)

	var movies []model.Movie
	query.Order("id desc").Offset(offset).Limit(pageSize).Find(&movies)

	totalPages := int((totalCount + int64(pageSize) - 1) / int64(pageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	// Filter Stats
	filterStats := make(map[string]int64)
	var countAll, countNoDL, countNoSyn, countNoStream, countManual int64
	s.db.Model(&model.Movie{}).Where(hCond).Count(&countAll)
	s.db.Model(&model.Movie{}).Where("("+hCond+") AND NOT EXISTS (SELECT 1 FROM download_links WHERE download_links.movie_id = movies.id AND download_links.deleted_at IS NULL)").Count(&countNoDL)
	s.db.Model(&model.Movie{}).Where("("+hCond+") AND (synopsis IS NULL OR synopsis = '' OR TRIM(synopsis) = '')").Count(&countNoSyn)
	s.db.Model(&model.Movie{}).Where("("+hCond+") AND NOT EXISTS (SELECT 1 FROM stream_links WHERE stream_links.movie_id = movies.id AND stream_links.deleted_at IS NULL)").Count(&countNoStream)
	s.db.Model(&model.Movie{}).Where("("+hCond+") AND is_manual_edit = ?", true).Count(&countManual)
	filterStats["all"] = countAll
	filterStats["no_download"] = countNoDL
	filterStats["no_synopsis"] = countNoSyn
	filterStats["no_stream"] = countNoStream
	filterStats["manual_edit"] = countManual

	var years []int
	s.db.Model(&model.Movie{}).Where(hCond).Distinct().Order("year desc").Pluck("year", &years)

	data := AdminPageData{
		Title:         "Kelola Hollywood & Box Office - CMS CINEMBROT",
		ActiveMenu:    "hollywood",
		User:          user,
		Movies:        movies,
		SearchQuery:   queryStr,
		CurrentFilter: filterStr,
		CurrentYear:   currentYear,
		FilterStats:   filterStats,
		Years:         years,
		CurrentPage:   page,
		TotalPages:    totalPages,
		TotalCount:    totalCount,
		SuccessMsg:    r.URL.Query().Get("success"),
		TypeContext:   "hollywood",
	}

	s.RenderHTML(w, "admin_movies.html", "admin_layout.html", data)
}

// HandleAdminAnime menampilkan daftar konten bertipe 'anime' di CMS
func (s *Server) HandleAdminAnime(w http.ResponseWriter, r *http.Request) {
	user := s.GetLoggedInUser(r)
	queryStr := strings.TrimSpace(r.URL.Query().Get("q"))
	filterStr := strings.TrimSpace(r.URL.Query().Get("filter"))
	yearStr := strings.TrimSpace(r.URL.Query().Get("year"))
	pageStr := r.URL.Query().Get("p")

	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	pageSize := 20
	offset := (page - 1) * pageSize

	// Filter hanya konten bertipe anime
	query := s.db.Model(&model.Movie{}).Preload("Genres").Preload("DownloadLinks").Preload("StreamLinks").
		Where("type = ?", "anime")

	if queryStr != "" {
		query = query.Where("title LIKE ? OR original_title LIKE ? OR slug LIKE ?",
			"%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%")
	}

	currentYear := 0
	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil && y > 0 {
			query = query.Where("year = ?", y)
			currentYear = y
		}
	}

	// Status Filters
	switch filterStr {
	case "no_download":
		query = query.Where("NOT EXISTS (SELECT 1 FROM download_links WHERE download_links.movie_id = movies.id AND download_links.deleted_at IS NULL)")
	case "has_download":
		query = query.Where("EXISTS (SELECT 1 FROM download_links WHERE download_links.movie_id = movies.id AND download_links.deleted_at IS NULL)")
	case "no_synopsis":
		query = query.Where("synopsis IS NULL OR synopsis = '' OR TRIM(synopsis) = ''")
	case "has_synopsis":
		query = query.Where("synopsis IS NOT NULL AND synopsis <> ''")
	case "no_stream":
		query = query.Where("NOT EXISTS (SELECT 1 FROM stream_links WHERE stream_links.movie_id = movies.id AND stream_links.deleted_at IS NULL)")
	case "manual_edit":
		query = query.Where("is_manual_edit = ?", true)
	}

	var totalCount int64
	query.Count(&totalCount)

	var movies []model.Movie
	query.Order("id desc").Offset(offset).Limit(pageSize).Find(&movies)

	totalPages := int((totalCount + int64(pageSize) - 1) / int64(pageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	// Filter Stats (scope ke type=anime)
	filterStats := make(map[string]int64)
	var countAll, countNoDL, countNoSyn, countNoStream, countManual int64
	s.db.Model(&model.Movie{}).Where("type = ?", "anime").Count(&countAll)
	s.db.Model(&model.Movie{}).Where("type = ? AND NOT EXISTS (SELECT 1 FROM download_links WHERE download_links.movie_id = movies.id AND download_links.deleted_at IS NULL)", "anime").Count(&countNoDL)
	s.db.Model(&model.Movie{}).Where("type = ? AND (synopsis IS NULL OR synopsis = '' OR TRIM(synopsis) = '')", "anime").Count(&countNoSyn)
	s.db.Model(&model.Movie{}).Where("type = ? AND NOT EXISTS (SELECT 1 FROM stream_links WHERE stream_links.movie_id = movies.id AND stream_links.deleted_at IS NULL)", "anime").Count(&countNoStream)
	s.db.Model(&model.Movie{}).Where("type = ? AND is_manual_edit = ?", "anime", true).Count(&countManual)
	filterStats["all"] = countAll
	filterStats["no_download"] = countNoDL
	filterStats["no_synopsis"] = countNoSyn
	filterStats["no_stream"] = countNoStream
	filterStats["manual_edit"] = countManual

	var years []int
	s.db.Model(&model.Movie{}).Where("type = ?", "anime").Distinct().Order("year desc").Pluck("year", &years)

	data := AdminPageData{
		Title:         "Kelola Anime - CMS CINEMBROT",
		ActiveMenu:    "anime",
		User:          user,
		Movies:        movies,
		SearchQuery:   queryStr,
		CurrentFilter: filterStr,
		CurrentYear:   currentYear,
		FilterStats:   filterStats,
		Years:         years,
		CurrentPage:   page,
		TotalPages:    totalPages,
		TotalCount:    totalCount,
		SuccessMsg:    r.URL.Query().Get("success"),
		TypeContext:   "anime",
	}

	s.RenderHTML(w, "admin_movies.html", "admin_layout.html", data)
}

// HandleAdminDramaPendek menampilkan daftar konten bertipe 'drama_pendek' di CMS
func (s *Server) HandleAdminDramaPendek(w http.ResponseWriter, r *http.Request) {
	user := s.GetLoggedInUser(r)
	queryStr := strings.TrimSpace(r.URL.Query().Get("q"))
	filterStr := strings.TrimSpace(r.URL.Query().Get("filter"))
	yearStr := strings.TrimSpace(r.URL.Query().Get("year"))
	pageStr := r.URL.Query().Get("p")

	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	pageSize := 20
	offset := (page - 1) * pageSize

	// Filter hanya konten bertipe drama_pendek
	query := s.db.Model(&model.Movie{}).Preload("Genres").Preload("DownloadLinks").Preload("StreamLinks").
		Where("type = ?", "drama_pendek")

	if queryStr != "" {
		query = query.Where("title LIKE ? OR original_title LIKE ? OR slug LIKE ?",
			"%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%")
	}

	currentYear := 0
	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil && y > 0 {
			query = query.Where("year = ?", y)
			currentYear = y
		}
	}

	// Status Filters
	switch filterStr {
	case "no_download":
		query = query.Where("NOT EXISTS (SELECT 1 FROM download_links WHERE download_links.movie_id = movies.id AND download_links.deleted_at IS NULL)")
	case "has_download":
		query = query.Where("EXISTS (SELECT 1 FROM download_links WHERE download_links.movie_id = movies.id AND download_links.deleted_at IS NULL)")
	case "no_synopsis":
		query = query.Where("synopsis IS NULL OR synopsis = '' OR TRIM(synopsis) = ''")
	case "has_synopsis":
		query = query.Where("synopsis IS NOT NULL AND synopsis <> ''")
	case "no_stream":
		query = query.Where("NOT EXISTS (SELECT 1 FROM stream_links WHERE stream_links.movie_id = movies.id AND stream_links.deleted_at IS NULL)")
	case "manual_edit":
		query = query.Where("is_manual_edit = ?", true)
	}

	var totalCount int64
	query.Count(&totalCount)

	var movies []model.Movie
	query.Order("id desc").Offset(offset).Limit(pageSize).Find(&movies)

	totalPages := int((totalCount + int64(pageSize) - 1) / int64(pageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	// Filter Stats (scope ke type=drama_pendek)
	filterStats := make(map[string]int64)
	var countAll, countNoDL, countNoSyn, countNoStream, countManual int64
	s.db.Model(&model.Movie{}).Where("type = ?", "drama_pendek").Count(&countAll)
	s.db.Model(&model.Movie{}).Where("type = ? AND NOT EXISTS (SELECT 1 FROM download_links WHERE download_links.movie_id = movies.id AND download_links.deleted_at IS NULL)", "drama_pendek").Count(&countNoDL)
	s.db.Model(&model.Movie{}).Where("type = ? AND (synopsis IS NULL OR synopsis = '' OR TRIM(synopsis) = '')", "drama_pendek").Count(&countNoSyn)
	s.db.Model(&model.Movie{}).Where("type = ? AND NOT EXISTS (SELECT 1 FROM stream_links WHERE stream_links.movie_id = movies.id AND stream_links.deleted_at IS NULL)", "drama_pendek").Count(&countNoStream)
	s.db.Model(&model.Movie{}).Where("type = ? AND is_manual_edit = ?", "drama_pendek", true).Count(&countManual)
	filterStats["all"] = countAll
	filterStats["no_download"] = countNoDL
	filterStats["no_synopsis"] = countNoSyn
	filterStats["no_stream"] = countNoStream
	filterStats["manual_edit"] = countManual

	var years []int
	s.db.Model(&model.Movie{}).Where("type = ?", "drama_pendek").Distinct().Order("year desc").Pluck("year", &years)

	data := AdminPageData{
		Title:         "Kelola Drama Pendek - CMS CINEMBROT",
		ActiveMenu:    "drama_pendek",
		User:          user,
		Movies:        movies,
		SearchQuery:   queryStr,
		CurrentFilter: filterStr,
		CurrentYear:   currentYear,
		FilterStats:   filterStats,
		Years:         years,
		CurrentPage:   page,
		TotalPages:    totalPages,
		TotalCount:    totalCount,
		SuccessMsg:    r.URL.Query().Get("success"),
		TypeContext:   "drama_pendek",
	}

	s.RenderHTML(w, "admin_movies.html", "admin_layout.html", data)
}

// HandleAdminMovies renders the movies data table with search, status filters, and pagination
func (s *Server) HandleAdminMovies(w http.ResponseWriter, r *http.Request) {
	user := s.GetLoggedInUser(r)
	queryStr := strings.TrimSpace(r.URL.Query().Get("q"))
	filterStr := strings.TrimSpace(r.URL.Query().Get("filter"))
	yearStr := strings.TrimSpace(r.URL.Query().Get("year"))
	licenseStr := strings.TrimSpace(r.URL.Query().Get("license"))
	pageStr := r.URL.Query().Get("p")

	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	pageSize := 20
	offset := (page - 1) * pageSize

	query := s.db.Model(&model.Movie{}).Preload("Genres").Preload("DownloadLinks").Preload("StreamLinks")
	if queryStr != "" {
		query = query.Where("title LIKE ? OR original_title LIKE ? OR slug LIKE ?",
			"%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%")
	}

	currentYear := 0
	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil && y > 0 {
			query = query.Where("year = ?", y)
			currentYear = y
		}
	}

	if licenseStr != "" {
		if licenseStr == "free" {
			query = query.Where("is_free = ?", true)
		} else if licenseStr == "commercial" {
			query = query.Where("is_free = ?", false)
		}
	}

	// 🔍 Status Filters (Belum Ada Link Download, Belum Ada Sinopsis, dll)
	switch filterStr {
	case "no_download":
		// Film yang belum ada link download
		query = query.Where("NOT EXISTS (SELECT 1 FROM download_links WHERE download_links.movie_id = movies.id AND download_links.deleted_at IS NULL)")
	case "has_download":
		// Film yang sudah ada link download
		query = query.Where("EXISTS (SELECT 1 FROM download_links WHERE download_links.movie_id = movies.id AND download_links.deleted_at IS NULL)")
	case "no_synopsis":
		// Film yang belum ada sinopsis
		query = query.Where("synopsis IS NULL OR synopsis = '' OR TRIM(synopsis) = ''")
	case "has_synopsis":
		// Film yang sudah ada sinopsis
		query = query.Where("synopsis IS NOT NULL AND synopsis <> ''")
	case "no_stream":
		// Film yang belum ada server player
		query = query.Where("NOT EXISTS (SELECT 1 FROM stream_links WHERE stream_links.movie_id = movies.id AND stream_links.deleted_at IS NULL)")
	case "manual_edit":
		// Film yang diedit manual di CMS
		query = query.Where("is_manual_edit = ?", true)
	case "auto_scrape":
		// Film hasil auto-scraping
		query = query.Where("is_manual_edit = ?", false)
	}

	sourceStr := strings.TrimSpace(r.URL.Query().Get("source"))
	if sourceStr != "" {
		switch sourceStr {
		case "yts":
			query = query.Where("source_website LIKE ?", "%yts%")
		case "tmdb":
			query = query.Where("source_website LIKE ?", "%themoviedb%")
		case "archive":
			query = query.Where("source_website LIKE ?", "%archive.org%")
		case "blender":
			query = query.Where("source_website LIKE ?", "%blender%")
		case "publicdomain":
			query = query.Where("source_website LIKE ?", "%publicdomain%")
		case "manual":
			query = query.Where("source_website = '' OR source_website LIKE ? OR is_manual_edit = ?", "%CMS%", true)
		default:
			query = query.Where("source_website LIKE ?", "%"+sourceStr+"%")
		}
	}

	var totalCount int64
	query.Count(&totalCount)

	var movies []model.Movie
	query.Order("id desc").Offset(offset).Limit(pageSize).Find(&movies)

	totalPages := int((totalCount + int64(pageSize) - 1) / int64(pageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	// Hitung Ringkasan Statistik Filter Cepat
	filterStats := make(map[string]int64)
	var countAll, countNoDL, countNoSyn, countNoStream, countManual int64
	s.db.Model(&model.Movie{}).Count(&countAll)
	s.db.Model(&model.Movie{}).Where("NOT EXISTS (SELECT 1 FROM download_links WHERE download_links.movie_id = movies.id AND download_links.deleted_at IS NULL)").Count(&countNoDL)
	s.db.Model(&model.Movie{}).Where("synopsis IS NULL OR synopsis = '' OR TRIM(synopsis) = ''").Count(&countNoSyn)
	s.db.Model(&model.Movie{}).Where("NOT EXISTS (SELECT 1 FROM stream_links WHERE stream_links.movie_id = movies.id AND stream_links.deleted_at IS NULL)").Count(&countNoStream)
	s.db.Model(&model.Movie{}).Where("is_manual_edit = ?", true).Count(&countManual)

	filterStats["all"] = countAll
	filterStats["no_download"] = countNoDL
	filterStats["no_synopsis"] = countNoSyn
	filterStats["no_stream"] = countNoStream
	filterStats["manual_edit"] = countManual

	var years []int
	s.db.Model(&model.Movie{}).Distinct().Order("year desc").Pluck("year", &years)

	data := AdminPageData{
		Title:          "Kelola Film - CMS CINEMBROT",
		ActiveMenu:     "movies",
		User:           user,
		Movies:         movies,
		SearchQuery:    queryStr,
		CurrentFilter:  filterStr,
		CurrentYear:    currentYear,
		CurrentLicense: licenseStr,
		CurrentSource:  sourceStr,
		FilterStats:    filterStats,
		Years:          years,
		CurrentPage:    page,
		TotalPages:     totalPages,
		TotalCount:     totalCount,
		SuccessMsg:     r.URL.Query().Get("success"),
	}

	s.RenderHTML(w, "admin_movies.html", "admin_layout.html", data)
}

// HandleAdminMovieNew handles creating a new movie
func (s *Server) HandleAdminMovieNew(w http.ResponseWriter, r *http.Request) {
	user := s.GetLoggedInUser(r)

	if r.Method == http.MethodGet {
		var genres []model.Genre
		s.db.Find(&genres)

		// Mendukung pre-select tipe dari query URL: /admin/movies/new?type=anime
		typeCtx := r.URL.Query().Get("type")
		if typeCtx == "" {
			typeCtx = "movie"
		}

		// Tentukan ActiveMenu berdasarkan tipe
		activeMenu := "movies"
		if typeCtx == "anime" {
			activeMenu = "anime"
		} else if typeCtx == "drama_pendek" {
			activeMenu = "drama_pendek"
		}

		data := AdminPageData{
			Title:      "Tambah Konten Baru - CMS CINEMBROT",
			ActiveMenu: activeMenu,
			User:       user,
			Movie:      &model.Movie{Year: time.Now().Year(), Rating: 7.5, IsLegal: true, LicenseType: "Public Domain", Type: typeCtx},
			Genres:     genres,
			TypeContext: typeCtx,
		}
		s.RenderHTML(w, "admin_movie_form.html", "admin_layout.html", data)
		return
	}

	// POST Create
	_ = r.ParseForm()
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		http.Redirect(w, r, "/admin/movies/new?error=title_required", http.StatusSeeOther)
		return
	}

	slug := strings.TrimSpace(r.FormValue("slug"))
	if slug == "" {
		slug = strings.ToLower(strings.ReplaceAll(title, " ", "-"))
	}

	year, _ := strconv.Atoi(r.FormValue("year"))
	rating, _ := strconv.ParseFloat(r.FormValue("rating"), 64)
	duration, _ := strconv.Atoi(r.FormValue("duration_minutes"))
	isFree := r.FormValue("is_free") == "1" || r.FormValue("is_free") == "true"
	isLegal := r.FormValue("is_legal") == "1" || r.FormValue("is_legal") == "true"

	synopsis := scraper.CleanHTMLToPlainText(r.FormValue("synopsis"))

	// Baca tipe konten dari form (default: movie)
	movieType := strings.TrimSpace(r.FormValue("type"))
	if movieType == "" {
		movieType = "movie"
	}

	movie := model.Movie{
		Title:           title,
		OriginalTitle:   r.FormValue("original_title"),
		Slug:            slug,
		Year:            year,
		Rating:          rating,
		DurationMinutes: duration,
		Country:         r.FormValue("country"),
		Language:        r.FormValue("language"),
		Quality:         r.FormValue("quality"),
		TrailerURL:      r.FormValue("trailer_url"),
		PosterURL:       r.FormValue("poster_url"),
		BackdropURL:     r.FormValue("backdrop_url"),
		Synopsis:        synopsis,
		IsFree:          isFree,
		IsLegal:         isLegal,
		LicenseType:     r.FormValue("license_type"),
		Type:            movieType,
		IsManualEdit:    true,
		SourceURL:       "admin-manual-" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	if err := s.db.Create(&movie).Error; err != nil {
		log.Printf("[CMS ERROR] Failed to create movie: %v\n", err)
		http.Redirect(w, r, "/admin/movies/new?error=create_failed", http.StatusSeeOther)
		return
	}

	// Process download links
	s.saveMovieDownloadLinks(movie.ID, r)
	// Process stream links
	s.saveMovieStreamLinks(movie.ID, r)

	// Redirect ke halaman list yang sesuai tipe
	switch movieType {
	case "anime":
		http.Redirect(w, r, "/admin/anime?success=Anime+berhasil+ditambahkan", http.StatusSeeOther)
	case "drama_pendek":
		http.Redirect(w, r, "/admin/drama-pendek?success=Drama+Pendek+berhasil+ditambahkan", http.StatusSeeOther)
	default:
		http.Redirect(w, r, "/admin/movies?success=Film+berhasil+ditambahkan", http.StatusSeeOther)
	}
}

// HandleAdminMovieEdit handles editing an existing movie
func (s *Server) HandleAdminMovieEdit(w http.ResponseWriter, r *http.Request) {
	user := s.GetLoggedInUser(r)
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var movie model.Movie
	if err := s.db.Preload("Genres").Preload("DownloadLinks").Preload("StreamLinks").First(&movie, id).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodGet {
		var genres []model.Genre
		s.db.Find(&genres)

		data := AdminPageData{
			Title:      "Edit Film: " + movie.Title + " - CMS CINEMBROT",
			ActiveMenu: "movies",
			User:       user,
			Movie:      &movie,
			Genres:     genres,
			SuccessMsg: r.URL.Query().Get("success"),
		}
		s.RenderHTML(w, "admin_movie_form.html", "admin_layout.html", data)
		return
	}

	// POST Update
	_ = r.ParseForm()
	title := strings.TrimSpace(r.FormValue("title"))
	if title != "" {
		movie.Title = title
	}
	movie.OriginalTitle = r.FormValue("original_title")
	if slug := strings.TrimSpace(r.FormValue("slug")); slug != "" {
		movie.Slug = slug
	}
	if year, err := strconv.Atoi(r.FormValue("year")); err == nil && year > 0 {
		movie.Year = year
	}
	if rating, err := strconv.ParseFloat(r.FormValue("rating"), 64); err == nil {
		movie.Rating = rating
	}
	if duration, err := strconv.Atoi(r.FormValue("duration_minutes")); err == nil {
		movie.DurationMinutes = duration
	}
	movie.Country = r.FormValue("country")
	movie.Language = r.FormValue("language")
	movie.Quality = r.FormValue("quality")
	movie.TrailerURL = r.FormValue("trailer_url")
	movie.PosterURL = r.FormValue("poster_url")
	movie.BackdropURL = r.FormValue("backdrop_url")
	movie.Synopsis = scraper.CleanHTMLToPlainText(r.FormValue("synopsis"))
	movie.IsFree = r.FormValue("is_free") == "1" || r.FormValue("is_free") == "true"
	movie.IsLegal = r.FormValue("is_legal") == "1" || r.FormValue("is_legal") == "true"
	movie.LicenseType = r.FormValue("license_type")
	movie.IsManualEdit = true // 🔒 Kunci dari scraper otomatis
	// Simpan tipe konten (movie / anime / drama_pendek / series)
	if t := strings.TrimSpace(r.FormValue("type")); t != "" {
		movie.Type = t
	}

	_ = s.db.Save(&movie)

	// Update download links & stream links
	s.saveMovieDownloadLinks(movie.ID, r)
	s.saveMovieStreamLinks(movie.ID, r)

	http.Redirect(w, r, fmt.Sprintf("/admin/movies/edit/%d?success=Perubahan+film+berhasil+disimpan", movie.ID), http.StatusSeeOther)
}

// Helper to save/update download links from form
func (s *Server) saveMovieDownloadLinks(movieID uint, r *http.Request) {
	urls := r.Form["dl_url[]"]
	providers := r.Form["dl_provider[]"]
	qualities := r.Form["dl_quality[]"]
	resolutions := r.Form["dl_resolution[]"]
	formats := r.Form["dl_format[]"]
	sizes := r.Form["dl_size[]"]

	// Delete old links if new list submitted
	if len(urls) > 0 {
		s.db.Where("movie_id = ?", movieID).Delete(&model.DownloadLink{})
		now := time.Now()

		for i := range urls {
			cleanURL := strings.TrimSpace(urls[i])
			if cleanURL == "" {
				continue
			}

			provider := "Direct Download"
			if i < len(providers) && strings.TrimSpace(providers[i]) != "" {
				provider = providers[i]
			}
			quality := "1080p Full HD"
			if i < len(qualities) && strings.TrimSpace(qualities[i]) != "" {
				quality = qualities[i]
			}
			res := "1920x1080"
			if i < len(resolutions) && strings.TrimSpace(resolutions[i]) != "" {
				res = resolutions[i]
			}
			format := "MP4"
			if i < len(formats) && strings.TrimSpace(formats[i]) != "" {
				format = formats[i]
			}
			size := ""
			if i < len(sizes) {
				size = sizes[i]
			}

			link := model.DownloadLink{
				MovieID:        movieID,
				Provider:       provider,
				URL:            cleanURL,
				Quality:        quality,
				Resolution:     res,
				Format:         format,
				FileSize:       size,
				IsValid:        true,
				Status:         "ACTIVE",
				LastCheckedAt:  &now,
			}
			s.db.Create(&link)
		}
	}
}

// Helper to save/update stream links from form
func (s *Server) saveMovieStreamLinks(movieID uint, r *http.Request) {
	urls := r.Form["stream_url[]"]
	servers := r.Form["stream_server[]"]
	qualities := r.Form["stream_quality[]"]

	if len(urls) > 0 {
		s.db.Where("movie_id = ?", movieID).Delete(&model.StreamLink{})
		now := time.Now()

		for i := range urls {
			cleanURL := strings.TrimSpace(urls[i])
			if cleanURL == "" {
				continue
			}

			serverName := "Server Utama"
			if i < len(servers) && strings.TrimSpace(servers[i]) != "" {
				serverName = servers[i]
			}
			quality := "HD 1080p"
			if i < len(qualities) && strings.TrimSpace(qualities[i]) != "" {
				quality = qualities[i]
			}

			stream := model.StreamLink{
				MovieID:       movieID,
				ServerName:    serverName,
				EmbedURL:      cleanURL,
				Quality:       quality,
				IsValid:       true,
				Status:        "ACTIVE",
				LastCheckedAt: &now,
			}
			s.db.Create(&stream)
		}
	}
}

// HandleAdminMovieDelete handles movie deletion
func (s *Server) HandleAdminMovieDelete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err == nil && id > 0 {
		s.db.Where("movie_id = ?", id).Delete(&model.DownloadLink{})
		s.db.Where("movie_id = ?", id).Delete(&model.StreamLink{})
		s.db.Where("movie_id = ?", id).Delete(&model.Comment{})
		s.db.Delete(&model.Movie{}, id)
	}
	http.Redirect(w, r, "/admin/movies?success=Film+berhasil+dihapus", http.StatusSeeOther)
}

// HandleAdminComments renders user comments management
func (s *Server) HandleAdminComments(w http.ResponseWriter, r *http.Request) {
	user := s.GetLoggedInUser(r)

	var comments []model.Comment
	s.db.Order("id desc").Find(&comments)

	data := AdminPageData{
		Title:      "Kelola Komentar - CMS CINEMBROT",
		ActiveMenu: "comments",
		User:       user,
		Comments:   comments,
		SuccessMsg: r.URL.Query().Get("success"),
	}

	s.RenderHTML(w, "admin_comments.html", "admin_layout.html", data)
}

// HandleAdminCommentApprove toggles comment approval status
func (s *Server) HandleAdminCommentApprove(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if id, err := strconv.Atoi(idStr); err == nil {
		var comment model.Comment
		if err := s.db.First(&comment, id).Error; err == nil {
			s.db.Model(&comment).Update("is_approved", !comment.IsApproved)
		}
	}
	http.Redirect(w, r, "/admin/comments?success=Status+komentar+diperbarui", http.StatusSeeOther)
}

// HandleAdminCommentDelete deletes a user comment
func (s *Server) HandleAdminCommentDelete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if id, err := strconv.Atoi(idStr); err == nil {
		s.db.Delete(&model.Comment{}, id)
	}
	http.Redirect(w, r, "/admin/comments?success=Komentar+berhasil+dihapus", http.StatusSeeOther)
}

// HandleAdminUsers manages CMS admin & editor accounts
func (s *Server) HandleAdminUsers(w http.ResponseWriter, r *http.Request) {
	user := s.GetLoggedInUser(r)

	var users []model.User
	s.db.Order("id asc").Find(&users)

	data := AdminPageData{
		Title:      "Kelola Pengguna CMS - CINEMBROT",
		ActiveMenu: "users",
		User:       user,
		Users:      users,
		SuccessMsg: r.URL.Query().Get("success"),
		ErrorMsg:   r.URL.Query().Get("error"),
	}

	s.RenderHTML(w, "admin_users.html", "admin_layout.html", data)
}

// HandleAdminUserNew adds a new CMS user
func (s *Server) HandleAdminUserNew(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	username := strings.TrimSpace(r.FormValue("username"))
	fullName := strings.TrimSpace(r.FormValue("full_name"))
	password := strings.TrimSpace(r.FormValue("password"))
	role := strings.TrimSpace(r.FormValue("role"))

	if username == "" || password == "" {
		http.Redirect(w, r, "/admin/users?error=Username+dan+password+harus+diisi", http.StatusSeeOther)
		return
	}

	if role == "" {
		role = "editor"
	}

	var existing model.User
	if err := s.db.Where("username = ?", username).First(&existing).Error; err == nil {
		http.Redirect(w, r, "/admin/users?error=Username+sudah+digunakan", http.StatusSeeOther)
		return
	}

	newUser := model.User{
		Username:     username,
		FullName:     fullName,
		PasswordHash: auth.HashPassword(password),
		Role:         role,
		IsActive:     true,
	}

	if err := s.db.Create(&newUser).Error; err != nil {
		http.Redirect(w, r, "/admin/users?error=Gagal+menambahkan+pengguna", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/users?success=Pengguna+baru+berhasil+dibuat", http.StatusSeeOther)
}

// HandleAdminUserEdit edits a CMS user's details or resets password
func (s *Server) HandleAdminUserEdit(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	_ = r.ParseForm()
	fullName := strings.TrimSpace(r.FormValue("full_name"))
	role := strings.TrimSpace(r.FormValue("role"))
	newPassword := strings.TrimSpace(r.FormValue("password"))

	if fullName != "" {
		user.FullName = fullName
	}
	if role != "" {
		user.Role = role
	}
	if newPassword != "" {
		user.PasswordHash = auth.HashPassword(newPassword)
	}

	_ = s.db.Save(&user)
	http.Redirect(w, r, "/admin/users?success=Data+pengguna+berhasil+diperbarui", http.StatusSeeOther)
}

// HandleAdminUserDelete deletes a CMS user
func (s *Server) HandleAdminUserDelete(w http.ResponseWriter, r *http.Request) {
	currentUser := s.GetLoggedInUser(r)
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err == nil {
		// Prevent deleting oneself
		if currentUser != nil && currentUser.ID == uint(id) {
			http.Redirect(w, r, "/admin/users?error=Tidak+dapat+menghapus+akun+sendiri", http.StatusSeeOther)
			return
		}
		s.db.Delete(&model.User{}, id)
	}
	http.Redirect(w, r, "/admin/users?success=Pengguna+berhasil+dihapus", http.StatusSeeOther)
}

// HandleAdminTools renders scraper & link checker tools along with scheduler settings
func (s *Server) HandleAdminTools(w http.ResponseWriter, r *http.Request) {
	user := s.GetLoggedInUser(r)

	var sources []model.ScrapeSource
	s.db.Where("is_active = ?", true).Order("id asc").Find(&sources)

	var logs []model.ScrapeLog
	s.db.Order("id desc").Limit(50).Find(&logs)

	schedCfg := scheduler.GetSchedulerConfig(s.db, s.cfg)

	var dlSetting model.SystemSetting
	dlPath := "public/download/movie"
	if err := s.db.Where("`key` = ?", "download_movie_path").First(&dlSetting).Error; err == nil && strings.TrimSpace(dlSetting.Value) != "" {
		dlPath = strings.TrimSpace(dlSetting.Value)
	}

	data := AdminPageData{
		Title:             "Alat & Scraper - CMS CINEMBROT",
		ActiveMenu:        "tools",
		User:              user,
		Sources:           sources,
		Logs:              logs,
		SchedulerConfig:   schedCfg,
		DownloadPath:      dlPath,
		ShowTorrentPublic: database.GetShowTorrentPublic(s.db),
		SuccessMsg:        r.URL.Query().Get("success"),
	}

	s.RenderHTML(w, "admin_tools.html", "admin_layout.html", data)
}

// HandleAdminSaveSchedulerSettings saves dynamic scheduler configuration to MariaDB
func (s *Server) HandleAdminSaveSchedulerSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/tools", http.StatusSeeOther)
		return
	}

	_ = r.ParseForm()
	enabledVal := "false"
	if r.FormValue("enabled") == "on" || r.FormValue("enabled") == "true" || r.FormValue("enabled") == "1" {
		enabledVal = "true"
	}

	intervalVal := r.FormValue("interval_minutes")
	if intervalVal == "" {
		intervalVal = "30"
	}

	startYearVal := r.FormValue("start_year")
	if startYearVal == "" {
		startYearVal = strconv.Itoa(time.Now().Year())
	}

	endYearVal := r.FormValue("end_year")
	if endYearVal == "" {
		endYearVal = "2015"
	}

	pagesVal := r.FormValue("pages_per_year")
	if pagesVal == "" {
		pagesVal = "1"
	}

	delayVal := r.FormValue("delay_ms")
	if delayVal == "" {
		delayVal = "500"
	}

	dlPathVal := strings.TrimSpace(r.FormValue("download_movie_path"))
	if dlPathVal == "" {
		dlPathVal = "public/download/movie"
	}

	showTorrentVal := "false"
	if r.FormValue("show_torrent_public") == "on" || r.FormValue("show_torrent_public") == "true" || r.FormValue("show_torrent_public") == "1" {
		showTorrentVal = "true"
	}

	settingsToSave := map[string]string{
		"auto_scrape_enabled":          enabledVal,
		"auto_scrape_interval_minutes": intervalVal,
		"auto_scrape_start_year":       startYearVal,
		"auto_scrape_end_year":         endYearVal,
		"auto_scrape_pages_per_year":   pagesVal,
		"auto_scrape_delay_ms":         delayVal,
		"download_movie_path":          dlPathVal,
		"show_torrent_public":          showTorrentVal,
	}

	for k, v := range settingsToSave {
		var existing model.SystemSetting
		if err := s.db.Where("`key` = ?", k).First(&existing).Error; err != nil {
			s.db.Create(&model.SystemSetting{Key: k, Value: v, UpdatedAt: time.Now()})
		} else {
			s.db.Model(&existing).Updates(map[string]interface{}{
				"value":      v,
				"updated_at": time.Now(),
			})
		}
	}

	// Ensure target folder exists
	_ = os.MkdirAll(database.GetDownloadMoviePath(s.db), 0755)

	log.Printf("[CMS] ⚙️ Pengaturan Jadwal & Lokasi Download diperbarui: Path='%s', Enabled=%s, Interval=%s mnt\n",
		dlPathVal, enabledVal, intervalVal)

	http.Redirect(w, r, "/admin/tools?success=Pengaturan+jadwal+dan+direktori+download+berhasil+disimpan!", http.StatusSeeOther)
}

// HandleAdminTriggerScrape triggers on-demand scraping
func (s *Server) HandleAdminTriggerScrape(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	yearStr := r.FormValue("year")
	pagesStr := r.FormValue("pages")
	source := r.FormValue("source")

	year, _ := strconv.Atoi(yearStr)
	if year == 0 {
		year = time.Now().Year()
	}
	pages, _ := strconv.Atoi(pagesStr)
	if pages <= 0 {
		pages = 1
	}
	if source == "" {
		source = "all"
	}

	repo := scraper.NewRepository(s.db)
	pipe := pipeline.NewPipeline(s.cfg, repo)

	go func() {
		log.Printf("[CMS TOOL] 🚀 Menjalankan scraping manual tahun %d (%d halaman, sumber: %s)...\n", year, pages, source)
		_, _ = pipe.IngestByYear(year, pages, source)
	}()

	http.Redirect(w, r, "/admin/tools?success=Scraping+dimulai+di+latar+belakang!+Periksa+log+dalam+beberapa+saat.", http.StatusSeeOther)
}

// HandleAdminTriggerCheckLinks triggers health check of all download links in DB
func (s *Server) HandleAdminTriggerCheckLinks(w http.ResponseWriter, r *http.Request) {
	go func() {
		log.Println("[CMS TOOL] 🔍 Menjalankan validasi seluruh link download dari CMS...")
		validator.ValidateAllDatabaseLinks(s.db, s.cfg.ScraperUserAgent, 0)
	}()

	http.Redirect(w, r, "/admin/tools?success=Pengecekan+kesehatan+link+download+sedang+berjalan+di+latar+belakang!", http.StatusSeeOther)
}

// HandleAdminTriggerScrapeAnime triggers on-demand anime scraping via Jikan API
func (s *Server) HandleAdminTriggerScrapeAnime(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	category := r.FormValue("category")
	limitStr := r.FormValue("limit")

	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 15
	}
	if category == "" {
		category = "top"
	}

	repo := scraper.NewRepository(s.db)
	pipe := pipeline.NewPipeline(s.cfg, repo)

	go func() {
		log.Printf("[CMS TOOL] 🎌 Menjalankan scraping Anime MyAnimeList/Jikan (Kategori: %s, Limit: %d)...\n", category, limit)
		_, _ = pipe.IngestAnime(category, limit)
	}()

	http.Redirect(w, r, "/admin/tools?success=Scraping+Anime+resmi+(Jikan/MAL)+berhasil+dimulai+di+latar+belakang!", http.StatusSeeOther)
}

// HandleAdminTriggerScrapeDrama triggers on-demand Asian drama scraping via TMDb TV API
func (s *Server) HandleAdminTriggerScrapeDrama(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	language := r.FormValue("language")
	pageStr := r.FormValue("page")

	page, _ := strconv.Atoi(pageStr)
	if page <= 0 {
		page = 1
	}
	if language == "" {
		language = "ko"
	}

	repo := scraper.NewRepository(s.db)
	pipe := pipeline.NewPipeline(s.cfg, repo)

	go func() {
		log.Printf("[CMS TOOL] 🎭 Menjalankan scraping Drama Asia TMDb TV (Bahasa: %s, Hal: %d)...\n", language, page)
		_, _ = pipe.IngestAsianDramas(language, page)
	}()

	http.Redirect(w, r, "/admin/tools?success=Scraping+Drama+Asia+resmi+(TMDb+TV)+berhasil+dimulai+di+latar+belakang!", http.StatusSeeOther)
}

// HandleAdminTriggerScrapeHollywood triggers Hollywood / Box Office scraping via TMDb Movie API
func (s *Server) HandleAdminTriggerScrapeHollywood(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	category := strings.TrimSpace(r.FormValue("category"))
	if category == "" {
		category = "boxoffice"
	}
	pageStr := r.FormValue("page")
	page, _ := strconv.Atoi(pageStr)
	if page <= 0 {
		page = 1
	}
	yearStr := r.FormValue("year")
	year, _ := strconv.Atoi(yearStr)

	repo := scraper.NewRepository(s.db)
	pipe := pipeline.NewPipeline(s.cfg, repo)

	go func() {
		log.Printf("[CMS TOOL] 🎬 Menjalankan scraping Hollywood TMDb (Kategori: %s, Hal: %d, Tahun: %d)...\n", category, page, year)
		_, _ = pipe.IngestHollywoodMovies(category, page, year)
	}()

	http.Redirect(w, r, "/admin/tools?success=Scraping+Film+Hollywood+resmi+(TMDb)+berhasil+dimulai+di+latar+belakang!", http.StatusSeeOther)
}

// HandleAdminSources renders the website sources list from MariaDB
func (s *Server) HandleAdminSources(w http.ResponseWriter, r *http.Request) {
	user := s.GetLoggedInUser(r)

	var sources []model.ScrapeSource
	s.db.Order("id asc").Find(&sources)

	data := AdminPageData{
		Title:      "Sumber Website Scraping - CMS CINEMBROT",
		ActiveMenu: "sources",
		User:       user,
		Sources:    sources,
		SuccessMsg: r.URL.Query().Get("success"),
		ErrorMsg:   r.URL.Query().Get("error"),
	}

	s.RenderHTML(w, "admin_sources.html", "admin_layout.html", data)
}

// HandleAdminSourceNew creates a new website source in MariaDB
func (s *Server) HandleAdminSourceNew(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	code := strings.TrimSpace(strings.ToLower(r.FormValue("code")))
	baseURL := strings.TrimSpace(r.FormValue("base_url"))
	apiKey := strings.TrimSpace(r.FormValue("api_key"))
	sourceType := strings.TrimSpace(r.FormValue("type"))
	category := strings.TrimSpace(r.FormValue("category"))
	description := strings.TrimSpace(r.FormValue("description"))
	isActive := r.FormValue("is_active") == "1" || r.FormValue("is_active") == "true"
	rateLimit, _ := strconv.Atoi(r.FormValue("rate_limit_per_sec"))

	if name == "" || code == "" || baseURL == "" {
		http.Redirect(w, r, "/admin/sources?error=Nama,+Kode,+dan+Base+URL+harus+diisi", http.StatusSeeOther)
		return
	}

	if sourceType == "" {
		sourceType = "html_scrape"
	}
	if category == "" {
		category = "General"
	}
	if rateLimit <= 0 {
		rateLimit = 3
	}

	var existing model.ScrapeSource
	if err := s.db.Where("code = ?", code).First(&existing).Error; err == nil {
		http.Redirect(w, r, "/admin/sources?error=Kode+sumber+website+sudah+digunakan", http.StatusSeeOther)
		return
	}

	newSource := model.ScrapeSource{
		Name:            name,
		Code:            code,
		BaseURL:         baseURL,
		APIKey:          apiKey,
		Type:            sourceType,
		Category:        category,
		Description:     description,
		IsActive:        isActive,
		RateLimitPerSec: rateLimit,
	}

	if err := s.db.Create(&newSource).Error; err != nil {
		http.Redirect(w, r, "/admin/sources?error=Gagal+menyimpan+sumber+website", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/sources?success=Sumber+website+baru+berhasil+disimpan+di+database", http.StatusSeeOther)
}

// HandleAdminSourceEdit updates a website source in MariaDB
func (s *Server) HandleAdminSourceEdit(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var source model.ScrapeSource
	if err := s.db.First(&source, id).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	baseURL := strings.TrimSpace(r.FormValue("base_url"))
	apiKey := strings.TrimSpace(r.FormValue("api_key"))
	sourceType := strings.TrimSpace(r.FormValue("type"))
	category := strings.TrimSpace(r.FormValue("category"))
	description := strings.TrimSpace(r.FormValue("description"))
	isActive := r.FormValue("is_active") == "1" || r.FormValue("is_active") == "true"
	rateLimit, _ := strconv.Atoi(r.FormValue("rate_limit_per_sec"))

	if name != "" {
		source.Name = name
	}
	if baseURL != "" {
		source.BaseURL = baseURL
	}
	source.APIKey = apiKey
	if sourceType != "" {
		source.Type = sourceType
	}
	if category != "" {
		source.Category = category
	}
	source.Description = description
	source.IsActive = isActive
	if rateLimit > 0 {
		source.RateLimitPerSec = rateLimit
	}

	_ = s.db.Save(&source)
	http.Redirect(w, r, "/admin/sources?success=Data+sumber+website+berhasil+diperbarui", http.StatusSeeOther)
}

// HandleAdminSourceToggle toggles is_active status of a website source
func (s *Server) HandleAdminSourceToggle(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err == nil {
		var source model.ScrapeSource
		if err := s.db.First(&source, id).Error; err == nil {
			source.IsActive = !source.IsActive
			s.db.Save(&source)
		}
	}
	http.Redirect(w, r, "/admin/sources?success=Status+sumber+website+berhasil+diubah", http.StatusSeeOther)
}

// HandleAdminSourceDelete deletes a website source from MariaDB
func (s *Server) HandleAdminSourceDelete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err == nil {
		s.db.Delete(&model.ScrapeSource{}, id)
	}
	http.Redirect(w, r, "/admin/sources?success=Sumber+website+berhasil+dihapus+dari+database", http.StatusSeeOther)
}
