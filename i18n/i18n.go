package i18n

import (
	"net/http"
	"strings"
	"time"
)

// Supported languages
const (
	LangID = "id"
	LangEN = "en"
)

// translations dictionary for Indonesian (Default) and English
var dictionary = map[string]map[string]string{
	LangID: {
		// Navbar & General
		"nav.home":         "Home",
		"nav.anime":        "Anime",
		"nav.drama_pendek": "Drama Pendek",
		"nav.popular":      "Populer",
		"nav.filter":       "Filter Lengkap",
		"nav.search_ph":    "Cari judul film, anime, aktor...",
		"nav.theme_toggle": "Ganti Tema",
		"nav.free_movies":  "Film Legal Gratis",

		// Hero Carousel & Actions
		"hero.watch_now":    "Tonton Sekarang",
		"hero.download":     "Unduh Video",
		"hero.details":      "Lihat Detail",
		"hero.synopsis":     "Sinopsis",
		"hero.rating":       "Rating",
		"hero.quality":      "Kualitas",

		// Home Filters
		"filter.title":       "Filter & Cari Film",
		"filter.subtitle":    "Filter berdasarkan Tahun, Genre, Negara, dan Urutan",
		"filter.year":        "Tahun Rilis",
		"filter.genre":       "Genre Film",
		"filter.country":     "Negara Asal",
		"filter.sort":        "Urutkan Hasil",
		"filter.all_years":   "Semua Tahun",
		"filter.all_genres":  "Semua Genre",
		"filter.all_country": "Semua Negara",
		"filter.sort_newest": "Rilis Terbaru",
		"filter.sort_rating": "Rating Tertinggi",
		"filter.sort_popular": "Terpopuler",
		"filter.sort_title":  "Berdasarkan Abjad",
		"filter.btn_apply":   "Filter",
		"filter.btn_reset":   "Reset",

		// Catalog Sections
		"section.latest_movies":   "Katalog Film Terbaru",
		"section.showing_movies":  "Menampilkan %d film",
		"section.no_movies":       "Belum ada film yang ditemukan.",
		"section.latest_anime":    "Katalog Anime Terbaru",
		"section.latest_drama":    "Katalog Drama Pendek & Serial Asia",
		"section.legal_free":      "Film Legal & Domain Publik Gratis",
		"section.similar_movies":  "Rekomendasi Film Terkait",

		// Anime & Drama Specials
		"anime.badge":             "JAPANESE ANIMATION",
		"anime.hero_title":        "Streaming & Koleksi Anime Terlengkap",
		"anime.hero_desc":         "Jelajahi anime populer, serial on-going musiman, dan anime klasik berkualitas tinggi dengan takarir bahasa Indonesia dan Inggris.",
		"anime.cat_top":           "Top Anime Terpopuler",
		"anime.cat_seasonal":      "Anime Musim Ini (On-Going)",

		"drama.badge":             "ASIAN SHORT DRAMA & SERIES",
		"drama.hero_title":        "Drama Pendek & Serial Asia Terpopuler",
		"drama.hero_desc":         "Koleksi drama Korea (K-Drama), drama China (C-Drama), serial Jepang, dan drama Thailand dengan plot seru dan cepat.",
		"drama.lang_all":          "Semua Bahasa",
		"drama.lang_ko":           "Drama Korea (K-Drama)",
		"drama.lang_zh":           "Drama China (C-Drama)",
		"drama.lang_ja":           "Drama Jepang (J-Drama)",
		"drama.lang_th":           "Drama Thailand",

		// Detail Page
		"detail.info":             "Informasi Film",
		"detail.duration":         "Durasi",
		"detail.release_date":     "Tanggal Rilis",
		"detail.country":          "Negara",
		"detail.language":         "Bahasa",
		"detail.status":           "Status",
		"detail.status_ongoing":   "Sedang Tayang (On-Going)",
		"detail.status_released":  "Selesai / Tamat",
		"detail.directors":        "Sutradara",
		"detail.cast":             "Pemeran Utama",
		"detail.trailer":          "Tonton Trailer Resmi",
		"detail.downloads":        "Pilihan Tautan Download",
		"detail.subtitles":        "Pilihan Tautan Subtitle",
		"detail.no_downloads":     "Belum ada link unduh langsung. Gunakan link subtitle di bawah.",
		"detail.download_btn":     "Unduh Sekarang",
		"detail.copy_link":        "Salin Tautan",
		"detail.copied":           "Tautan berhasil disalin!",
		"detail.stream_player":    "Pemutar Streaming",
		"detail.free_legal_badge": "100% Legal & Bebas Hak Cipta",
		"detail.comments":         "Komentar & Diskusi",

		// Footer & Branding
		"footer.tagline":          "Platform Streaming & Download Film Gratis Legal",
		"footer.desc":             "Didukung oleh Internet Archive, Blender Open Movies, dan TMDb REST API dengan integrasi subtitle otomatis.",
		"footer.legal":            "Film Legal Gratis",
		"footer.catalog":          "Katalog Lengkap",
		"footer.copyright":        "Hak Cipta Dilindungi Undang-Undang.",
	},
	LangEN: {
		// Navbar & General
		"nav.home":         "Home",
		"nav.anime":        "Anime",
		"nav.drama_pendek": "Short Dramas",
		"nav.popular":      "Popular",
		"nav.filter":       "All Filters",
		"nav.search_ph":    "Search movies, anime, actors...",
		"nav.theme_toggle": "Toggle Theme",
		"nav.free_movies":  "Free Legal Movies",

		// Hero Carousel & Actions
		"hero.watch_now":    "Watch Now",
		"hero.download":     "Download Video",
		"hero.details":      "View Details",
		"hero.synopsis":     "Synopsis",
		"hero.rating":       "Rating",
		"hero.quality":      "Quality",

		// Home Filters
		"filter.title":       "Filter & Search Movies",
		"filter.subtitle":    "Filter by Release Year, Genre, Country, and Sorting",
		"filter.year":        "Release Year",
		"filter.genre":       "Movie Genre",
		"filter.country":     "Country of Origin",
		"filter.sort":        "Sort Results",
		"filter.all_years":   "All Years",
		"filter.all_genres":  "All Genres",
		"filter.all_country": "All Countries",
		"filter.sort_newest": "Latest Release",
		"filter.sort_rating": "Highest Rating",
		"filter.sort_popular": "Most Popular",
		"filter.sort_title":  "Alphabetical Order",
		"filter.btn_apply":   "Apply Filter",
		"filter.btn_reset":   "Reset",

		// Catalog Sections
		"section.latest_movies":   "Latest Movie Catalog",
		"section.showing_movies":  "Showing %d titles",
		"section.no_movies":       "No movies found matching criteria.",
		"section.latest_anime":    "Latest Anime Catalog",
		"section.latest_drama":    "Popular Asian Short Dramas & Series",
		"section.legal_free":      "Free Public Domain & Legal Films",
		"section.similar_movies":  "Recommended Related Titles",

		// Anime & Drama Specials
		"anime.badge":             "JAPANESE ANIMATION",
		"anime.hero_title":        "Watch & Collect Trending Anime Series",
		"anime.hero_desc":         "Explore top-rated anime, seasonal on-going releases, and all-time classics with Indonesian and English subtitle support.",
		"anime.cat_top":           "Top Rated Anime",
		"anime.cat_seasonal":      "Seasonal Anime (On-Going)",

		"drama.badge":             "ASIAN SHORT DRAMA & SERIES",
		"drama.hero_title":        "Trending Asian Short Dramas & Series",
		"drama.hero_desc":         "Rich collection of Korean Dramas (K-Drama), Chinese Dramas (C-Drama), Japanese Series, and Thai Dramas with fast-paced storytelling.",
		"drama.lang_all":          "All Languages",
		"drama.lang_ko":           "Korean Drama (K-Drama)",
		"drama.lang_zh":           "Chinese Drama (C-Drama)",
		"drama.lang_ja":           "Japanese Drama (J-Drama)",
		"drama.lang_th":           "Thai Drama",

		// Detail Page
		"detail.info":             "Movie Information",
		"detail.duration":         "Runtime",
		"detail.release_date":     "Release Date",
		"detail.country":          "Country",
		"detail.language":         "Language",
		"detail.status":           "Status",
		"detail.status_ongoing":   "Airing (On-Going)",
		"detail.status_released":  "Completed / Ended",
		"detail.directors":        "Director(s)",
		"detail.cast":             "Main Cast",
		"detail.trailer":          "Watch Official Trailer",
		"detail.downloads":        "Download Links",
		"detail.subtitles":        "Subtitle Links",
		"detail.no_downloads":     "No direct download links available yet. Use subtitle sources below.",
		"detail.download_btn":     "Download Now",
		"detail.copy_link":        "Copy Link",
		"detail.copied":           "Link copied to clipboard!",
		"detail.stream_player":    "Streaming Player",
		"detail.free_legal_badge": "100% Legal & Public Domain",
		"detail.comments":         "Comments & Discussion",

		// Footer & Branding
		"footer.tagline":          "Free & Legal Movie Streaming & Download Platform",
		"footer.desc":             "Powered by Internet Archive, Blender Open Movies, and TMDb REST API with automated dual-language subtitle generation.",
		"footer.legal":            "Free Legal Movies",
		"footer.catalog":          "Full Catalog",
		"footer.copyright":        "All rights reserved.",
	},
}

