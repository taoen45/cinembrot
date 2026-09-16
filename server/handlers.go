package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"cinembrot/database"
	"cinembrot/i18n"
	"cinembrot/model"
	"cinembrot/provider/embed"
	"cinembrot/scraper"
	"gorm.io/gorm"
)

type PageData struct {
	Lang                   string
	Title                  string
	SiteName               string
	SiteTagline            string
	ActiveMenu             string
	Movies                 []model.Movie
	Slides                 []model.Movie
	BoxOffice              []model.Movie
	TopRated               []model.Movie
	FreeMovies             []model.Movie
	Featured               *model.Movie
	Movie                  *model.Movie
	Related                []model.Movie
	Genres                 []model.Genre
	Years                  []int
	Countries              []string
	Categories             []string
	CurrentYear            int
	CurrentGenre           string
	CurrentCountry         string
	CurrentCategory        string
	CurrentSort            string
	SearchQuery            string
	TotalCount             int64
	EnableAds              bool
	EnableComments         bool
	ShowTorrentPublic      bool
	TranslateEnabled       bool
	GoogleTranslateEnabled bool
	AdsterraPopunder       template.HTML
	AdsterraSocialBar      template.HTML
	AdsterraBanner728      template.HTML
	AdsterraBanner468      template.HTML
	AdsterraBanner300      template.HTML
	AdsterraBanner160x600  template.HTML
	AdsterraBanner160x300  template.HTML
	AdsterraBanner320      template.HTML
	AdsterraNative         template.HTML
	AdsterraSmartlink      string
	CaptchaQuestion        string
	CaptchaToken           string
	EpisodesList           []int
	CurrentEpisode         int
	TMDbID                 int

	// SEO & Search Engine Optimization Fields
	CanonicalURL       string
	MetaDescription    string
	MetaKeywords       string
	MetaImage          string
	SiteURL            string
	OGType             string
	GoogleVerification string
	BingVerification   string
	YandexVerification string
	JSONLD             template.HTML
}

// PopulatePageData automatically sets dynamic settings from database into PageData
func (s *Server) PopulatePageData(r *http.Request, data *PageData) {
	settings := database.GetAllSettings(s.db)

	if data.Lang == "" && r != nil {
		data.Lang = i18n.GetLang(r)
	}
	if data.SiteName == "" {
		if val, ok := settings["site_name"]; ok && strings.TrimSpace(val) != "" {
			data.SiteName = val
		} else {
			data.SiteName = "CINEMBROT"
		}
	}
	if data.SiteTagline == "" {
		data.SiteTagline = settings["site_tagline"]
	}

	adsVal := settings["ads_enabled"]
	adsEnabled := adsVal != "false" && adsVal != "0"
	data.EnableAds = adsEnabled

	if adsEnabled {
		data.AdsterraPopunder = template.HTML(settings["adsterra_popunder_code"])
		data.AdsterraSocialBar = template.HTML(settings["adsterra_socialbar_code"])
		data.AdsterraBanner728 = template.HTML(settings["adsterra_banner_728_code"])
		data.AdsterraBanner468 = template.HTML(settings["adsterra_banner_468_code"])
		data.AdsterraBanner300 = template.HTML(settings["adsterra_banner_300_code"])
		data.AdsterraBanner160x600 = template.HTML(settings["adsterra_banner_160x600_code"])
		data.AdsterraBanner160x300 = template.HTML(settings["adsterra_banner_160x300_code"])
		data.AdsterraBanner320 = template.HTML(settings["adsterra_banner_320_code"])
		data.AdsterraNative = template.HTML(settings["adsterra_native_banner_code"])
		data.AdsterraSmartlink = settings["adsterra_smartlink_url"]
	}

	transVal := settings["translate_enabled"]
	data.TranslateEnabled = transVal != "false" && transVal != "0"

	gTransVal := settings["google_translate_enabled"]
	data.GoogleTranslateEnabled = gTransVal != "false" && gTransVal != "0"

	comVal := settings["comments_enabled"]
	data.EnableComments = comVal != "false" && comVal != "0"

	torVal := settings["show_torrent_public"]
	data.ShowTorrentPublic = torVal == "true" || torVal == "1"
}

