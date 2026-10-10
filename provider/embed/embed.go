package embed

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"cinembrot/model"
)

// ExtractTMDbID attempts to extract the TMDb ID from SourceURL, RawMetadata, or alternative fields
func ExtractTMDbID(movie *model.Movie) int {
	if movie == nil {
		return 0
	}

	// 1. From SourceURL: https://www.themoviedb.org/tv/12345 or /movie/12345
	reTMDbURL := regexp.MustCompile(`themoviedb\.org/(?:tv|movie)/(\d+)`)
	if matches := reTMDbURL.FindStringSubmatch(movie.SourceURL); len(matches) > 1 {
		if id, err := strconv.Atoi(matches[1]); err == nil && id > 0 {
			return id
		}
	}

	// 2. From RawMetadata JSON: prioritas tmdb_id, hindari salah ambil MAL ID sebagai TMDb ID
	if movie.RawMetadata != "" {
		var meta struct {
			TMDbID int `json:"tmdb_id"`
			ID     int `json:"id"`
		}
		if err := json.Unmarshal([]byte(movie.RawMetadata), &meta); err == nil {
			if meta.TMDbID > 0 {
				return meta.TMDbID
			}
			// Hanya gunakan field "id" jika sumber asalnya memang dari TMDb
			sourceWebLower := strings.ToLower(movie.SourceWebsite)
			if meta.ID > 0 && (strings.Contains(sourceWebLower, "tmdb") || strings.Contains(movie.SourceURL, "themoviedb.org")) {
				return meta.ID
			}
		}
	}

	return 0
}

// ExtractIMDbID attempts to extract the IMDb ID (tt...) from SourceURL or RawMetadata
func ExtractIMDbID(movie *model.Movie) string {
	if movie == nil {
		return ""
	}

	// 1. From SourceURL: imdb.com/title/tt1234567
	reIMDb := regexp.MustCompile(`(tt\d{7,10})`)
	if matches := reIMDb.FindStringSubmatch(movie.SourceURL); len(matches) > 1 {
		return matches[1]
	}

	// 2. From RawMetadata JSON: "imdb_id": "tt1234567"
	if movie.RawMetadata != "" {
		var meta struct {
			IMDbID string `json:"imdb_id"`
			IMDb   string `json:"imdbId"`
		}
		if err := json.Unmarshal([]byte(movie.RawMetadata), &meta); err == nil {
			if meta.IMDbID != "" && strings.HasPrefix(meta.IMDbID, "tt") {
				return meta.IMDbID
			}
			if meta.IMDb != "" && strings.HasPrefix(meta.IMDb, "tt") {
				return meta.IMDb
			}
		}
	}

	return ""
}

// IsDeadStreamURL checks if a streaming embed URL points to a known dead or DNS-failed domain
func IsDeadStreamURL(rawURL string) bool {
	u := strings.ToLower(rawURL)
	return strings.Contains(u, "vidsrc.xyz") ||
		strings.Contains(u, "embed.su") ||
		strings.Contains(u, "moviesapi.club") ||
		strings.Contains(u, "streamembed.cc") ||
		strings.Contains(u, "vidsrc.pm")
}

// ExtractEpisodeCount determines the number of episodes for an anime or drama
func ExtractEpisodeCount(movie *model.Movie) int {
	if movie == nil {
		return 1
	}

	// 1. From RawMetadata if TMDb TV
	if movie.RawMetadata != "" {
		var meta struct {
			NumberOfEpisodes int `json:"number_of_episodes"`
		}
		if err := json.Unmarshal([]byte(movie.RawMetadata), &meta); err == nil && meta.NumberOfEpisodes > 0 {
			return meta.NumberOfEpisodes
		}
	}

	// 2. From Tagline: "Anime TV • 24 Episodes • Skor 8.5/10"
	reEps := regexp.MustCompile(`(\d+)\s+Episodes?`)
	if matches := reEps.FindStringSubmatch(movie.Tagline); len(matches) > 1 {
		if eps, err := strconv.Atoi(matches[1]); err == nil && eps > 0 {
			return eps
		}
	}

	// Default for ongoing / unspecified anime or drama
	if movie.Type == "anime" || movie.Type == "drama_pendek" || movie.Type == "series" {
		return 12
	}

	return 1
}