// GetLang extracts the user language from query param (?lang=) or cookie ("cinembrot_lang").
// Default is "id" (Indonesian).
func GetLang(r *http.Request) string {
	// 1. Query parameter has highest priority
	if qLang := strings.ToLower(r.URL.Query().Get("lang")); qLang == LangEN || qLang == LangID {
		return qLang
	}

	// 2. Cookie priority
	if cookie, err := r.Cookie("cinembrot_lang"); err == nil {
		cLang := strings.ToLower(cookie.Value)
		if cLang == LangEN || cLang == LangID {
			return cLang
		}
	}

	// 3. Fallback to default Indonesian
	return LangID
}

// SetLangCookie sets the language preference cookie for 1 year
func SetLangCookie(w http.ResponseWriter, lang string) {
	if lang != LangEN && lang != LangID {
		lang = LangID
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "cinembrot_lang",
		Value:    lang,
		Path:     "/",
		Expires:  time.Now().Add(365 * 24 * time.Hour),
		MaxAge:   365 * 24 * 3600,
		HttpOnly: false, // Accessible to client-side scripts if needed
		SameSite: http.SameSiteLaxMode,
	})
}

// T translates a key according to the selected language.
// If key is not found in the selected language, it falls back to Indonesian, then returns the key itself.
func T(lang string, key string) string {
	if lang != LangEN && lang != LangID {
		lang = LangID
	}

	if val, ok := dictionary[lang][key]; ok && val != "" {
		return val
	}

	// Fallback to Indonesian
	if val, ok := dictionary[LangID][key]; ok && val != "" {
		return val
	}

	return key
}