// HandleHome displays home page with top 10 movies slider & multi-filter dropdown bar
func (s *Server) HandleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// 1. Top 10 latest/popular movies for Big Hero Carousel
	var slides []model.Movie
	s.db.Preload("Genres").Preload("Directors").
		Where("backdrop_url <> '' OR poster_url <> ''").
		Order("year desc, rating desc, id desc").
		Limit(10).
		Find(&slides)

	// Ambil bahasa pengguna
	lang := i18n.GetLang(r)
	if lang == "en" {
		for i := range slides {
			if enSyn := s.tmdbCli.GetEnglishSynopsis(slides[i].SourceURL, slides[i].Title, slides[i].Type); enSyn != "" {
				slides[i].Synopsis = enSyn
			}
		}
	}

	var featured *model.Movie
	if len(slides) > 0 {
		featured = &slides[0]
	}

	// 2. Box Office & Populer movies for horizontal slider 1
	var boxOffice []model.Movie
	s.db.Preload("Genres").
		Where("poster_url <> ''").
		Order("year desc, views desc, id desc").
		Limit(12).
		Find(&boxOffice)

	// 3. Top Rated movies for horizontal slider 2
	var topRated []model.Movie
	s.db.Preload("Genres").
		Where("poster_url <> '' AND rating > 0").
		Order("rating desc, id desc").
		Limit(12).
		Find(&topRated)

	// 4. Free & Legal movies for horizontal slider 3
	var freeMovies []model.Movie
	s.db.Preload("Genres").
		Where("is_free = ? AND poster_url <> ''", true).
		Order("views desc, id desc").
		Limit(12).
		Find(&freeMovies)

	// 5. Latest movie catalog grid
	var movies []model.Movie
	s.db.Preload("Genres").Order("id desc").Limit(24).Find(&movies)

	// 6. Dropdown filter datasets
	var genres []model.Genre
	s.db.Order("name asc").Find(&genres)

	var years []int
	s.db.Model(&model.Movie{}).Distinct().Where("year > 0").Order("year desc").Pluck("year", &years)
	if len(years) == 0 {
		years = []int{2026, 2025, 2024, 2023, 2022, 2021, 2020, 2019, 2018, 2015, 2010, 2000, 1994, 1968}
	}

	var countries []string
	s.db.Model(&model.Movie{}).Distinct().Where("country <> ''").Order("country asc").Pluck("country", &countries)

	data := PageData{
		Lang:       i18n.GetLang(r),
		Title:      "Nonton Film Streaming & Download Gratis Legal",
		SiteName:   "CINEMBROT",
		ActiveMenu: "home",
		Slides:     slides,
		BoxOffice:  boxOffice,
		TopRated:   topRated,
		FreeMovies: freeMovies,
		Featured:   featured,
		Movies:     movies,
		Genres:     genres,
		Years:      years,
		Countries:  countries,
		EnableAds:  s.cfg.EnableAds,
	}

	s.PopulateSEO(r, &data)
	s.RenderHTML(w, "home.html", "layout.html", data)
}

