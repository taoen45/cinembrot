package safelinku

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	DefaultAPIEndpoint = "https://safelinku.com/api/v1/links"
	DefaultAPIToken    = "0e88844335a361150b23252773b50a0b03ded280"
)

var (
	// In-memory cache untuk URL yang telah dipendekkan agar tidak melebihi rate limit 60 req/min
	cacheMutex sync.RWMutex
	urlCache   = make(map[string]string)
	httpClient = &http.Client{
		Timeout: 6 * time.Second,
	}
)

type createLinkRequest struct {
	URL string `json:"url"`
}

type createLinkResponse struct {
	URL     string `json:"url"`
	Message string `json:"message,omitempty"`
}

// ShortenURL memendekkan targetURL menggunakan SafelinkU REST API resmi.
// Jika API gagal atau timeout, fungsi mengembalikan original URL dengan error (safe fallback).
func ShortenURL(targetURL string, apiToken string) (string, error) {
	targetURL = strings.TrimSpace(targetURL)
	if targetURL == "" {
		return "", errors.New("target URL cannot be empty")
	}

	// Jangan shorten jika sudah merupakan link SafelinkU atau sfl.gl/sfl.ink
	lowerURL := strings.ToLower(targetURL)
	if strings.Contains(lowerURL, "safelinku.com") || strings.Contains(lowerURL, "sfl.gl") || strings.Contains(lowerURL, "sfl.ink") {
		return targetURL, nil
	}

	if apiToken == "" {
		apiToken = DefaultAPIToken
	}

	// 1. Cek cache in-memory
	cacheMutex.RLock()
	if cached, ok := urlCache[targetURL]; ok && cached != "" {
		cacheMutex.RUnlock()
		return cached, nil
	}
	cacheMutex.RUnlock()

	// 2. Kirim request POST ke SafelinkU REST API
	reqPayload := createLinkRequest{
		URL: targetURL,
	}
	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return targetURL, fmt.Errorf("marshal payload error: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, DefaultAPIEndpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return targetURL, fmt.Errorf("create request error: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CINEMBROT/2.0 (Linux; x86_64)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return targetURL, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return targetURL, fmt.Errorf("read response body error: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return targetURL, fmt.Errorf("safelinku API returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var resData createLinkResponse
	if err := json.Unmarshal(respBody, &resData); err != nil {
		return targetURL, fmt.Errorf("unmarshal response error: %w", err)
	}

	if resData.URL == "" {
		return targetURL, errors.New("safelinku API returned empty shortened URL")
	}

	shortened := resData.URL

	// 3. Simpan ke in-memory cache
	cacheMutex.Lock()
	urlCache[targetURL] = shortened
	cacheMutex.Unlock()

	return shortened, nil
}
