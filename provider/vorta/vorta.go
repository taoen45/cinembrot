package vorta

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultBaseURL = "https://api.vorta.tech"
	DefaultSecret  = "b15bc8fff70cc4acdb67f6e8a274c065c1c8d579c2297782ac5bb017d1871ae2"
)

// Client menangani request terautentikasi ke Vorta API
type Client struct {
	BaseURL    string
	SecretKey  string
	HTTPClient *http.Client
}

// NewClient menginisialisasi client Vorta dengan konfigurasi env atau default secret
func NewClient() *Client {
	secret := os.Getenv("VORTA_API_SECRET")
	if secret == "" {
		secret = DefaultSecret
	}

	return &Client{
		BaseURL:   DefaultBaseURL,
		SecretKey: secret,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// generateSignature menghasilkan signature HMAC-SHA256: method + ":" + path + ":" + timestamp
func (c *Client) generateSignature(method, path, timestamp string) string {
	payload := fmt.Sprintf("%s:%s:%s", method, path, timestamp)
	mac := hmac.New(sha256.New, []byte(c.SecretKey))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// executeRequest menjalankan HTTP request dengan header X-Timestamp dan X-Signature
func (c *Client) executeRequest(method, path string) ([]byte, error) {
	ts := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	sig := c.generateSignature(method, path, ts)

	reqURL := c.BaseURL + path
	req, err := http.NewRequest(method, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat request: %w", err)
	}

	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Signature", sig)
	req.Header.Set("User-Agent", "Cinembrot-Vorta-Client/1.0")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("koneksi Vorta API error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca body response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, fmt.Errorf("vorta API mengembalikan HTTP %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// KeyStatusResponse merepresentasikan respons status kuota API key
type KeyStatusResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Label           string `json:"label"`
		IsActive        bool   `json:"is_active"`
		IsExpired       bool   `json:"is_expired"`
		Limit           int    `json:"limit"`
		Used            int    `json:"used"`
		Remaining       int    `json:"remaining"`
		ResetsIn        string `json:"resets_in"`
		Plan            string `json:"plan"`
		HasVideoAccess  bool   `json:"has_video_access"`
		TotalPlatforms  int    `json:"total_platforms"`
	} `json:"data"`
}

// GetKeyStatus mengecek status, kuota harian, dan masa aktif lisensi API
func (c *Client) GetKeyStatus() (*KeyStatusResponse, error) {
	body, err := c.executeRequest("GET", "/api/v2/key/status")
	if err != nil {
		return nil, err
	}

	var res KeyStatusResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("gagal unmarshal key status: %w", err)
	}

	return &res, nil
}

// CategoryItem merepresentasikan platform yang didukung (meloshort, shortmax, mydrama)
type CategoryItem struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	LogoURL     string `json:"logo_url"`
	Order       int    `json:"order"`
	IsActive    bool   `json:"is_active"`
	Is18Plus    bool   `json:"is_18_plus"`
}

type CategoriesResponse struct {
	Success bool           `json:"success"`
	Data    []CategoryItem `json:"data"`
}

// GetCategories mengambil daftar platform yang aktif
func (c *Client) GetCategories() ([]CategoryItem, error) {
	body, err := c.executeRequest("GET", "/api/v2/categories")
	if err != nil {
		return nil, err
	}

	var res CategoriesResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("gagal unmarshal categories: %w", err)
	}

	return res.Data, nil
}

// DramaItem merepresentasikan ringkasan drama pendek
type DramaItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Author   string `json:"author"`
	Cover    string `json:"cover"`
	Status   string `json:"status"`
	Chapters string `json:"chapters"`
	Views    string `json:"views"`
}

// HomeResponse merepresentasikan respons feed beranda
type HomeResponse struct {
	Success bool        `json:"success"`
	Data    []DramaItem `json:"data"`
}