// HandleFilter handles dynamic multi-parameter filtering (Year, Genre, Country, Category, Sort)
func (s *Server) HandleFilter(w http.ResponseWriter, r *http.Request) {
	yearStr := strings.TrimSpace(r.URL.Query().Get("year"))
	genreStr := strings.TrimSpace(r.URL.Query().Get("genre"))
	countryStr := strings.TrimSpace(r.URL.Query().Get("country"))
	catStr := strings.TrimSpace(r.URL.Query().Get("category"))
	sortStr := strings.TrimSpace(r.URL.Query().Get("sort"))

	query := s.db.Model(&model.Movie{}).Preload("Genres")

	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil && y > 0 {
			query = query.Where("year = ?", y)
		}
	}

	if genreStr != "" {
		var genre model.Genre
		if err := s.db.Where("slug = ? OR name = ?", genreStr, genreStr).First(&genre).Error; err == nil {
			query = query.Joins("JOIN movie_genres ON movie_genres.movie_id = movies.id").
				Where("movie_genres.genre_id = ?", genre.ID)
		}
	}

	if countryStr != "" {
		query = query.Where("country LIKE ?", "%"+countryStr+"%")
	}

	if catStr != "" {
		switch catStr {
		case "free", "100% Gratis & Legal":
			query = query.Where("is_free = ?", true)
		case "public_domain", "Public Domain":
			query = query.Where("license_type = ?", "Public Domain")
		case "creative_commons", "Creative Commons":
			query = query.Where("license_type = ?", "Creative Commons")
		case "commercial", "Berlisensi Komersil":
			query = query.Where("is_free = ?", false)
		}
	}

	// Sorting
	switch sortStr {
	case "rating_desc":
		query = query.Order("rating desc, id desc")
	case "rating_asc":
		query = query.Order("rating asc, id desc")
	case "year_asc":
		query = query.Order("year asc, id desc")
	case "title_asc":
		query = query.Order("title asc")
	case "views_desc":
		query = query.Order("views desc, id desc")
	default:
		query = query.Order("year desc, id desc") // Default: Terbaru
	}

	var movies []model.Movie
	query.Limit(60).Find(&movies)

	var genres []model.Genre
	s.db.Order("name asc").Find(&genres)

	var years []int
	s.db.Model(&model.Movie{}).Distinct().Where("year > 0").Order("year desc").Pluck("year", &years)

	var countries []string
	s.db.Model(&model.Movie{}).Distinct().Where("country <> ''").Order("country asc").Pluck("country", &countries)

	categories := []string{"100% Gratis & Legal", "Public Domain", "Creative Commons", "Berlisensi Komersil"}

	title := "Katalog & Filter Film"
	if genreStr != "" {
		title += " - Genre: " + genreStr
	}
	if yearStr != "" {
		title += " (" + yearStr + ")"
	}

	data := PageData{
		Lang:            i18n.GetLang(r),
		Title:           title,
		SiteName:        "CINEMBROT",
		ActiveMenu:      "filter",
		Movies:          movies,
		Genres:          genres,
		Years:           years,
		Countries:       countries,
		Categories:      categories,
		CurrentYear:     0,
		CurrentGenre:    genreStr,
		CurrentCountry:  countryStr,
		CurrentCategory: catStr,
		CurrentSort:     sortStr,
		TotalCount:      int64(len(movies)),
		EnableAds:       s.cfg.EnableAds,
	}
	if y, err := strconv.Atoi(yearStr); err == nil {
		data.CurrentYear = y
	}

	s.PopulateSEO(r, &data)
	s.RenderHTML(w, "list.html", "layout.html", data)
}

