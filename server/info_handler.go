package server

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"cinembrot/database"
)

type InfoPageData struct {
	SiteName           string
	SiteURL            string
	AdsEnabled         bool
	PopunderScript     string
	SocialBarScript    string
	SmartlinkURL       string
	TopBannerDesktop   template.HTML
	TopBannerMobile    template.HTML
	NativeBannerCode   template.HTML
	FooterBannerCode   template.HTML
	GoogleVerification string
	BingVerification   string
	YandexVerification string
	GoogleAnalyticsID  string
	CustomHeadCode     template.HTML
	CustomFooterCode   template.HTML
	MetaDescription    string
	MetaKeywords       string
}

// HandleInfoDomain serves the official information landing page (cinembrot.my.id)
// dynamically populated from MariaDB system_settings
func (s *Server) HandleInfoDomain(w http.ResponseWriter, r *http.Request) {
	settings := database.GetAllSettings(s.db)

	getVal := func(key, fallback string) string {
		if val, ok := settings[key]; ok && strings.TrimSpace(val) != "" {
			return val
		}
		return fallback
	}

	adsEnabled := getVal("info_ads_enabled", "true") != "false"

	data := InfoPageData{
		SiteName:           getVal("site_name", "CINEMBROT"),
		SiteURL:            getVal("site_url", "https://cinembrot.web.id"),
		AdsEnabled:         adsEnabled,
		PopunderScript:     getVal("info_adsterra_popunder_code", "https://ripenhopperwitty.com/54/e2/d0/54e2d041392f6fe60241fb7998b44e22.js"),
		SocialBarScript:    getVal("info_adsterra_socialbar_code", "https://ripenhopperwitty.com/e8/02/6c/e8026c7abe894a6b7c4f480910f52fc3.js"),
		SmartlinkURL:       getVal("info_adsterra_smartlink_url", "https://ripenhopperwitty.com/dshsjtw4mm?key=b8e97cccf7d6fdfe0a74963ec89f2ae2"),
		TopBannerDesktop:   template.HTML(getVal("info_adsterra_banner_top_desktop", "")),
		TopBannerMobile:    template.HTML(getVal("info_adsterra_banner_top_mobile", "")),
		NativeBannerCode:   template.HTML(getVal("info_adsterra_native_banner_code", "")),
		FooterBannerCode:   template.HTML(getVal("info_adsterra_banner_footer_code", "")),
		GoogleVerification: getVal("info_google_site_verification", getVal("google_site_verification", "")),
		BingVerification:   getVal("info_bing_site_verification", getVal("bing_site_verification", "")),
		YandexVerification: getVal("info_yandex_verification", getVal("yandex_verification", "")),
		GoogleAnalyticsID:  getVal("info_google_analytics_id", "G-LN4TYGYDT2"),
		CustomHeadCode:     template.HTML(getVal("info_custom_head_code", "")),
		CustomFooterCode:   template.HTML(getVal("info_custom_footer_code", "")),
		MetaDescription:    getVal("info_meta_description", "Informasi link domain resmi aktif CINEMBROT (cinembrot.web.id). Platform nonton film, anime, drama korea, dan box office streaming gratis subtitle Indonesia kualitas Full HD."),
		MetaKeywords:       getVal("info_meta_keywords", "cinembrot, cinembrot.web.id, cinembrot.my.id, link cinembrot, streaming film gratis, nonton anime sub indo, lk21, filmapik, layarkaca21"),
	}

	infoPath := filepath.Join("public", "info", "index.html")
	content, err := os.ReadFile(infoPath)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html><html><head><title>CINEMBROT Info</title></head><body><h1>Domain Resmi CINEMBROT: <a href="https://cinembrot.web.id">cinembrot.web.id</a></h1></body></html>`))
		return
	}

	funcMap := template.FuncMap{
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
		"safeJS":   func(s string) template.JS { return template.JS(s) },
	}

	tmpl, err := template.New("info_index").Funcs(funcMap).Parse(string(content))
	if err != nil {
		log.Printf("[INFO SERVER ERROR] Parse template info error: %v\n", err)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("[INFO SERVER ERROR] Execute template info error: %v\n", err)
	}
}
