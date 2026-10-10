package hoster

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"cinembrot/model"
	"cinembrot/provider/aggregator"

	"gorm.io/gorm"
)

// AccountInfo represents account statistics for DoodStream / Streamtape
type AccountInfo struct {
	Provider    string `json:"provider"`
	Email       string `json:"email"`
	Balance     string `json:"balance"`
	StorageUsed string `json:"storage_used,omitempty"`
	StorageLeft string `json:"storage_left,omitempty"`
	Status      string `json:"status"` // OK, ERROR, UNCONFIGURED
	Message     string `json:"message"`
}

// Client manages integration with PPD Video Hosters (DoodStream & Streamtape)
type Client struct {
	DoodStreamKey   string
	StreamtapeLogin string
	StreamtapeKey   string
	httpClient      *http.Client
	aggCli          *aggregator.Client
}

// NewClient creates a new video hoster API client with robust timeouts, redirect handling, and DNS spoofing bypass
func NewClient(doodKey, stLogin, stKey string, aggCli ...*aggregator.Client) *Client {
	dialer := &net.Dialer{
		Timeout:   15 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Toleransi expired certificate pada host lingkungan lokal
		},
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Bypass ISP DNS Hijacking / Internet Positif Indonesia untuk Streamtape API
			// Domain api.streamtape.com dialihkan ISP lokal ke aduankonten/internetpositif.
			// Kita arahkan langsung ke IP resmi Cloudflare untuk Streamtape (104.21.96.46).
			if strings.HasPrefix(addr, "api.streamtape.com:") || strings.HasPrefix(addr, "streamtape.com:") {
				parts := strings.Split(addr, ":")
				port := "443"
				if len(parts) > 1 {
					port = parts[1]
				}
				return dialer.DialContext(ctx, network, "104.21.96.46:"+port)
			}
			return dialer.DialContext(ctx, network, addr)
		},
		DisableKeepAlives: false,
	}

	var aggregatorClient *aggregator.Client
	if len(aggCli) > 0 && aggCli[0] != nil {
		aggregatorClient = aggCli[0]
	}

	return &Client{
		DoodStreamKey:   strings.TrimSpace(doodKey),
		StreamtapeLogin: strings.TrimSpace(stLogin),
		StreamtapeKey:   strings.TrimSpace(stKey),
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   25 * time.Second,
		},
		aggCli: aggregatorClient,
	}
}

// CheckDoodStream verifies credentials and checks balance on DoodStream
func (c *Client) CheckDoodStream() (*AccountInfo, error) {
	if c.DoodStreamKey == "" {
		return &AccountInfo{
			Provider: "DoodStream",
			Status:   "UNCONFIGURED",
			Message:  "API Key DoodStream belum dikonfigurasi.",
		}, nil
	}

	apiEndpoints := []string{
		fmt.Sprintf("https://doodapi.co/api/account/info?key=%s", url.QueryEscape(c.DoodStreamKey)),
		fmt.Sprintf("https://doodapi.com/api/account/info?key=%s", url.QueryEscape(c.DoodStreamKey)),
	}

	var lastErr error
	for _, endpoint := range apiEndpoints {
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "CINEMBROT/2.0 VideoBot (+https://cinembrot.web.id)")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		var res struct {
			Status int    `json:"status"`
			Msg    string `json:"msg"`
			Result struct {
				Email       string `json:"email"`
				Balance     string `json:"balance"`
				StorageUsed string `json:"storage_used"`
				StorageLeft string `json:"storage_left"`
			} `json:"result"`
		}

		if err := json.Unmarshal(body, &res); err != nil {
			lastErr = fmt.Errorf("gagal parse respons DoodStream: %w", err)
			continue
		}

		if res.Status != 200 && res.Status != 0 && res.Msg != "OK" {
			return &AccountInfo{
				Provider: "DoodStream",
				Status:   "ERROR",
				Message:  fmt.Sprintf("DoodStream Error: %s", res.Msg),
			}, nil
		}

		return &AccountInfo{
			Provider:    "DoodStream",
			Email:       res.Result.Email,
			Balance:     "$" + res.Result.Balance,
			StorageUsed: res.Result.StorageUsed,
			StorageLeft: res.Result.StorageLeft,
			Status:      "OK",
			Message:     "Akun aktif & siap melayani remote upload streaming.",
		}, nil
	}

	return &AccountInfo{
		Provider: "DoodStream",
		Status:   "ERROR",
		Message:  fmt.Sprintf("Gagal menghubungi server DoodStream: %v", lastErr),
	}, lastErr
}