// GenerateMultiServerStreams creates a list of stream server links for a given movie/episode
// Uses fast, resilient, and active streaming providers (VidLink HD, SmashyStream, AutoEmbed, VidSrc CC, 2Embed, MultiEmbed, NontonGo)
func GenerateMultiServerStreams(movie *model.Movie, tmdbID int, season int, episode int) []model.StreamLink {
	if movie == nil {
		return nil
	}

	if tmdbID <= 0 {
		tmdbID = ExtractTMDbID(movie)
	}
	imdbID := ExtractIMDbID(movie)

	isTV := movie.Type == "anime" || movie.Type == "drama_pendek" || movie.Type == "series"
	isAnime := movie.Type == "anime"
	if season <= 0 {
		season = 1
	}
	if episode <= 0 {
		episode = 1
	}

	var servers []model.StreamLink

	// Tentukan ID video yang dapat diputar (prioritas TMDb ID, cadangan IMDb ID)
	var videoIDStr string
	if tmdbID > 0 {
		videoIDStr = strconv.Itoa(tmdbID)
	} else if imdbID != "" {
		videoIDStr = imdbID
	}

	if videoIDStr != "" {
		if isTV {
			// TV Show / Anime / Drama embeds
			// Server 1: VidLink HD (Next.js player, sangat cepat, HD 1080p, minim iklan)
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "VidLink HD",
				ServerName: "Server 1 (VidLink HD)",
				Quality:    "HD 1080p",
				EmbedURL:   fmt.Sprintf("https://vidlink.pro/tv/%s/%d/%d", videoIDStr, season, episode),
				IsValid:    true,
				Status:     "ACTIVE",
			})
			// Server 2: SmashyStream (Sangat stabil, multi-source video cadangan di dalam player)
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "SmashyStream",
				ServerName: "Server 2 (SmashyStream)",
				Quality:    "HD 1080p",
				EmbedURL:   fmt.Sprintf("https://embed.smashystream.com/playere.php?tmdb=%s&season=%d&episode=%d", videoIDStr, season, episode),
				IsValid:    true,
				Status:     "ACTIVE",
			})
			// Server 3: AutoEmbed Fast (player.autoembed.cc - Responsif, buffer cepat)
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "AutoEmbed Fast",
				ServerName: "Server 3 (AutoEmbed Fast)",
				Quality:    "HD 1080p",
				EmbedURL:   fmt.Sprintf("https://player.autoembed.cc/embed/tv/%s/%d/%d", videoIDStr, season, episode),
				IsValid:    true,
				Status:     "ACTIVE",
			})
			// Server 4: VidSrc CC v2 (Mendukung anime endpoint khusus atau TV)
			vidSrcTVURL := fmt.Sprintf("https://vidsrc.cc/v2/embed/tv/%s/%d/%d", videoIDStr, season, episode)
			if isAnime {
				vidSrcTVURL = fmt.Sprintf("https://vidsrc.cc/v2/embed/anime/%s/%d/%d", videoIDStr, season, episode)
			}
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "VidSrc CC",
				ServerName: "Server 4 (VidSrc CC)",
				Quality:    "HD 1080p",
				EmbedURL:   vidSrcTVURL,
				IsValid:    true,
				Status:     "ACTIVE",
			})
			// Server 5: 2Embed VIP (2embed.cc - format TV embed stabil)
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "2Embed VIP",
				ServerName: "Server 5 (2Embed VIP)",
				Quality:    "HD 720p",
				EmbedURL:   fmt.Sprintf("https://www.2embed.cc/embedtv/%s&s=%d&e=%d", videoIDStr, season, episode),
				IsValid:    true,
				Status:     "ACTIVE",
			})
			// Server 6: MultiEmbed (multiembed.mov - Multi provider)
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "MultiEmbed",
				ServerName: "Server 6 (MultiEmbed)",
				Quality:    "HD 720p",
				EmbedURL:   fmt.Sprintf("https://multiembed.mov/?video_id=%s&tmdb=1&s=%d&e=%d", videoIDStr, season, episode),
				IsValid:    true,
				Status:     "ACTIVE",
			})
			// Server 7: NontonGo Asia (Ramah ISP lokal Indonesia)
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "NontonGo Asia",
				ServerName: "Server 7 (NontonGo Asia)",
				Quality:    "HD 720p",
				EmbedURL:   fmt.Sprintf("https://www.nontongo.win/embed/tv/%s/%d/%d", videoIDStr, season, episode),
				IsValid:    true,
				Status:     "ACTIVE",
			})
		} else {
			// Regular Movie embeds
			// Server 1: VidLink HD (Utama - Super Cepat, HD 1080p)
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "VidLink HD",
				ServerName: "Server 1 (VidLink HD)",
				Quality:    "HD 1080p",
				EmbedURL:   fmt.Sprintf("https://vidlink.pro/movie/%s", videoIDStr),
				IsValid:    true,
				Status:     "ACTIVE",
			})
			// Server 2: SmashyStream (Sangat stabil & banyak source)
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "SmashyStream",
				ServerName: "Server 2 (SmashyStream)",
				Quality:    "HD 1080p",
				EmbedURL:   fmt.Sprintf("https://embed.smashystream.com/playere.php?tmdb=%s", videoIDStr),
				IsValid:    true,
				Status:     "ACTIVE",
			})
			// Server 3: AutoEmbed Fast (player.autoembed.cc)
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "AutoEmbed Fast",
				ServerName: "Server 3 (AutoEmbed Fast)",
				Quality:    "HD 1080p",
				EmbedURL:   fmt.Sprintf("https://player.autoembed.cc/embed/movie/%s", videoIDStr),
				IsValid:    true,
				Status:     "ACTIVE",
			})
			// Server 4: VidSrc CC v2 (vidsrc.cc)
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "VidSrc CC",
				ServerName: "Server 4 (VidSrc CC)",
				Quality:    "HD 1080p",
				EmbedURL:   fmt.Sprintf("https://vidsrc.cc/v2/embed/movie/%s", videoIDStr),
				IsValid:    true,
				Status:     "ACTIVE",
			})
			// Server 5: 2Embed VIP (2embed.cc)
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "2Embed VIP",
				ServerName: "Server 5 (2Embed VIP)",
				Quality:    "HD 720p",
				EmbedURL:   fmt.Sprintf("https://www.2embed.cc/embed/%s", videoIDStr),
				IsValid:    true,
				Status:     "ACTIVE",
			})
			// Server 6: MultiEmbed (multiembed.mov)
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "MultiEmbed",
				ServerName: "Server 6 (MultiEmbed)",
				Quality:    "HD 720p",
				EmbedURL:   fmt.Sprintf("https://multiembed.mov/?video_id=%s&tmdb=1", videoIDStr),
				IsValid:    true,
				Status:     "ACTIVE",
			})
			// Server 7: NontonGo Asia (Ramah ISP lokal Indonesia)
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "NontonGo Asia",
				ServerName: "Server 7 (NontonGo Asia)",
				Quality:    "HD 720p",
				EmbedURL:   fmt.Sprintf("https://www.nontongo.win/embed/movie/%s", videoIDStr),
				IsValid:    true,
				Status:     "ACTIVE",
			})
		}
	}

	// If trailer exists, provide official YouTube as fallback server
	if movie.TrailerURL != "" {
		youtubeEmbedURL := movie.TrailerURL
		if strings.Contains(movie.TrailerURL, "watch?v=") {
			youtubeEmbedURL = strings.Replace(movie.TrailerURL, "watch?v=", "embed/", 1)
		}
		servers = append(servers, model.StreamLink{
			MovieID:    movie.ID,
			Provider:   "YouTube Trailer",
			ServerName: "Trailer Resmi",
			Quality:    "HD",
			EmbedURL:   youtubeEmbedURL,
			IsValid:    true,
			Status:     "ACTIVE",
		})
	}

	return servers
}