// HandleHollywood displays the dedicated Hollywood & Box Office movies catalog with multi-parameter filtering
func (s *Server) HandleHollywood(w http.ResponseWriter, r *http.Request) {
	yearStr := strings.TrimSpace(r.URL.Query().Get("year"))
	genreStr := strings.TrimSpace(r.URL.Query().Get("genre"))
	countryStr := strings.TrimSpace(r.URL.Query().Get("country"))
	sortStr := strings.TrimSpace(r.URL.Query().Get("sort"))

	query := s.db.Model(&model.Movie{}).
		Where("type = ? OR (type = 'movie' AND (country LIKE '%United States%' OR country LIKE '%USA%' OR country LIKE '%UK%' OR country LIKE '%Amerika%' OR language LIKE '%English%' OR language = 'en'))", "hollywood").
		Preload("Genres")

	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil && y > 0 {
			query = query.Where("year = ?", y)
		}
	}

	if genreStr != "" {
		var genre model.Genre
		if err := s.db.Where("slug = ? OR name = ?", genreStr, genreStr).First(&genre).Error; err == nil {
			query = query.Joins("JOIN movie_genres ON movie_genres.movie_id = movies.id").
				Where("movie_genres.genre_id = ?", genre.ID)
		}
	}

	if countryStr != "" {
		query = query.Where("country LIKE ?", "%"+countryStr+"%")
	}

	// Sorting
	switch sortStr {
	case "rating_desc":
		query = query.Order("rating desc, id desc")
	case "rating_asc":
		query = query.Order("rating asc, id desc")
	case "year_asc":
		query = query.Order("year asc, id desc")
	case "title_asc":
		query = query.Order("title asc")
	case "views_desc":
		query = query.Order("views desc, id desc")
	default:
		query = query.Order("id desc") // Default: Terbaru ditambahkan
	}

	var totalCount int64
	query.Count(&totalCount)

	var movies []model.Movie
	query.Limit(60).Find(&movies)

	var genres []model.Genre
	s.db.Order("name asc").Find(&genres)

	var years []int
	s.db.Model(&model.Movie{}).Distinct().
		Where("(type = ? OR (type = 'movie' AND (country LIKE '%United States%' OR country LIKE '%USA%' OR country LIKE '%UK%' OR country LIKE '%Amerika%' OR language LIKE '%English%' OR language = 'en'))) AND year > 0", "hollywood").
		Order("year desc").Pluck("year", &years)
	if len(years) == 0 {
		years = []int{2026, 2025, 2024, 2023, 2022, 2021, 2020, 2019, 2018, 2015, 2010}
	}

	var countries []string
	s.db.Model(&model.Movie{}).Distinct().
		Where("(type = ? OR (type = 'movie' AND (country LIKE '%United States%' OR country LIKE '%USA%' OR country LIKE '%UK%' OR country LIKE '%Amerika%' OR language LIKE '%English%' OR language = 'en'))) AND country <> ''", "hollywood").
		Order("country asc").Pluck("country", &countries)
	if len(countries) == 0 {
		countries = []string{"United States", "United Kingdom", "Canada", "Australia"}
	}

	data := PageData{
		Lang:            i18n.GetLang(r),
		Title:           "Nonton Film Hollywood & Box Office Subtitle Indonesia HD",
		SiteName:        "CINEMBROT",
		ActiveMenu:      "hollywood",
		Movies:          movies,
		Genres:          genres,
		Years:           years,
		Countries:       countries,
		CurrentGenre:    genreStr,
		CurrentCountry:  countryStr,
		CurrentSort:     sortStr,
		TotalCount:      totalCount,
		EnableAds:       s.cfg.EnableAds,
		MetaDescription: "Koleksi film bioskop Hollywood terlaris, box office blockbuster, dan film barat terbaru subtitle Indonesia gratis kualitas HD di CINEMBROT.",
		MetaKeywords:    "nonton film hollywood, box office sub indo, film barat terbaru, streaming film bioskop barat, download film hollywood, cinembrot box office",
	}
	if y, err := strconv.Atoi(yearStr); err == nil {
		data.CurrentYear = y
	}

	s.PopulateSEO(r, &data)
	s.RenderHTML(w, "hollywood.html", "layout.html", data)
}

