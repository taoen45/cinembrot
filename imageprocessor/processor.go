package imageprocessor

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cinembrot/model"

	"github.com/disintegration/imaging"
	"github.com/nickalie/go-webpbin"
	_ "golang.org/x/image/webp"
)

var httpClient = &http.Client{
	Timeout: 25 * time.Second,
}

// EnsureUploadDirs creates upload directories if they don't exist
func EnsureUploadDirs() {
	dirs := []string{
		filepath.Join("public", "uploads", "posters"),
		filepath.Join("public", "uploads", "backdrops"),
		filepath.Join("public", "uploads", "actors"),
	}
	for _, dir := range dirs {
		_ = os.MkdirAll(dir, 0755)
	}
}

// ProcessMovieImages downloads movie poster & backdrop, converts to WebP (Original + Thumbnail), and updates model paths
func ProcessMovieImages(movie *model.Movie, userAgent string) {
	EnsureUploadDirs()

	if movie.Slug == "" {
		movie.Slug = fmt.Sprintf("movie-%d", time.Now().UnixNano())
	}

	// 1. Process Poster (Original WebP max 800px & Thumb WebP 300px)
	if movie.PosterURL != "" && !strings.HasPrefix(movie.PosterURL, "/uploads/") {
		origPath, thumbPath, err := DownloadAndConvertToWebP(
			movie.PosterURL,
			"posters",
			"poster_"+movie.Slug,
			800, // Max orig width (sangat tajam & hemat storage)
			300, // Thumb width (optimal untuk card katalog)
			userAgent,
		)
		if err == nil {
			movie.PosterURL = origPath
			movie.PosterThumbURL = thumbPath
			if movie.ThumbnailURL == "" {
				movie.ThumbnailURL = thumbPath
			}
		} else {
			log.Printf("[WARN] Failed to process poster for '%s': %v\n", movie.Title, err)
		}
	}

	// 2. Process Backdrop (Original WebP max 1280px & Thumb WebP 500px)
	if movie.BackdropURL != "" && !strings.HasPrefix(movie.BackdropURL, "/uploads/") {
		origPath, thumbPath, err := DownloadAndConvertToWebP(
			movie.BackdropURL,
			"backdrops",
			"backdrop_"+movie.Slug,
			1280, // Max width banner (efisien & tajam)
			500,  // Thumb width
			userAgent,
		)
		if err == nil {
			movie.BackdropURL = origPath
			movie.BackdropThumbURL = thumbPath
		} else {
			log.Printf("[WARN] Failed to process backdrop for '%s': %v\n", movie.Title, err)
		}
	}

	// 3. Process Actor Photos (Thumbnails 160px)
	for i := range movie.Actors {
		actor := &movie.Actors[i]
		if actor.PhotoURL != "" && !strings.HasPrefix(actor.PhotoURL, "/uploads/") {
			actorSlug := actor.Slug
			if actorSlug == "" {
				actorSlug = fmt.Sprintf("actor-%x", md5.Sum([]byte(actor.Name)))
			}
			origPath, thumbPath, err := DownloadAndConvertToWebP(
				actor.PhotoURL,
				"actors",
				"actor_"+actorSlug,
				350,
				160,
				userAgent,
			)
			if err == nil {
				actor.PhotoURL = origPath
				actor.PhotoThumbURL = thumbPath
			}
		}
	}
}

// DownloadAndConvertToWebP downloads an image, resizes it, and saves as WebP (original & thumbnail)
func DownloadAndConvertToWebP(
	imageURL string,
	folderType string,
	filenameBase string,
	maxOrigWidth int,
	thumbWidth int,
	userAgent string,
) (origWebURL string, thumbWebURL string, err error) {
	if strings.TrimSpace(imageURL) == "" {
		return "", "", nil
	}

	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return "", "", err
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("http download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read image body: %w", err)
	}

	// Decode source image
	img, _, err := image.Decode(bytes.NewReader(bodyBytes))
	if err != nil {
		return "", "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Target file paths
	targetDir := filepath.Join("public", "uploads", folderType)
	_ = os.MkdirAll(targetDir, 0755)

	origFilename := fmt.Sprintf("%s.webp", filenameBase)
	thumbFilename := fmt.Sprintf("thumb_%s.webp", filenameBase)

	origFilePath := filepath.Join(targetDir, origFilename)
	thumbFilePath := filepath.Join(targetDir, thumbFilename)

	// 1. Process & Save Original (constrained to maxOrigWidth if larger)
	var origImg image.Image = img
	bounds := img.Bounds()
	if bounds.Dx() > maxOrigWidth && maxOrigWidth > 0 {
		origImg = imaging.Resize(img, maxOrigWidth, 0, imaging.Lanczos)
	}

	origFile, err := os.Create(origFilePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create original file: %w", err)
	}
	defer origFile.Close()

	// Encode to WebP with 80% quality using webpbin (standar emas WebP: hemat s/d 85%)
	err = webpbin.NewCWebP().
		Quality(80).
		InputImage(origImg).
		Output(origFile).
		Run()

	if err != nil {
		// Fallback to JPEG if cwebp binary is unavailable
		_ = jpeg.Encode(origFile, origImg, &jpeg.Options{Quality: 80})
	}

	// 2. Process & Save Thumbnail
	var thumbImg image.Image = img
	if thumbWidth > 0 && bounds.Dx() > thumbWidth {
		thumbImg = imaging.Resize(img, thumbWidth, 0, imaging.Lanczos)
	}

	thumbFile, err := os.Create(thumbFilePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create thumb file: %w", err)
	}
	defer thumbFile.Close()

	// Encode to WebP with 75% quality (sangat ringan, ukuran ~10-18KB per thumbnail)
	err = webpbin.NewCWebP().
		Quality(75).
		InputImage(thumbImg).
		Output(thumbFile).
		Run()

	if err != nil {
		// Fallback to JPEG if cwebp binary is unavailable
		_ = jpeg.Encode(thumbFile, thumbImg, &jpeg.Options{Quality: 75})
	}

	origWebURL = fmt.Sprintf("/uploads/%s/%s", folderType, origFilename)
	thumbWebURL = fmt.Sprintf("/uploads/%s/%s", folderType, thumbFilename)

	return origWebURL, thumbWebURL, nil
}