// CheckStreamtape verifies credentials and checks balance on Streamtape
func (c *Client) CheckStreamtape() (*AccountInfo, error) {
	if c.StreamtapeLogin == "" || c.StreamtapeKey == "" {
		return &AccountInfo{
			Provider: "Streamtape",
			Status:   "UNCONFIGURED",
			Message:  "Username atau API Key Streamtape belum diisi lengkap di Pengaturan CMS.",
		}, nil
	}

	apiEndpoints := []string{
		fmt.Sprintf("https://api.streamtape.com/account/info?login=%s&key=%s", url.QueryEscape(c.StreamtapeLogin), url.QueryEscape(c.StreamtapeKey)),
		fmt.Sprintf("https://api.streamta.pe/account/info?login=%s&key=%s", url.QueryEscape(c.StreamtapeLogin), url.QueryEscape(c.StreamtapeKey)),
	}

	var lastErr error
	for _, endpoint := range apiEndpoints {
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "CINEMBROT/2.0 VideoBot (+https://cinembrot.web.id)")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		var res struct {
			Status int    `json:"status"`
			Msg    string `json:"msg"`
			Result struct {
				Login   string      `json:"login"`
				Email   string      `json:"email"`
				Balance interface{} `json:"balance"`
			} `json:"result"`
		}

		if err := json.Unmarshal(body, &res); err != nil {
			lastErr = fmt.Errorf("gagal parse respons Streamtape: %w (respons: %s)", err, string(body))
			continue
		}

		if res.Status != 200 {
			return &AccountInfo{
				Provider: "Streamtape",
				Status:   "ERROR",
				Message:  fmt.Sprintf("Streamtape Error: %s", res.Msg),
			}, nil
		}

		balanceStr := fmt.Sprintf("%v", res.Result.Balance)
		if !strings.HasPrefix(balanceStr, "$") {
			balanceStr = "$" + balanceStr
		}

		return &AccountInfo{
			Provider: "Streamtape",
			Email:    res.Result.Email,
			Balance:  balanceStr,
			Status:   "OK",
			Message:  "Akun Streamtape terhubung & siap digunakan.",
		}, nil
	}

	return &AccountInfo{
		Provider: "Streamtape",
		Status:   "ERROR",
		Message:  fmt.Sprintf("Gagal menghubungi API Streamtape: %v", lastErr),
	}, lastErr
}

// UploadDoodStream triggers server-to-server remote upload on DoodStream
// Returns fileCode and iframe EmbedURL: https://dood.so/e/{fileCode}
func (c *Client) UploadDoodStream(videoURL, title string) (string, string, error) {
	if c.DoodStreamKey == "" {
		return "", "", fmt.Errorf("DoodStream API Key belum diatur")
	}

	directURL := ConvertToDirectStreamableURL(videoURL)
	if directURL == "" || !IsDirectPlayableVideoURL(directURL) {
		return "", "", fmt.Errorf("URL bukan file video yang valid untuk remote upload")
	}

	cleanTitle := strings.TrimSpace(title)
	if cleanTitle == "" {
		cleanTitle = "Cinembrot Movie"
	}

	apiEndpoints := []string{
		fmt.Sprintf("https://doodapi.co/api/upload/url?key=%s&url=%s&new_title=%s",
			url.QueryEscape(c.DoodStreamKey), url.QueryEscape(directURL), url.QueryEscape(cleanTitle)),
		fmt.Sprintf("https://doodapi.com/api/upload/url?key=%s&url=%s&new_title=%s",
			url.QueryEscape(c.DoodStreamKey), url.QueryEscape(directURL), url.QueryEscape(cleanTitle)),
	}

	var lastErr error
	for _, endpoint := range apiEndpoints {
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "CINEMBROT/2.0 RemoteUploader (+https://cinembrot.web.id)")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		var res struct {
			Status int    `json:"status"`
			Msg    string `json:"msg"`
			Result struct {
				FileCode string `json:"filecode"`
			} `json:"result"`
		}

		if err := json.Unmarshal(body, &res); err != nil {
			lastErr = fmt.Errorf("gagal decode json DoodStream: %w", err)
			continue
		}

		if res.Status != 200 && res.Result.FileCode == "" {
			lastErr = fmt.Errorf("doodstream api error: %s", res.Msg)
			continue
		}

		fileCode := strings.TrimSpace(res.Result.FileCode)
		if fileCode != "" {
			embedURL := fmt.Sprintf("https://dood.so/e/%s", fileCode)
			return fileCode, embedURL, nil
		}
	}

	return "", "", lastErr
}

