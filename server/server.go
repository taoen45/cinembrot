package server

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cinembrot/config"
	"cinembrot/database"
	"cinembrot/i18n"
	"cinembrot/provider/archive"
	"cinembrot/provider/tmdb"
	"cinembrot/provider/yts"
	"cinembrot/scraper"
	"cinembrot/torrentmgr"
	"gorm.io/gorm"
)

type PageItem struct {
	Number   int
	IsActive bool
	IsDots   bool
}

type Server struct {
	cfg        *config.Config
	db         *gorm.DB
	templates  map[string]*template.Template
	torrentMgr *torrentmgr.Manager
	tmdbCli    *tmdb.Client
	archiveCli *archive.Client
	ytsCli     *yts.Client
}

func NewServer(cfg *config.Config, db *gorm.DB) *Server {
	dlDir := database.GetDownloadMoviePath(db)
	tm, err := torrentmgr.NewManager(db, dlDir)
	if err != nil {
		log.Printf("[WARN] Failed to initialize Torrent Manager: %v\n", err)
	}

	s := &Server{
		cfg:        cfg,
		db:         db,
		templates:  make(map[string]*template.Template),
		torrentMgr: tm,
		tmdbCli:    tmdb.NewClient(cfg),
		archiveCli: archive.NewClient(cfg),
		ytsCli:     yts.NewClient(cfg),
	}
	s.loadTemplates()
	return s
}

func (s *Server) loadTemplates() {
	funcMap := template.FuncMap{
		"t": func(lang string, key string) string {
			return i18n.T(lang, key)
		},
		"safeHTML": func(str string) template.HTML {
			return template.HTML(str)
		},
		"safeJSON": func(v interface{}) template.JS {
			switch val := v.(type) {
			case template.HTML:
				return template.JS(val)
			case string:
				return template.JS(val)
			default:
				return template.JS(fmt.Sprint(val))
			}
		},
		"cleanText": func(str string) string {
			return scraper.CleanHTMLToPlainText(str)
		},
		"cleanQuality": func(q string) string {
			for _, pat := range []string{"(Hardsub Indo)", "(Hardsub Indonesia)", "(hardsub indo)", "(hardsub indonesia)", "Hardsub Indo", "Hardsub"} {
				q = strings.ReplaceAll(q, pat, "")
			}
			q = strings.TrimSpace(q)
			if q == "" {
				return "720p HD"
			}
			return q
		},
		"cleanProvider": func(p string) string {
			for _, pat := range []string{"(Hardsub Indonesia)", "(Hardsub Indo)", "(hardsub indonesia)", "(hardsub indo)"} {
				p = strings.ReplaceAll(p, pat, "")
			}
			p = strings.TrimSpace(p)
			if p == "" {
				return "Server Lokal"
			}
			return p
		},
		"paragraphs": func(str string) []string {
			clean := scraper.CleanHTMLToPlainText(str)
			if clean == "" {
				return nil
			}
			parts := strings.Split(clean, "\n\n")
			var res []string
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					res = append(res, p)
				}
			}
			return res
		},
		"safeURL": func(u string) template.URL {
			return template.URL(u)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"div": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"float": func(v interface{}) float64 {
			switch val := v.(type) {
			case int:
				return float64(val)
			case int64:
				return float64(val)
			case uint:
				return float64(val)
			case float64:
				return val
			default:
				return 0
			}
		},
		"youtubeEmbed": func(rawURL string) string {
			if strings.Contains(rawURL, "watch?v=") {
				return strings.Replace(rawURL, "watch?v=", "embed/", 1)
			} else if strings.Contains(rawURL, "youtu.be/") {
				return strings.Replace(rawURL, "youtu.be/", "www.youtube.com/embed/", 1)
			}
			return rawURL
		},
		"pageItems": func(current, total int) []PageItem {
			if total <= 1 {
				return nil
			}
			var items []PageItem
			if total <= 7 {
				for i := 1; i <= total; i++ {
					items = append(items, PageItem{Number: i, IsActive: i == current})
				}
				return items
			}

			if current <= 4 {
				for i := 1; i <= 5; i++ {
					items = append(items, PageItem{Number: i, IsActive: i == current})
				}
				items = append(items, PageItem{IsDots: true})
				items = append(items, PageItem{Number: total, IsActive: current == total})
				return items
			}

			if current >= total-3 {
				items = append(items, PageItem{Number: 1, IsActive: current == 1})
				items = append(items, PageItem{IsDots: true})
				for i := total - 4; i <= total; i++ {
					items = append(items, PageItem{Number: i, IsActive: i == current})
				}
				return items
			}

			items = append(items, PageItem{Number: 1, IsActive: false})
			items = append(items, PageItem{IsDots: true})
			for i := current - 1; i <= current+1; i++ {
				items = append(items, PageItem{Number: i, IsActive: i == current})
			}
			items = append(items, PageItem{IsDots: true})
			items = append(items, PageItem{Number: total, IsActive: false})
			return items
		},
		"timeAgo": func(t time.Time) string {
			if t.IsZero() {
				return "Baru saja"
			}
			dur := time.Since(t)
			if dur < time.Minute {
				return "Baru saja"
			} else if dur < time.Hour {
				return fmt.Sprintf("%d menit yang lalu", int(dur.Minutes()))
			} else if dur < 24*time.Hour {
				return fmt.Sprintf("%d jam yang lalu", int(dur.Hours()))
			} else if dur < 30*24*time.Hour {
				return fmt.Sprintf("%d hari yang lalu", int(dur.Hours()/24))
			}
			return t.Format("02 Jan 2006")
		},
	}

	viewsDir := filepath.Join("server", "views")
	layoutPath := filepath.Join(viewsDir, "layout.html")

	// Public Pages
	pages := []string{"home.html", "detail.html", "list.html", "anime.html", "drama_pendek.html", "hollywood.html"}
	for _, page := range pages {
		pagePath := filepath.Join(viewsDir, page)
		tmpl := template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles(layoutPath, pagePath))
		s.templates[page] = tmpl
	}

	// CMS Admin Pages
	adminLayoutPath := filepath.Join(viewsDir, "admin_layout.html")
	adminPages := []string{
		"admin_dashboard.html",
		"admin_movies.html",
		"admin_movie_form.html",
		"admin_comments.html",
		"admin_users.html",
		"admin_sources.html",
		"admin_tools.html",
		"admin_downloads.html",
		"admin_download_review.html",
		"admin_settings.html",
	}
	for _, page := range adminPages {
		pagePath := filepath.Join(viewsDir, page)
		tmpl := template.Must(template.New("admin_layout.html").Funcs(funcMap).ParseFiles(adminLayoutPath, pagePath))
		s.templates[page] = tmpl
	}

	// Standalone Admin Login Page
	loginPath := filepath.Join(viewsDir, "admin_login.html")
	s.templates["admin_login.html"] = template.Must(template.New("admin_login.html").Funcs(funcMap).ParseFiles(loginPath))
}