// HandleAnime displays the dedicated Anime catalog with multi-parameter filtering
func (s *Server) HandleAnime(w http.ResponseWriter, r *http.Request) {
	yearStr := strings.TrimSpace(r.URL.Query().Get("year"))
	genreStr := strings.TrimSpace(r.URL.Query().Get("genre"))
	countryStr := strings.TrimSpace(r.URL.Query().Get("country"))
	sortStr := strings.TrimSpace(r.URL.Query().Get("sort"))

	query := s.db.Model(&model.Movie{}).Where("type = ?", "anime").Preload("Genres")

	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil && y > 0 {
			query = query.Where("year = ?", y)
		}
	}

	if genreStr != "" {
		var genre model.Genre
		if err := s.db.Where("slug = ? OR name = ?", genreStr, genreStr).First(&genre).Error; err == nil {
			query = query.Joins("JOIN movie_genres ON movie_genres.movie_id = movies.id").
				Where("movie_genres.genre_id = ?", genre.ID)
		}
	}

	if countryStr != "" {
		query = query.Where("country LIKE ?", "%"+countryStr+"%")
	}

	// Sorting
	switch sortStr {
	case "rating_desc":
		query = query.Order("rating desc, id desc")
	case "rating_asc":
		query = query.Order("rating asc, id desc")
	case "year_asc":
		query = query.Order("year asc, id desc")
	case "title_asc":
		query = query.Order("title asc")
	case "views_desc":
		query = query.Order("views desc, id desc")
	default:
		query = query.Order("id desc") // Default: Terbaru ditambahkan
	}

	var totalCount int64
	query.Count(&totalCount)

	var movies []model.Movie
	query.Limit(60).Find(&movies)

	var genres []model.Genre
	s.db.Order("name asc").Find(&genres)

	var years []int
	s.db.Model(&model.Movie{}).Distinct().Where("type = ? AND year > 0", "anime").Order("year desc").Pluck("year", &years)
	if len(years) == 0 {
		years = []int{2026, 2025, 2024, 2023, 2022, 2021, 2020}
	}

	var countries []string
	s.db.Model(&model.Movie{}).Distinct().Where("type = ? AND country <> ''", "anime").Order("country asc").Pluck("country", &countries)
	if len(countries) == 0 {
		countries = []string{"Japan", "Jepang", "China", "Korea"}
	}

	data := PageData{
		Lang:            i18n.GetLang(r),
		Title:           "Nonton Anime Subtitle Indonesia Terbaru & Populer HD",
		SiteName:        "CINEMBROT",
		ActiveMenu:      "anime",
		Movies:          movies,
		Genres:          genres,
		Years:           years,
		Countries:       countries,
		CurrentGenre:    genreStr,
		CurrentCountry:  countryStr,
		CurrentSort:     sortStr,
		TotalCount:      totalCount,
		EnableAds:       s.cfg.EnableAds,
		MetaDescription: "Nonton anime subtitle Indonesia online terlengkap dan terupdate kualitas HD. Streaming dan download anime movie & series gratis tanpa ribet di CINEMBROT.",
		MetaKeywords:    "nonton anime, anime sub indo, streaming anime, download anime gratis, anime terbaru, anime jepang, otaku indo, cinembrot anime",
	}
	if y, err := strconv.Atoi(yearStr); err == nil {
		data.CurrentYear = y
	}

	s.PopulateSEO(r, &data)
	s.RenderHTML(w, "anime.html", "layout.html", data)
}

