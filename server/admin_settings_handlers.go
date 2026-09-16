package server

import (
	"log"
	"net/http"
	"strings"

	"cinembrot/database"
)

// HandleAdminSettings renders the dynamic website settings & Adsterra monetization control page
func (s *Server) HandleAdminSettings(w http.ResponseWriter, r *http.Request) {
	settings := database.GetAllSettings(s.db)

	var successMsg string
	if r.URL.Query().Get("saved") == "1" {
		successMsg = "Pengaturan website dan unit iklan Adsterra berhasil diperbarui ke database!"
	}

	data := AdminPageData{
		Title:      "Pengaturan Website & Monetisasi Iklan",
		ActiveMenu: "settings",
		User:       s.GetLoggedInUser(r),
		Settings:   settings,
		SuccessMsg: successMsg,
	}

	s.RenderHTML(w, "admin_settings.html", "admin_layout.html", data)
}

// HandleAdminSaveSettings processes the submitted settings form and persists values into MariaDB
func (s *Server) HandleAdminSaveSettings(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Gagal memproses formulir: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Helper function to extract toggle/checkbox status
	checkboxVal := func(name string) string {
		if r.FormValue(name) == "true" || r.FormValue(name) == "on" || r.FormValue(name) == "1" {
			return "true"
		}
		return "false"
	}

	// 1. Iklan Adsterra & Monetisasi
	adsEnabled := checkboxVal("ads_enabled")
	popunderCode := strings.TrimSpace(r.FormValue("adsterra_popunder_code"))
	socialbarCode := strings.TrimSpace(r.FormValue("adsterra_socialbar_code"))
	banner728Code := strings.TrimSpace(r.FormValue("adsterra_banner_728_code"))
	banner468Code := strings.TrimSpace(r.FormValue("adsterra_banner_468_code"))
	banner300Code := strings.TrimSpace(r.FormValue("adsterra_banner_300_code"))
	banner160x600Code := strings.TrimSpace(r.FormValue("adsterra_banner_160x600_code"))
	banner160x300Code := strings.TrimSpace(r.FormValue("adsterra_banner_160x300_code"))
	banner320Code := strings.TrimSpace(r.FormValue("adsterra_banner_320_code"))
	nativeBannerCode := strings.TrimSpace(r.FormValue("adsterra_native_banner_code"))
	smartlinkURL := strings.TrimSpace(r.FormValue("adsterra_smartlink_url"))

	// 2. Fitur Translate & Multi-Bahasa
	translateEnabled := checkboxVal("translate_enabled")
	googleTranslateEnabled := checkboxVal("google_translate_enabled")
	defaultLang := strings.TrimSpace(r.FormValue("default_language"))
	if defaultLang != "en" {
		defaultLang = "id"
	}

	// 3. Fitur Interaksi & Publik
	commentsEnabled := checkboxVal("comments_enabled")
	showTorrentPublic := checkboxVal("show_torrent_public")

	// 4. Identitas & Branding Website
	siteName := strings.TrimSpace(r.FormValue("site_name"))
	if siteName == "" {
		siteName = "CINEMBROT"
	}
	siteTagline := strings.TrimSpace(r.FormValue("site_tagline"))

	// 5. Search Engine Optimization (SEO) & Webmaster Verifications
	siteURL := strings.TrimRight(strings.TrimSpace(r.FormValue("site_url")), "/")
	metaDescription := strings.TrimSpace(r.FormValue("meta_description"))
	metaKeywords := strings.TrimSpace(r.FormValue("meta_keywords"))
	googleVerification := strings.TrimSpace(r.FormValue("google_site_verification"))
	bingVerification := strings.TrimSpace(r.FormValue("bing_site_verification"))
	yandexVerification := strings.TrimSpace(r.FormValue("yandex_verification"))

	// Save all to MariaDB system_settings table
	_ = database.SaveSetting(s.db, "ads_enabled", adsEnabled, "Saklar Master ON/OFF Iklan di Website")
	_ = database.SaveSetting(s.db, "adsterra_popunder_code", popunderCode, "Kode Script Iklan Adsterra Popunder")
	_ = database.SaveSetting(s.db, "adsterra_socialbar_code", socialbarCode, "Kode Script Iklan Adsterra Social Bar")
	_ = database.SaveSetting(s.db, "adsterra_banner_728_code", banner728Code, "Kode Script Iklan Adsterra Top Banner 728x90")
	_ = database.SaveSetting(s.db, "adsterra_banner_468_code", banner468Code, "Kode Script Iklan Adsterra Banner 468x60")
	_ = database.SaveSetting(s.db, "adsterra_banner_300_code", banner300Code, "Kode Script Iklan Adsterra Medium Rectangle 300x250")
	_ = database.SaveSetting(s.db, "adsterra_banner_160x600_code", banner160x600Code, "Kode Script Iklan Adsterra Left Skyscraper 160x600")
	_ = database.SaveSetting(s.db, "adsterra_banner_160x300_code", banner160x300Code, "Kode Script Iklan Adsterra Right Skyscraper 160x300")
	_ = database.SaveSetting(s.db, "adsterra_banner_320_code", banner320Code, "Kode Script Iklan Adsterra Mobile Sticky Banner 320x50")
	_ = database.SaveSetting(s.db, "adsterra_native_banner_code", nativeBannerCode, "Kode Script Iklan Adsterra Native Banner Rekomendasi")
	_ = database.SaveSetting(s.db, "adsterra_smartlink_url", smartlinkURL, "URL Direct Adsterra Smartlink (Download / Streaming)")

	_ = database.SaveSetting(s.db, "translate_enabled", translateEnabled, "Saklar ON/OFF Fitur Multi-Bahasa / Translate")
	_ = database.SaveSetting(s.db, "google_translate_enabled", googleTranslateEnabled, "Saklar ON/OFF Mesin Google Website Translator")
	_ = database.SaveSetting(s.db, "default_language", defaultLang, "Bahasa Default Website (id / en)")

	_ = database.SaveSetting(s.db, "comments_enabled", commentsEnabled, "Saklar ON/OFF Kolom Komentar Penonton")
	_ = database.SaveSetting(s.db, "show_torrent_public", showTorrentPublic, "Tampilkan link torrent mentah di halaman publik film (true/false)")

	_ = database.SaveSetting(s.db, "site_name", siteName, "Nama Brand Website")
	_ = database.SaveSetting(s.db, "site_tagline", siteTagline, "Tagline atau Slogan Website")

	_ = database.SaveSetting(s.db, "site_url", siteURL, "URL Publik Utama Website (contoh: https://cinembrot.my.id)")
	_ = database.SaveSetting(s.db, "meta_description", metaDescription, "Meta Description Default Mesin Pencari Google/Bing")
	_ = database.SaveSetting(s.db, "meta_keywords", metaKeywords, "Meta Keywords Default Website")
	_ = database.SaveSetting(s.db, "google_site_verification", googleVerification, "Kode Verifikasi Google Search Console")
	_ = database.SaveSetting(s.db, "bing_site_verification", bingVerification, "Kode Verifikasi Bing Webmaster Tools")
	_ = database.SaveSetting(s.db, "yandex_verification", yandexVerification, "Kode Verifikasi Yandex Webmaster")

	log.Printf("[CMS SETTINGS] ⚙️ Pengaturan website & SEO berhasil diperbarui oleh admin '%s'\n", s.GetLoggedInUser(r).Username)

	http.Redirect(w, r, "/admin/settings?saved=1", http.StatusSeeOther)
}