// UploadStreamtape triggers remote URL download task on Streamtape
// Returns taskID from Streamtape
func (c *Client) UploadStreamtape(videoURL, title string) (string, error) {
	if c.StreamtapeLogin == "" || c.StreamtapeKey == "" {
		return "", fmt.Errorf("Streamtape Login atau API Key belum diatur di Pengaturan CMS")
	}

	directURL := ConvertToDirectStreamableURL(videoURL)
	if directURL == "" || !IsDirectPlayableVideoURL(directURL) {
		return "", fmt.Errorf("URL bukan file video yang valid untuk remote upload")
	}

	cleanTitle := strings.TrimSpace(title)
	if cleanTitle == "" {
		cleanTitle = "Cinembrot Movie"
	}
	if !strings.HasSuffix(strings.ToLower(cleanTitle), ".mp4") && !strings.HasSuffix(strings.ToLower(cleanTitle), ".mkv") {
		cleanTitle += ".mp4"
	}

	endpoint := fmt.Sprintf("https://api.streamtape.com/remotedl/add?login=%s&key=%s&url=%s&name=%s",
		url.QueryEscape(c.StreamtapeLogin), url.QueryEscape(c.StreamtapeKey), url.QueryEscape(directURL), url.QueryEscape(cleanTitle))

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "CINEMBROT/2.0 RemoteUploader (+https://cinembrot.web.id)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var res struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
		Result struct {
			ID string `json:"id"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return "", fmt.Errorf("gagal unmarshal Streamtape remoteDL: %w (respons: %s)", err, string(body))
	}

	if res.Status != 200 || res.Result.ID == "" {
		return "", fmt.Errorf("streamtape remoteDL error: %s", res.Msg)
	}

	return res.Result.ID, nil
}

// CheckStreamtapeRemoteDL checks status of a remote upload task on Streamtape
func (c *Client) CheckStreamtapeRemoteDL(taskID string) (status string, extid string, err error) {
	if c.StreamtapeLogin == "" || c.StreamtapeKey == "" || strings.TrimSpace(taskID) == "" {
		return "", "", fmt.Errorf("parameter tidak lengkap")
	}

	endpoint := fmt.Sprintf("https://api.streamtape.com/remotedl/status?login=%s&key=%s&id=%s",
		url.QueryEscape(c.StreamtapeLogin), url.QueryEscape(c.StreamtapeKey), url.QueryEscape(taskID))

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "CINEMBROT/2.0 RemoteUploader (+https://cinembrot.web.id)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var res struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
		Result map[string]struct {
			ID     string      `json:"id"`
			Status string      `json:"status"`
			ExtID  interface{} `json:"extid"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return "", "", err
	}

	if item, ok := res.Result[taskID]; ok {
		extidStr := ""
		if item.ExtID != nil {
			if s, ok := item.ExtID.(string); ok {
				extidStr = s
			} else {
				extidStr = fmt.Sprintf("%v", item.ExtID)
			}
		}
		return item.Status, extidStr, nil
	}

	return "", "", fmt.Errorf("task ID tidak ditemukan di respons status")
}