// RenderHTML sets the Content-Type header to text/html and renders the template.
// Templates are reloaded on every request so HTML edits are visible with just F5 (no restart needed).
func (s *Server) RenderHTML(w http.ResponseWriter, tmplName, layout string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Automatically populate system settings if rendering public pages with PageData
	if pd, ok := data.(*PageData); ok {
		s.PopulatePageData(nil, pd)
	} else if pd, ok := data.(PageData); ok {
		s.PopulatePageData(nil, &pd)
		data = pd
	}

	s.loadTemplates() // Auto-reload: baca ulang file HTML dari disk setiap request
	if tmpl, ok := s.templates[tmplName]; ok {
		var err error
		if layout != "" {
			err = tmpl.ExecuteTemplate(w, layout, data)
		} else {
			err = tmpl.Execute(w, data)
		}
		if err != nil {
			log.Printf("[ERROR] RenderHTML error for '%s' (%s): %v\n", tmplName, layout, err)
		}
		return
	}
	log.Printf("[ERROR] Template not found: %s\n", tmplName)
	http.Error(w, "Template Not Found", http.StatusInternalServerError)
}

// Start runs the HTTP web server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Route Public Web Endpoints
	mux.HandleFunc("GET /", s.HandleHome)
	mux.HandleFunc("GET /hollywood", s.HandleHollywood)
	mux.HandleFunc("GET /box-office", s.HandleHollywood)
	mux.HandleFunc("GET /anime", s.HandleAnime)
	mux.HandleFunc("GET /drama-pendek", s.HandleDramaPendek)
	mux.HandleFunc("GET /filter", s.HandleFilter)
	mux.HandleFunc("GET /movie/{slug}", s.HandleMovieDetail)
	mux.HandleFunc("POST /movie/{slug}/comment", s.HandleSubmitComment)
	mux.HandleFunc("GET /year/{year}", s.HandleYearFilter)
	mux.HandleFunc("GET /genre/{slug}", s.HandleGenreFilter)
	mux.HandleFunc("GET /free", s.HandleFreeFilter)
	mux.HandleFunc("GET /search", s.HandleSearch)
	mux.HandleFunc("GET /set-lang", s.HandleSetLang)
	mux.HandleFunc("GET /api/movies", s.HandleAPIMovies)
	mux.HandleFunc("POST /api/movie/{id}/refresh-download", s.HandleRefreshDownloadLink)
	mux.HandleFunc("POST /api/movie/{id}/refresh-stream", s.HandleRefreshStreamLink)

	// SEO & Search Engine Discovery Endpoints
	mux.HandleFunc("GET /sitemap.xml", s.HandleSitemapXML)
	mux.HandleFunc("GET /robots.txt", s.HandleRobotsTXT)
	mux.HandleFunc("GET /ads.txt", s.HandleAdsTXT)

	// Route CMS Admin Endpoints
	mux.HandleFunc("GET /admin/login", s.HandleAdminLogin)
	mux.HandleFunc("POST /admin/login", s.HandleAdminLogin)
	mux.HandleFunc("GET /admin/logout", s.HandleAdminLogout)

	// Protected CMS Routes (Guarded by RequireAdmin)
	mux.HandleFunc("GET /admin", s.RequireAdmin(s.HandleAdminDashboard))
	mux.HandleFunc("GET /admin/movies", s.RequireAdmin(s.HandleAdminMovies))
	mux.HandleFunc("GET /admin/hollywood", s.RequireAdmin(s.HandleAdminHollywood))
	mux.HandleFunc("GET /admin/anime", s.RequireAdmin(s.HandleAdminAnime))
	mux.HandleFunc("GET /admin/drama-pendek", s.RequireAdmin(s.HandleAdminDramaPendek))
	mux.HandleFunc("GET /admin/movies/new", s.RequireAdmin(s.HandleAdminMovieNew))
	mux.HandleFunc("POST /admin/movies/new", s.RequireAdmin(s.HandleAdminMovieNew))
	mux.HandleFunc("GET /admin/movies/edit/{id}", s.RequireAdmin(s.HandleAdminMovieEdit))
	mux.HandleFunc("POST /admin/movies/edit/{id}", s.RequireAdmin(s.HandleAdminMovieEdit))
	mux.HandleFunc("POST /admin/movies/delete/{id}", s.RequireAdmin(s.HandleAdminMovieDelete))

	mux.HandleFunc("GET /admin/comments", s.RequireAdmin(s.HandleAdminComments))
	mux.HandleFunc("POST /admin/comments/approve/{id}", s.RequireAdmin(s.HandleAdminCommentApprove))
	mux.HandleFunc("POST /admin/comments/delete/{id}", s.RequireAdmin(s.HandleAdminCommentDelete))

	mux.HandleFunc("GET /admin/users", s.RequireAdmin(s.HandleAdminUsers))
	mux.HandleFunc("POST /admin/users/new", s.RequireAdmin(s.HandleAdminUserNew))
	mux.HandleFunc("POST /admin/users/edit/{id}", s.RequireAdmin(s.HandleAdminUserEdit))
	mux.HandleFunc("POST /admin/users/delete/{id}", s.RequireAdmin(s.HandleAdminUserDelete))

	mux.HandleFunc("GET /admin/sources", s.RequireAdmin(s.HandleAdminSources))
	mux.HandleFunc("POST /admin/sources/new", s.RequireAdmin(s.HandleAdminSourceNew))
	mux.HandleFunc("POST /admin/sources/edit/{id}", s.RequireAdmin(s.HandleAdminSourceEdit))
	mux.HandleFunc("POST /admin/sources/toggle/{id}", s.RequireAdmin(s.HandleAdminSourceToggle))
	mux.HandleFunc("POST /admin/sources/delete/{id}", s.RequireAdmin(s.HandleAdminSourceDelete))

	mux.HandleFunc("GET /admin/tools", s.RequireAdmin(s.HandleAdminTools))
	mux.HandleFunc("POST /admin/tools/scheduler-settings", s.RequireAdmin(s.HandleAdminSaveSchedulerSettings))
	mux.HandleFunc("POST /admin/tools/scrape", s.RequireAdmin(s.HandleAdminTriggerScrape))
	mux.HandleFunc("POST /admin/tools/scrape-hollywood", s.RequireAdmin(s.HandleAdminTriggerScrapeHollywood))
	mux.HandleFunc("POST /admin/tools/scrape-anime", s.RequireAdmin(s.HandleAdminTriggerScrapeAnime))
	mux.HandleFunc("POST /admin/tools/scrape-drama", s.RequireAdmin(s.HandleAdminTriggerScrapeDrama))
	mux.HandleFunc("POST /admin/tools/run-daily-scrape", s.RequireAdmin(s.HandleAdminTriggerDailyScrape))
	mux.HandleFunc("POST /admin/tools/check-links", s.RequireAdmin(s.HandleAdminTriggerCheckLinks))

	// Torrent & Subtitle Downloader & Hardsub Routes
	mux.HandleFunc("GET /admin/downloads", s.RequireAdmin(s.HandleAdminDownloads))
	mux.HandleFunc("GET /admin/downloads/review/{id}", s.RequireAdmin(s.HandleAdminDownloadsReview))
	mux.HandleFunc("GET /api/admin/downloads/status", s.RequireAdmin(s.HandleAdminDownloadsStatusAPI))
	mux.HandleFunc("POST /admin/downloads/cancel/{id}", s.RequireAdmin(s.HandleAdminDownloadsCancelAPI))
	mux.HandleFunc("POST /admin/downloads/delete/{id}", s.RequireAdmin(s.HandleAdminDownloadsCancelAPI))
	mux.HandleFunc("POST /api/admin/downloads/cancel/{id}", s.RequireAdmin(s.HandleAdminDownloadsCancelAPI))
	mux.HandleFunc("POST /api/admin/downloads/delete/{id}", s.RequireAdmin(s.HandleAdminDownloadsCancelAPI))
	mux.HandleFunc("DELETE /api/admin/downloads/{id}", s.RequireAdmin(s.HandleAdminDownloadsCancelAPI))
	mux.HandleFunc("POST /api/admin/downloads/select-subtitle/{id}", s.RequireAdmin(s.HandleAdminDownloadsSelectSubtitleAPI))
	mux.HandleFunc("POST /api/admin/downloads/upload-subtitle/{id}", s.RequireAdmin(s.HandleAdminDownloadsUploadSubtitleAPI))
	mux.HandleFunc("POST /api/admin/downloads/hardsub/{id}", s.RequireAdmin(s.HandleAdminDownloadsHardsubAPI))

	// System Settings & Adsterra Ad Unit Controls
	mux.HandleFunc("GET /admin/settings", s.RequireAdmin(s.HandleAdminSettings))
	mux.HandleFunc("POST /admin/settings", s.RequireAdmin(s.HandleAdminSaveSettings))

	// Serve Static Files (Local WebP Originals & Thumbnails with Smart Placeholder Fallback)
	uploadsDir := filepath.Join("public", "uploads")
	_ = os.MkdirAll(uploadsDir, 0755)
	fileServer := http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadsDir)))
	placeholderPath := filepath.Join("public", "img", "poster-placeholder.svg")
	mux.HandleFunc("GET /uploads/", func(w http.ResponseWriter, r *http.Request) {
		relPath := strings.TrimPrefix(r.URL.Path, "/uploads/")
		fullPath := filepath.Join(uploadsDir, filepath.Clean(relPath))
		if fi, err := os.Stat(fullPath); os.IsNotExist(err) || fi.IsDir() {
			w.Header().Set("Content-Type", "image/svg+xml")
			w.Header().Set("Cache-Control", "no-cache")
			http.ServeFile(w, r, placeholderPath)
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	// Serve Downloaded Videos & Subtitle Files (Configurable via system_settings, default: public/download/movie)
	downloadsDir := database.GetDownloadMoviePath(s.db)
	_ = os.MkdirAll(downloadsDir, 0755)
	mux.Handle("GET /downloads/", http.StripPrefix("/downloads/", http.FileServer(http.Dir(downloadsDir))))
	mux.Handle("GET /public/download/movie/", http.StripPrefix("/public/download/movie/", http.FileServer(http.Dir(downloadsDir))))

	// Static Brand Assets (Favicon & Mascot Logos)
	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join("public", "favicon.png"))
	})
	mux.HandleFunc("GET /favicon.png", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join("public", "favicon.png"))
	})
	mux.HandleFunc("GET /logo.png", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join("public", "img", "logo.png"))
	})
	mux.Handle("GET /img/", http.StripPrefix("/img/", http.FileServer(http.Dir(filepath.Join("public", "img")))))

	port := s.cfg.ServerPort
	if port == "" {
		port = "8080"
	}

	log.Printf("\n==============================================================")
	log.Printf(" 🚀 CINEMBROT WEB SERVER AKTIF DI: http://localhost:%s", port)
	log.Printf("==============================================================\n")

	return http.ListenAndServe(":"+port, mux)
}

// HandleSetLang handles switching language and redirects back to the previous page
func (s *Server) HandleSetLang(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")
	i18n.SetLangCookie(w, lang)

	if lang == "en" {
		http.SetCookie(w, &http.Cookie{
			Name:     "googtrans",
			Value:    "/id/en",
			Path:     "/",
			Expires:  time.Now().Add(365 * 24 * time.Hour),
			MaxAge:   365 * 24 * 3600,
			SameSite: http.SameSiteLaxMode,
		})
	} else {
		http.SetCookie(w, &http.Cookie{
			Name:     "googtrans",
			Value:    "/id/id",
			Path:     "/",
			Expires:  time.Now().Add(-24 * time.Hour),
			MaxAge:   -1,
			SameSite: http.SameSiteLaxMode,
		})
	}

	redirect := r.URL.Query().Get("redirect")
	if redirect == "" || strings.HasPrefix(redirect, "//") || strings.Contains(redirect, "://") {
		redirect = r.Header.Get("Referer")
	}
	if redirect == "" || strings.HasPrefix(redirect, "//") || strings.Contains(redirect, "://") {
		redirect = "/"
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}