// HandleDramaPendek displays the dedicated Drama Pendek / Mini Series catalog with multi-parameter filtering
func (s *Server) HandleDramaPendek(w http.ResponseWriter, r *http.Request) {
	yearStr := strings.TrimSpace(r.URL.Query().Get("year"))
	genreStr := strings.TrimSpace(r.URL.Query().Get("genre"))
	countryStr := strings.TrimSpace(r.URL.Query().Get("country"))
	sortStr := strings.TrimSpace(r.URL.Query().Get("sort"))

	query := s.db.Model(&model.Movie{}).Where("type = ?", "drama_pendek").Preload("Genres")

	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil && y > 0 {
			query = query.Where("year = ?", y)
		}
	}

	if genreStr != "" {
		var genre model.Genre
		if err := s.db.Where("slug = ? OR name = ?", genreStr, genreStr).First(&genre).Error; err == nil {
			query = query.Joins("JOIN movie_genres ON movie_genres.movie_id = movies.id").
				Where("movie_genres.genre_id = ?", genre.ID)
		}
	}

	if countryStr != "" {
		query = query.Where("country LIKE ?", "%"+countryStr+"%")
	}

	// Sorting
	switch sortStr {
	case "rating_desc":
		query = query.Order("rating desc, id desc")
	case "rating_asc":
		query = query.Order("rating asc, id desc")
	case "year_asc":
		query = query.Order("year asc, id desc")
	case "title_asc":
		query = query.Order("title asc")
	case "views_desc":
		query = query.Order("views desc, id desc")
	default:
		query = query.Order("id desc") // Default: Terbaru ditambahkan
	}

	var totalCount int64
	query.Count(&totalCount)

	var movies []model.Movie
	query.Limit(60).Find(&movies)

	var genres []model.Genre
	s.db.Order("name asc").Find(&genres)

	var years []int
	s.db.Model(&model.Movie{}).Distinct().Where("type = ? AND year > 0", "drama_pendek").Order("year desc").Pluck("year", &years)
	if len(years) == 0 {
		years = []int{2026, 2025, 2024, 2023, 2022}
	}

	var countries []string
	s.db.Model(&model.Movie{}).Distinct().Where("type = ? AND country <> ''", "drama_pendek").Order("country asc").Pluck("country", &countries)
	if len(countries) == 0 {
		countries = []string{"China", "Korea Selatan", "Indonesia", "Thailand"}
	}

	data := PageData{
		Lang:            i18n.GetLang(r),
		Title:           "Serial Drama Pendek & Mini Series Sub Indo Lengkap HD",
		SiteName:        "CINEMBROT",
		ActiveMenu:      "drama_pendek",
		Movies:          movies,
		Genres:          genres,
		Years:           years,
		Countries:       countries,
		CurrentGenre:    genreStr,
		CurrentCountry:  countryStr,
		CurrentSort:     sortStr,
		TotalCount:      totalCount,
		EnableAds:       s.cfg.EnableAds,
		MetaDescription: "Nonton serial drama pendek, mini series, dan drama China/Korea subtitle Indonesia episode lengkap gratis kualitas jernih di CINEMBROT.",
		MetaKeywords:    "nonton drama pendek, mini series sub indo, drama china pendek, drama korea pendek, streaming drama pendek, cinembrot drama",
	}
	if y, err := strconv.Atoi(yearStr); err == nil {
		data.CurrentYear = y
	}

	s.PopulateSEO(r, &data)
	s.RenderHTML(w, "drama_pendek.html", "layout.html", data)
}

