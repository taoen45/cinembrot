package server

import (
	"strconv"
	"strings"
	"sync"

	"cinembrot/model"
	"cinembrot/provider/vorta"
)

// Platform yang memakai ID chapter unik (bukan nomor episode) pada endpoint /api/v2/video.
// ID episode berikutnya hanya diketahui dari field next_video_id saat episode sebelumnya
// ditonton, sehingga dipelajari secara organik (tanpa memindai/brute-force API Vorta).
var chapterIDPlatforms = map[string]bool{"meloshort": true}

// Platform dengan hardsub permanen (teks bahasa tertanam di video), tidak punya pilihan subtitle.
var hardsubPlatforms = map[string]bool{"reelshort": true, "microdrama": true, "stardusttv": true}

var dramaChapterCache = struct {
	sync.Mutex
	m map[string]map[int]string
}{m: make(map[string]map[int]string)}

func chapterCacheKey(platform, rawID string) string {
	return strings.ToLower(platform) + ":" + rawID
}

func rememberChapterID(platform, rawID string, ep int, chapterID string) {
	if ep < 1 || chapterID == "" {
		return
	}
	key := chapterCacheKey(platform, rawID)
	dramaChapterCache.Lock()
	defer dramaChapterCache.Unlock()
	if dramaChapterCache.m[key] == nil {
		dramaChapterCache.m[key] = make(map[int]string)
	}
	dramaChapterCache.m[key][ep] = chapterID
}

func lookupChapterID(platform, rawID string, ep int) (string, bool) {
	key := chapterCacheKey(platform, rawID)
	dramaChapterCache.Lock()
	defer dramaChapterCache.Unlock()
	id, ok := dramaChapterCache.m[key][ep]
	return id, ok
}

// highestKnownEpisode mengembalikan nomor episode tertinggi yang ID-nya sudah dipelajari.
func highestKnownEpisode(platform, rawID string) int {
	key := chapterCacheKey(platform, rawID)
	dramaChapterCache.Lock()
	defer dramaChapterCache.Unlock()
	max := 0
	for ep := range dramaChapterCache.m[key] {
		if ep > max {
			max = ep
		}
	}
	return max
}

// resolveChapterID menentukan nilai chapterId untuk request video.
// Platform biasa memakai nomor episode. Platform ber-chapter-ID memakai cache organik;
// episode 1 diambil dari RawMetadata (bagian ke-4) atau dari 1x panggilan detail (lalu di-cache).
func resolveChapterID(client *vorta.Client, movie *model.Movie, platform, rawID string, ep int) (string, bool) {
	if !chapterIDPlatforms[strings.ToLower(platform)] {
		return strconv.Itoa(ep), true
	}
	if id, ok := lookupChapterID(platform, rawID, ep); ok {
		return id, true
	}
	if ep != 1 {
		return "", false
	}
	if movie != nil && strings.HasPrefix(movie.RawMetadata, "vorta:") {
		parts := strings.Split(movie.RawMetadata, ":")
		if len(parts) >= 4 && parts[3] != "" {
			rememberChapterID(platform, rawID, 1, parts[3])
			return parts[3], true
		}
	}
	detail, err := client.GetDetail(platform, rawID, "id")
	if err != nil || detail == nil || len(detail.Chapters) == 0 || detail.Chapters[0].ID == "" {
		return "", false
	}
	rememberChapterID(platform, rawID, 1, detail.Chapters[0].ID)
	return detail.Chapters[0].ID, true
}

