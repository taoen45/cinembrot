package server

import (
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ==================== 1. GLOBAL SECURITY HEADERS ====================

// SecurityHeadersMiddleware injects industry-standard HTTP security headers
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mencegah Clickjacking (embedding via iframe tersembunyi)
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")

		// Mencegah MIME-sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Perlindungan XSS bawaan browser
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer policy yang aman (melindungi URL parameter referer privat)
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Matikan akses fitur hardware sensitif yang tidak digunakan website streaming
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

		// HSTS (Strict-Transport-Security) jika diakses via HTTPS (Cloudflare)
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}

// ==================== 2. IN-MEMORY RATE LIMITER ====================

type clientBucket struct {
	tokens     float64
	lastRefill time.Time
}

// IPRateLimiter provides lightweight thread-safe token bucket rate limiting per IP address
type IPRateLimiter struct {
	mu          sync.Mutex
	clients     map[string]*clientBucket
	capacity    float64
	refillRate  float64 // tokens per second
	lastCleaned time.Time
}

// NewIPRateLimiter creates a new rate limiter (maxBurst = kapasitas maksimal, refillPerMinute = regenerasi per menit)
func NewIPRateLimiter(maxBurst float64, refillPerMinute float64) *IPRateLimiter {
	limiter := &IPRateLimiter{
		clients:     make(map[string]*clientBucket),
		capacity:    maxBurst,
		refillRate:  refillPerMinute / 60.0,
		lastCleaned: time.Now(),
	}

	// Background routine untuk membersihkan IP inactive setiap 5 menit agar hemat memori
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			limiter.cleanup(15 * time.Minute)
		}
	}()

	return limiter
}

func (l *IPRateLimiter) cleanup(idleTimeout time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for ip, bucket := range l.clients {
		if now.Sub(bucket.lastRefill) > idleTimeout {
			delete(l.clients, ip)
		}
	}
}

// Allow checks if the given IP address has tokens available
func (l *IPRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	bucket, exists := l.clients[ip]
	if !exists {
		l.clients[ip] = &clientBucket{
			tokens:     l.capacity - 1,
			lastRefill: now,
		}
		return true
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * l.refillRate
	if bucket.tokens > l.capacity {
		bucket.tokens = l.capacity
	}
	bucket.lastRefill = now

	if bucket.tokens >= 1.0 {
		bucket.tokens -= 1.0
		return true
	}

	return false
}

// ExtractClientIP retrieves the real client IP, respecting Cloudflare (CF-Connecting-IP) and reverse proxies
func ExtractClientIP(r *http.Request) string {
	// 1. Cloudflare header
	if cfIP := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); cfIP != "" {
		return cfIP
	}

	// 2. X-Forwarded-For header
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}

	// 3. X-Real-IP header
	if xRealIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); xRealIP != "" {
		return xRealIP
	}

	// 4. RemoteAddr fallback
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

// Global Rate Limiters
var (
	// Public Rate Limiter: 100 requests burst, 60 requests per minute
	globalPublicLimiter = NewIPRateLimiter(100, 60)

	// Admin Auth Limiter (Login Brute Force Protection): 5 attempts burst, 5 per minute
	globalAdminAuthLimiter = NewIPRateLimiter(5, 5)

	// Action Limiter (Comments, Refresh Links API): 15 requests burst, 15 per minute
	globalActionLimiter = NewIPRateLimiter(15, 15)
)

// RateLimitPublicMiddleware protects public endpoints from aggressive scraper bots
func RateLimitPublicMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Lewati static assets (/img/, /favicon.png, /uploads/)
		path := r.URL.Path
		if strings.HasPrefix(path, "/img/") || strings.HasPrefix(path, "/uploads/") || strings.HasPrefix(path, "/downloads/") || path == "/favicon.ico" || path == "/favicon.png" {
			next.ServeHTTP(w, r)
			return
		}

		ip := ExtractClientIP(r)
		if !globalPublicLimiter.Allow(ip) {
			http.Error(w, "Terlalu banyak permintaan (Rate Limit Exceeded). Harap tunggu beberapa saat sebelum mencoba lagi.", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RateLimitAdminAuth checks login rate limit
func RateLimitAdminAuth(w http.ResponseWriter, r *http.Request) bool {
	ip := ExtractClientIP(r)
	if !globalAdminAuthLimiter.Allow(ip) {
		http.Error(w, "Terlalu banyak percobaan login gagal. Demi keamanan server, silakan tunggu 1 menit.", http.StatusTooManyRequests)
		return false
	}
	return true
}

// RateLimitAction checks action API rate limit
func RateLimitAction(w http.ResponseWriter, r *http.Request) bool {
	ip := ExtractClientIP(r)
	if !globalActionLimiter.Allow(ip) {
		http.Error(w, "Batas permintaan tercapai. Harap tunggu beberapa saat.", http.StatusTooManyRequests)
		return false
	}
	return true
}

// ==================== 3. URL PARAMETER SANITIZERS & VALIDATORS ====================

var slugRegex = regexp.MustCompile(`^[a-zA-Z0-9\-_]{1,120}$`)

// ValidateSlug verifies if a path slug is safe from path traversal and injection
func ValidateSlug(slug string) bool {
	clean := strings.TrimSpace(slug)
	if clean == "" || len(clean) > 120 {
		return false
	}
	return slugRegex.MatchString(clean)
}

// SanitizeSearchQuery strips null bytes, trims, and bounds max length to 100 characters
func SanitizeSearchQuery(q string) string {
	clean := strings.ReplaceAll(q, "\x00", "")
	clean = strings.TrimSpace(clean)
	if len(clean) > 100 {
		clean = clean[:100]
	}
	return clean
}

// SanitizePageNumber parses integer page and bounds between 1 and maxPages
func SanitizePageNumber(pStr string, maxPages int) int {
	if maxPages <= 0 {
		maxPages = 500
	}
	p, err := strconv.Atoi(strings.TrimSpace(pStr))
	if err != nil || p < 1 {
		return 1
	}
	if p > maxPages {
		return maxPages
	}
	return p
}

// SanitizePositiveID parses string ID into uint safely
func SanitizePositiveID(idStr string) (uint, bool) {
	id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint(id), true
}

// IsSafeLocalRedirect checks that redirect URL is strictly an internal local path
func IsSafeLocalRedirect(target string) bool {
	clean := strings.TrimSpace(target)
	if clean == "" {
		return false
	}
	// Harus diawali dengan "/" tunggal
	if !strings.HasPrefix(clean, "/") {
		return false
	}
	// Mencegah "//evil.com"
	if strings.HasPrefix(clean, "//") {
		return false
	}
	// Mencegah "http:", "https:", "javascript:", "data:"
	lower := strings.ToLower(clean)
	if strings.Contains(lower, "://") || strings.Contains(lower, "javascript:") || strings.Contains(lower, "data:") {
		return false
	}
	return true
}