// IsDirectPlayableVideoURL strictly validates that the URL is an actual video file and NOT a subtitle/search HTML webpage
func IsDirectPlayableVideoURL(rawURL string) bool {
	u := strings.ToLower(strings.TrimSpace(rawURL))
	if u == "" {
		return false
	}

	cleanPath := strings.Split(u, "?")[0]

	// 1. Blacklist mutlak: Subtitle, Web Search, Torrent, Arsip ZIP/RAR/7z
	if strings.Contains(u, "subdl.com") ||
		strings.Contains(u, "opensubtitles.org") ||
		strings.Contains(u, "subsource.net") ||
		strings.Contains(u, "subscene") ||
		strings.Contains(u, "/search/") ||
		strings.Contains(u, "/subtitle/") ||
		strings.HasSuffix(cleanPath, ".srt") ||
		strings.HasSuffix(cleanPath, ".vtt") ||
		strings.HasSuffix(cleanPath, ".ass") ||
		strings.HasSuffix(cleanPath, ".zip") ||
		strings.HasSuffix(cleanPath, ".rar") ||
		strings.HasSuffix(cleanPath, ".7z") ||
		strings.HasSuffix(cleanPath, ".tar") ||
		strings.HasSuffix(cleanPath, ".gz") ||
		strings.HasPrefix(u, "magnet:") ||
		strings.HasSuffix(cleanPath, ".torrent") {
		return false
	}

	// 2. Direct Video Cloud Storage
	if strings.Contains(u, "drive.google.com/file/d/") ||
		strings.Contains(u, "drive.google.com/uc?") ||
		strings.Contains(u, "pixeldrain.com/u/") ||
		strings.Contains(u, "pixeldrain.com/api/file/") ||
		strings.Contains(u, "archive.org/download/") ||
		strings.Contains(u, "acefile.co") ||
		strings.Contains(u, "buzzheavier.com") ||
		strings.Contains(u, "fastdrive") ||
		strings.Contains(u, "mediafire.com") {
		return true
	}

	// 3. Direct Video File Extensions (.mp4, .mkv, .webm, .avi, .ts, .m4v, .flv)
	if strings.HasSuffix(cleanPath, ".mp4") ||
		strings.HasSuffix(cleanPath, ".mkv") ||
		strings.HasSuffix(cleanPath, ".webm") ||
		strings.HasSuffix(cleanPath, ".avi") ||
		strings.HasSuffix(cleanPath, ".ts") ||
		strings.HasSuffix(cleanPath, ".m4v") ||
		strings.HasSuffix(cleanPath, ".flv") {
		return true
	}

	return false
}

// ConvertToDirectStreamableURL converts shared storage links (GDrive, Pixeldrain) into direct raw video streams
func ConvertToDirectStreamableURL(rawURL string) string {
	u := strings.TrimSpace(rawURL)
	if u == "" {
		return ""
	}

	// Jika ini tautan subtitle / non-video, langsung tolak!
	if !IsDirectPlayableVideoURL(u) {
		return ""
	}

	// 1. Google Drive Sharing Link -> Direct Download URL
	// https://drive.google.com/file/d/1A2B3C.../view -> https://drive.google.com/uc?export=download&id=1A2B3C...
	reGDrive := regexp.MustCompile(`drive\.google\.com/(?:file/d/|open\?id=|uc\?id=)([a-zA-Z0-9_-]+)`)
	if matches := reGDrive.FindStringSubmatch(u); len(matches) > 1 {
		return fmt.Sprintf("https://drive.google.com/uc?export=download&id=%s", matches[1])
	}

	// 2. Pixeldrain Sharing Link -> Direct File Stream API
	// https://pixeldrain.com/u/abc12345 -> https://pixeldrain.com/api/file/abc12345
	rePixeldrain := regexp.MustCompile(`pixeldrain\.com/u/([a-zA-Z0-9_-]+)`)
	if matches := rePixeldrain.FindStringSubmatch(u); len(matches) > 1 {
		return fmt.Sprintf("https://pixeldrain.com/api/file/%s", matches[1])
	}

	// 3. Raw Video File (.mp4, .mkv, .webm, archive.org, acefile direct, mediafire direct)
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u
	}

	return ""
}