// GetHomeFeed mengambil daftar drama terpopuler di platform tertentu
func (c *Client) GetHomeFeed(platform, lang string) ([]DramaItem, error) {
	if platform == "" {
		platform = "meloshort"
	}
	if lang == "" {
		lang = "id"
	}

	path := fmt.Sprintf("/api/v2/home?category_p=%s&lang=%s", url.QueryEscape(platform), url.QueryEscape(lang))
	body, err := c.executeRequest("GET", path)
	if err != nil {
		return nil, err
	}

	var res HomeResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("gagal unmarshal home feed: %w", err)
	}

	return res.Data, nil
}

// DiscoverResponse merepresentasikan respons discovery katalog drama
type DiscoverResponse struct {
	Success  bool        `json:"success"`
	Platform string      `json:"platform"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	HasMore  bool        `json:"hasMore"`
	Limit    int         `json:"limit"`
	Language string      `json:"language"`
	Data     []DramaItem `json:"data"`
}

// GetDiscover mengambil katalog judul drama dari endpoint /api/v2/discover
func (c *Client) GetDiscover(platform, lang string, page int) (*DiscoverResponse, error) {
	if platform == "" {
		platform = "meloshort"
	}
	if lang == "" {
		lang = "id"
	}
	if page < 1 {
		page = 1
	}

	path := fmt.Sprintf("/api/v2/discover?category_p=%s&lang=%s&page=%d", url.QueryEscape(platform), url.QueryEscape(lang), page)
	body, err := c.executeRequest("GET", path)
	if err != nil {
		return nil, err
	}

	var res DiscoverResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("gagal unmarshal discover response: %w", err)
	}

	return &res, nil
}

// Search mencari drama pendek berdasarkan kata kunci dengan jeda aman anti-brute force
func (c *Client) Search(platform, query, lang string, page int) (*DiscoverResponse, error) {
	if platform == "" {
		platform = "meloshort"
	}
	if lang == "" {
		lang = "id"
	}
	if page < 1 {
		page = 1
	}

	path := fmt.Sprintf("/api/v2/search?category_p=%s&q=%s&lang=%s&page=%d", url.QueryEscape(platform), url.QueryEscape(query), url.QueryEscape(lang), page)
	body, err := c.executeRequest("GET", path)
	if err != nil {
		return nil, err
	}

	var res DiscoverResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("gagal unmarshal search results: %w", err)
	}

	return &res, nil
}

// ChapterInfo merepresentasikan episode dalam drama
type ChapterInfo struct {
	ID       string `json:"id"`
	Index    int    `json:"index"`
	IsLocked bool   `json:"is_locked"`
}

// DramaDetail merepresentasikan informasi lengkap judul dan daftar episode
type DramaDetail struct {
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	Cover         string        `json:"cover"`
	Synopsis      string        `json:"synopsis"`
	Author        string        `json:"author"`
	Genres        []string      `json:"genres"`
	TotalEpisodes int           `json:"total_episodes"`
	Status        string        `json:"status"`
	Chapters      []ChapterInfo `json:"chapters"`
}

type DetailResponse struct {
	Success bool        `json:"success"`
	Data    DramaDetail `json:"data"`
}

// GetDetail mengambil sinopsis dan daftar episode suatu judul
func (c *Client) GetDetail(platform, dramaID, lang string) (*DramaDetail, error) {
	if lang == "" {
		lang = "id"
	}
	path := fmt.Sprintf("/api/v2/detail?category_p=%s&id=%s&lang=%s", url.QueryEscape(platform), url.QueryEscape(dramaID), url.QueryEscape(lang))
	body, err := c.executeRequest("GET", path)
	if err != nil {
		return nil, err
	}

	var res DetailResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("gagal unmarshal detail drama: %w", err)
	}

	return &res.Data, nil
}

// VideoStreamItem merepresentasikan resolusi dan URL streaming
type VideoStreamItem struct {
	Quality          string `json:"quality"`
	Resolution       string `json:"resolution"`
	URL              string `json:"url"`
	Encode           string `json:"encode"`
	Encrypted        bool   `json:"encrypted"`
	EncryptionMethod string `json:"encryptionMethod"`
	Key              string `json:"key"`
	KID              string `json:"kid"`
}

// SubtitleItem merepresentasikan metadata track subtitle resmi
type SubtitleItem struct {
	Language     string `json:"language"`
	LanguageCode string `json:"languageCode"`
	Format       string `json:"format"`
	URL          string `json:"url"`
	Index        int    `json:"index"`
}

// VideoStreamData merepresentasikan data stream episode
type VideoStreamData struct {
	Streams       []VideoStreamItem `json:"streams"`
	Subtitles     []SubtitleItem    `json:"subtitles"`
	EpisodeIndex  int               `json:"episode_index"`
	TotalEpisodes int               `json:"total_episodes"`
	NextVideoID   string            `json:"next_video_id"`
	Title         string            `json:"title"`
	Duration      int               `json:"duration"`
	IsFree        bool              `json:"is_free"`
}

type VideoStreamResponse struct {
	Success bool            `json:"success"`
	Data    VideoStreamData `json:"data"`
}

// ResolveLinkData merepresentasikan hasil resolve URL resmi
type ResolveLinkData struct {
	CategoryP string      `json:"category_p"`
	Platform  string      `json:"platform"`
	Language  string      `json:"language"`
	Data      DramaDetail `json:"data"`
}

type ResolveLinkResponse struct {
	Success   bool            `json:"success"`
	CategoryP string          `json:"category_p"`
	Platform  string          `json:"platform"`
	Language  string          `json:"language"`
	Data      DramaDetail     `json:"data"`
}

// ResolveLink mengurai URL web resmi (misal: my-drama.com/series/...) menjadi metadata film
func (c *Client) ResolveLink(rawURL, lang string) (*DramaDetail, string, error) {
	if lang == "" {
		lang = "id"
	}
	path := fmt.Sprintf("/api/v2/resolve-link?url=%s&lang=%s", url.QueryEscape(rawURL), url.QueryEscape(lang))
	body, err := c.executeRequest("GET", path)
	if err != nil {
		return nil, "", err
	}

	var res ResolveLinkResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, "", fmt.Errorf("gagal unmarshal resolve link: %w", err)
	}

	return &res.Data, res.CategoryP, nil
}

// LanguageItem merepresentasikan bahasa/subtitle resmi platform
type LanguageItem struct {
	Code   string `json:"code"`
	Label  string `json:"label"`
	Native string `json:"native"`
	Flag   string `json:"flag"`
}

type LanguagesResponse struct {
	Success  bool           `json:"success"`
	Platform string         `json:"platform"`
	Default  string         `json:"default"`
	Data     []LanguageItem `json:"data"`
}

var (
	langsCacheMutex sync.RWMutex
	langsCache      = make(map[string][]LanguageItem)
)

// GetLanguages mengambil daftar bahasa resmi dari platform tertentu (dengan in-memory caching)
func (c *Client) GetLanguages(platform string) ([]LanguageItem, error) {
	platform = strings.ToLower(strings.TrimSpace(platform))
	if platform == "" {
		return nil, nil
	}

	langsCacheMutex.RLock()
	if cached, ok := langsCache[platform]; ok && len(cached) > 0 {
		langsCacheMutex.RUnlock()
		return cached, nil
	}
	langsCacheMutex.RUnlock()

	path := fmt.Sprintf("/api/v2/languages?category_p=%s", url.QueryEscape(platform))
	body, err := c.executeRequest("GET", path)
	if err != nil {
		return nil, err
	}

	var res LanguagesResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("gagal unmarshal languages: %w", err)
	}

	langsCacheMutex.Lock()
	langsCache[platform] = res.Data
	langsCacheMutex.Unlock()

	return res.Data, nil
}

// GetVideoStream mengambil URL streaming (HLS/M3U8 atau MP4) per episode secara on-demand
func (c *Client) GetVideoStream(platform, dramaID, chapterID, lang string) (*VideoStreamData, error) {
	if lang == "" {
		lang = "id"
	}
	path := fmt.Sprintf("/api/v2/video?category_p=%s&id=%s&chapterId=%s&lang=%s", url.QueryEscape(platform), url.QueryEscape(dramaID), url.QueryEscape(chapterID), url.QueryEscape(lang))
	body, err := c.executeRequest("GET", path)
	if err != nil {
		return nil, err
	}

	var res VideoStreamResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("gagal unmarshal video stream: %w", err)
	}

	return &res.Data, nil
}