// HandleMovieDetail displays movie details, player, and download links
func (s *Server) HandleMovieDetail(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	var movie model.Movie
	err := s.db.Preload("Genres").
		Preload("Directors").
		Preload("Actors").
		Preload("DownloadLinks", "is_valid = ? AND status <> ?", true, "DEAD").
		Preload("StreamLinks", "is_valid = ? AND status <> ?", true, "DEAD").
		Preload("Schedules").
		Preload("Comments", "is_approved = ?", true, func(db *gorm.DB) *gorm.DB {
			return db.Order("id desc")
		}).
		Where("slug = ?", slug).
		First(&movie).Error

	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Update view counter
	s.db.Model(&movie).UpdateColumn("views", movie.Views+1)

	// Jika bahasa aktif adalah English (en), berikan sinopsis resmi English dari TMDb
	lang := i18n.GetLang(r)
	if lang == "en" {
		if enSynopsis := s.tmdbCli.GetEnglishSynopsis(movie.SourceURL, movie.Title, movie.Type); enSynopsis != "" {
			movie.Synopsis = enSynopsis
		}
	}

	// Related movies
	var related []model.Movie
	s.db.Preload("Genres").Where("id <> ?", movie.ID).Order("rating desc, id desc").Limit(6).Find(&related)

	var genres []model.Genre
	s.db.Find(&genres)

	// Embed & Streaming Resolver: populate multi-server streams if not present in database
	tmdbID := embed.ExtractTMDbID(&movie)
	if tmdbID == 0 && s.tmdbCli != nil {
		// Try resolving TMDb ID by searching title
		if movie.Type == "anime" || movie.Type == "drama_pendek" || movie.Type == "series" {
			if tvRes, err := s.tmdbCli.SearchTV(movie.Title); err == nil && len(tvRes.Results) > 0 {
				tmdbID = tvRes.Results[0].ID
			} else if movieRes, err := s.tmdbCli.SearchMovie(movie.Title, movie.Year); err == nil && len(movieRes.Results) > 0 {
				tmdbID = movieRes.Results[0].ID
			}
		} else {
			if movieRes, err := s.tmdbCli.SearchMovie(movie.Title, movie.Year); err == nil && len(movieRes.Results) > 0 {
				tmdbID = movieRes.Results[0].ID
			}
		}
	}

	if len(movie.StreamLinks) == 0 {
		movie.StreamLinks = embed.GenerateMultiServerStreams(&movie, tmdbID, 1, 1)
	}

	// Generate Episode List for anime, drama pendek, and TV series
	var episodesList []int
	if movie.Type == "anime" || movie.Type == "drama_pendek" || movie.Type == "series" {
		totalEps := embed.ExtractEpisodeCount(&movie)
		if totalEps < 1 {
			totalEps = 1
		}
		if totalEps > 100 {
			totalEps = 100 // Safe upper bound for UI display
		}
		for ep := 1; ep <= totalEps; ep++ {
			episodesList = append(episodesList, ep)
		}
	}

	// Generate Anti-Spam Math CAPTCHA Challenge
	captcha := GenerateCaptcha()

	// SEO Rich Metadata Generator
	metaDesc := scraper.CleanHTMLToPlainText(movie.Synopsis)
	if len(metaDesc) > 160 {
		metaDesc = metaDesc[:157] + "..."
	}
	if metaDesc == "" {
		metaDesc = fmt.Sprintf("Nonton film %s (%d) subtitle Indonesia kualitas %s gratis dan legal. Streaming lancar dan download cepat tanpa ribet di CINEMBROT.", movie.Title, movie.Year, movie.Quality)
	}

	var kwList []string
	kwList = append(kwList, fmt.Sprintf("nonton %s", movie.Title), fmt.Sprintf("%s sub indo", movie.Title), fmt.Sprintf("streaming %s", movie.Title), fmt.Sprintf("download %s", movie.Title), fmt.Sprintf("film %s %d", movie.Title, movie.Year), "cinembrot")
	if movie.OriginalTitle != "" && movie.OriginalTitle != movie.Title {
		kwList = append(kwList, movie.OriginalTitle)
	}
	if movie.AlternativeTitles != "" {
		for _, alt := range strings.Split(movie.AlternativeTitles, ",") {
			alt = strings.TrimSpace(alt)
			if alt != "" {
				kwList = append(kwList, alt)
			}
		}
	}
	for _, g := range movie.Genres {
		kwList = append(kwList, g.Name)
	}
	for _, a := range movie.Actors {
		kwList = append(kwList, a.Name)
	}
	metaKeywords := strings.Join(kwList, ", ")

	metaImg := movie.PosterURL
	if metaImg == "" {
		metaImg = movie.ThumbnailURL
	}
	if metaImg == "" {
		metaImg = movie.BackdropURL
	}

	data := PageData{
		Lang:              lang,
		Title:             fmt.Sprintf("Nonton %s (%d) Sub Indo HD - Streaming & Download", movie.Title, movie.Year),
		SiteName:          "CINEMBROT",
		Movie:             &movie,
		Related:           related,
		Genres:            genres,
		EnableAds:         s.cfg.EnableAds,
		EnableComments:    s.cfg.EnableComments,
		ShowTorrentPublic: database.GetShowTorrentPublic(s.db),
		CaptchaQuestion:   captcha.Question,
		CaptchaToken:      captcha.Token,
		EpisodesList:      episodesList,
		CurrentEpisode:    1,
		TMDbID:            tmdbID,
		MetaDescription:   metaDesc,
		MetaKeywords:      metaKeywords,
		MetaImage:         metaImg,
		OGType:            "video.movie",
		JSONLD:            s.GenerateJSONLDMovie(s.GetSiteURL(r), &movie),
	}

	s.PopulateSEO(r, &data)
	s.RenderHTML(w, "detail.html", "layout.html", data)
}