// ProcessMovieRemoteUpload scans a movie's existing download links, triggers DoodStream & Streamtape remote upload,
// and saves the resulting playable embed URLs into the stream_links database table.
func (c *Client) ProcessMovieRemoteUpload(movie *model.Movie, db *gorm.DB) ([]model.StreamLink, error) {
	if movie == nil || db == nil {
		return nil, fmt.Errorf("parameter film atau database nil")
	}

	// 1. Saring HANYA link download yang benar-benar file video (bukan subtitle atau torrent)
	var validVideoLinks []model.DownloadLink
	for _, dl := range movie.DownloadLinks {
		if IsDirectPlayableVideoURL(dl.URL) {
			validVideoLinks = append(validVideoLinks, dl)
		}
	}

	// 2. Jika belum ada video di movie.DownloadLinks, periksa tabel database
	if len(validVideoLinks) == 0 {
		var dbLinks []model.DownloadLink
		db.Where("movie_id = ? AND url NOT LIKE 'magnet:%' AND url NOT LIKE '%.torrent%'", movie.ID).Find(&dbLinks)
		for _, dl := range dbLinks {
			if IsDirectPlayableVideoURL(dl.URL) {
				validVideoLinks = append(validVideoLinks, dl)
			}
		}
	}

	// 3. Jika film belum memiliki direct video sama sekali, cari otomatis menggunakan bot agregator (Pahe, Otakudesu, Samehadaku, Dramaid)
	if len(validVideoLinks) == 0 && c.aggCli != nil {
		log.Printf("[HOSTER] 🔍 Mencari link video langsung dari agregator untuk '%s'...\n", movie.Title)
		freshLinks := c.aggCli.ScrapeDirectDownloads(movie)
		for _, fl := range freshLinks {
			if IsDirectPlayableVideoURL(fl.URL) {
				fl.MovieID = movie.ID
				db.Create(&fl)
				validVideoLinks = append(validVideoLinks, fl)
			}
		}
	}

	// Jika tetap tidak ada link video asli, lewati dengan aman (JANGAN upload subtitle!)
	if len(validVideoLinks) == 0 {
		return nil, nil
	}

	// Periksa embed stream yang sudah ada agar tidak duplikat
	var existingStreams []model.StreamLink
	db.Where("movie_id = ?", movie.ID).Find(&existingStreams)

	hasDood := false
	hasStreamtape := false
	for _, s := range existingStreams {
		p := strings.ToLower(s.Provider)
		if strings.Contains(p, "dood") {
			hasDood = true
		}
		if strings.Contains(p, "streamtape") {
			hasStreamtape = true
		}
	}

	var newStreams []model.StreamLink

	// Pilih target URL video terbaik
	var targetURL string
	for _, vl := range validVideoLinks {
		direct := ConvertToDirectStreamableURL(vl.URL)
		if direct != "" {
			targetURL = direct
			break
		}
	}

	if targetURL == "" {
		return nil, nil
	}

	titleWithYear := fmt.Sprintf("%s (%d)", movie.Title, movie.Year)

	// 1. Remote Upload ke DoodStream
	if !hasDood && c.DoodStreamKey != "" {
		fileCode, embedURL, err := c.UploadDoodStream(targetURL, titleWithYear)
		if err == nil && fileCode != "" {
			doodStream := model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "DoodStream",
				ServerName: "Server VIP 1 (DoodStream)",
				Quality:    "HD 1080p",
				EmbedURL:   embedURL,
				DirectURL:  targetURL,
				IsValid:    true,
				Status:     "ACTIVE",
			}
			if err := db.Create(&doodStream).Error; err == nil {
				newStreams = append(newStreams, doodStream)
				log.Printf("[HOSTER] ✅ Film '%s' (ID %d) berhasil di-remote upload ke DoodStream: %s\n",
					movie.Title, movie.ID, embedURL)
			}
		} else if err != nil {
			log.Printf("[HOSTER] ⚠️ Gagal remote upload DoodStream untuk '%s': %v\n", movie.Title, err)
			if strings.Contains(strings.ToLower(err.Error()), "max urls limit") {
				return newStreams, fmt.Errorf("SLOT_LIMIT: antrean remote upload DoodStream mencapai batas maksimal (100 URLs): %w", err)
			}
		}
	}

	// 2. Remote Upload ke Streamtape
	if !hasStreamtape && c.StreamtapeLogin != "" && c.StreamtapeKey != "" {
		taskID, err := c.UploadStreamtape(targetURL, titleWithYear)
		if err == nil && taskID != "" {
			log.Printf("[HOSTER] ⏳ Streamtape RemoteDL task dibuat untuk '%s' (Task ID: %s)\n", movie.Title, taskID)
		} else if err != nil {
			log.Printf("[HOSTER] ⚠️ Gagal request remote upload Streamtape untuk '%s': %v\n", movie.Title, err)
		}
	}

	return newStreams, nil
}
