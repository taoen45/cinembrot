package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// TurnstileResponse represents the JSON response returned by Cloudflare siteverify API
type TurnstileResponse struct {
	Success     bool      `json:"success"`
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
	Action      string    `json:"action"`
	CData       string    `json:"cdata"`
}

// VerifyTurnstile verifies a Cloudflare Turnstile token with Cloudflare's siteverify API endpoint
func VerifyTurnstile(secretKey, token, remoteIP string) (bool, error) {
	if token == "" {
		return false, fmt.Errorf("token turnstile kosong")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	formData := url.Values{}
	formData.Set("secret", secretKey)
	formData.Set("response", token)
	if remoteIP != "" {
		formData.Set("remoteip", remoteIP)
	}

	resp, err := client.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", formData)
	if err != nil {
		return false, fmt.Errorf("gagal menghubungi cloudflare turnstile API: %w", err)
	}
	defer resp.Body.Close()

	var tr TurnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return false, fmt.Errorf("gagal mendecode respon turnstile: %w", err)
	}

	if !tr.Success {
		return false, fmt.Errorf("verifikasi ditolak oleh cloudflare: %v", tr.ErrorCodes)
	}

	return true, nil
}