// HandleSubmitComment saves user submitted comment for a movie
func (s *Server) HandleSubmitComment(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.EnableComments || database.GetSetting(s.db, "comments_enabled", "true") == "false" {
		http.Error(w, "Fitur komentar dinonaktifkan oleh administrator.", http.StatusForbidden)
		return
	}

	slug := r.PathValue("slug")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	var movie model.Movie
	if err := s.db.Where("slug = ?", slug).First(&movie).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	_ = r.ParseForm()

	// 1. Verify CAPTCHA
	captchaAnswer := r.FormValue("captcha_answer")
	captchaToken := r.FormValue("captcha_token")
	if !VerifyCaptcha(captchaAnswer, captchaToken) {
		// CAPTCHA invalid / wrong answer -> don't save comment
		http.Redirect(w, r, "/movie/"+slug+"?error=captcha#comments", http.StatusSeeOther)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(r.FormValue("email"))
	content := strings.TrimSpace(r.FormValue("content"))
	ratingStr := strings.TrimSpace(r.FormValue("rating"))

	if name == "" {
		name = "Pengunjung CINEMBROT"
	}

	// Sanitize and clean comment content from raw HTML
	content = scraper.CleanHTMLToPlainText(content)

	if content == "" {
		http.Redirect(w, r, "/movie/"+slug+"#comments", http.StatusSeeOther)
		return
	}

	rating := 10.0
	if rVal, err := strconv.ParseFloat(ratingStr, 64); err == nil && rVal > 0 && rVal <= 10 {
		rating = rVal
	}

	comment := model.Comment{
		MovieID:     movie.ID,
		AuthorName:  name,
		AuthorEmail: email,
		Content:     content,
		Rating:      rating,
		IsApproved:  true,
	}

	_ = s.db.Create(&comment)

	http.Redirect(w, r, "/movie/"+slug+"#comments", http.StatusSeeOther)
}

// HandleYearFilter filters movies by release year
func (s *Server) HandleYearFilter(w http.ResponseWriter, r *http.Request) {
	yearStr := r.PathValue("year")
	http.Redirect(w, r, "/filter?year="+yearStr, http.StatusFound)
}

// HandleGenreFilter filters movies by genre
func (s *Server) HandleGenreFilter(w http.ResponseWriter, r *http.Request) {
	genreSlug := r.PathValue("slug")
	http.Redirect(w, r, "/filter?genre="+genreSlug, http.StatusFound)
}

// HandleFreeFilter displays 100% legal and free movies
func (s *Server) HandleFreeFilter(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/filter?category=free", http.StatusFound)
}

// HandleSearch searches movies by title, original title, or synopsis
func (s *Server) HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	var movies []model.Movie
	if query != "" {
		s.db.Preload("Genres").
			Where("title LIKE ? OR original_title LIKE ? OR synopsis LIKE ?", "%"+query+"%", "%"+query+"%", "%"+query+"%").
			Order("rating desc, id desc").
			Find(&movies)
	}

	var genres []model.Genre
	s.db.Find(&genres)

	data := PageData{
		Lang:        i18n.GetLang(r),
		Title:       "Hasil Pencarian: " + query,
		SiteName:    "CINEMBROT",
		SearchQuery: query,
		Movies:      movies,
		Genres:          genres,
		TotalCount:      int64(len(movies)),
		EnableAds:       s.cfg.EnableAds,
		MetaDescription: fmt.Sprintf("Hasil pencarian untuk '%s' di CINEMBROT. Nonton streaming dan download film, anime, serta drama pendek subtitle Indonesia kualitas HD gratis.", query),
	}

	s.PopulateSEO(r, &data)
	s.RenderHTML(w, "list.html", "layout.html", data)
}

// HandleAPIMovies returns JSON movie feed
func (s *Server) HandleAPIMovies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var movies []model.Movie
	s.db.Preload("Genres").Preload("DownloadLinks").Preload("StreamLinks").Limit(50).Find(&movies)
	json.NewEncoder(w).Encode(movies)
}

