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

	// 2. From RawMetadata JSON: "id": 12345
	if movie.RawMetadata != "" {
		var meta struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal([]byte(movie.RawMetadata), &meta); err == nil && meta.ID > 0 {
			return meta.ID
		}
	}

	return 0
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
func GenerateMultiServerStreams(movie *model.Movie, tmdbID int, season int, episode int) []model.StreamLink {
	if movie == nil {
		return nil
	}

	if tmdbID <= 0 {
		tmdbID = ExtractTMDbID(movie)
	}

	isTV := movie.Type == "anime" || movie.Type == "drama_pendek" || movie.Type == "series"
	if season <= 0 {
		season = 1
	}
	if episode <= 0 {
		episode = 1
	}

	var servers []model.StreamLink

	if tmdbID > 0 {
		if isTV {
			// TV Show / Anime / Drama embeds
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "VidSrc HD",
				ServerName: "Server 1 (VidSrc)",
				Quality:    "HD 1080p",
				EmbedURL:   fmt.Sprintf("https://vidsrc.xyz/embed/tv/%d/%d/%d", tmdbID, season, episode),
				IsValid:    true,
				Status:     "ACTIVE",
			})
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "AutoEmbed Fast",
				ServerName: "Server 2 (AutoEmbed)",
				Quality:    "HD 1080p",
				EmbedURL:   fmt.Sprintf("https://player.autoembed.cc/embed/tv/%d/%d/%d", tmdbID, season, episode),
				IsValid:    true,
				Status:     "ACTIVE",
			})
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "2Embed VIP",
				ServerName: "Server 3 (2Embed)",
				Quality:    "HD 720p",
				EmbedURL:   fmt.Sprintf("https://www.2embed.cc/embedtv/%d&s=%d&e=%d", tmdbID, season, episode),
				IsValid:    true,
				Status:     "ACTIVE",
			})
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "VidLink Pro",
				ServerName: "Server 4 (VidLink)",
				Quality:    "HD 1080p",
				EmbedURL:   fmt.Sprintf("https://vidlink.pro/tv/%d/%d/%d", tmdbID, season, episode),
				IsValid:    true,
				Status:     "ACTIVE",
			})
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "SuperEmbed",
				ServerName: "Server 5 (Multi)",
				Quality:    "HD 720p",
				EmbedURL:   fmt.Sprintf("https://multiembed.mov/?video_id=%d&tmdb=1&s=%d&e=%d", tmdbID, season, episode),
				IsValid:    true,
				Status:     "ACTIVE",
			})
		} else {
			// Regular Movie embeds
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "VidSrc HD",
				ServerName: "Server 1 (VidSrc)",
				Quality:    "HD 1080p",
				EmbedURL:   fmt.Sprintf("https://vidsrc.xyz/embed/movie/%d", tmdbID),
				IsValid:    true,
				Status:     "ACTIVE",
			})
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "AutoEmbed Fast",
				ServerName: "Server 2 (AutoEmbed)",
				Quality:    "HD 1080p",
				EmbedURL:   fmt.Sprintf("https://player.autoembed.cc/embed/movie/%d", tmdbID),
				IsValid:    true,
				Status:     "ACTIVE",
			})
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "2Embed VIP",
				ServerName: "Server 3 (2Embed)",
				Quality:    "HD 720p",
				EmbedURL:   fmt.Sprintf("https://www.2embed.cc/embed/%d", tmdbID),
				IsValid:    true,
				Status:     "ACTIVE",
			})
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "VidLink Pro",
				ServerName: "Server 4 (VidLink)",
				Quality:    "HD 1080p",
				EmbedURL:   fmt.Sprintf("https://vidlink.pro/movie/%d", tmdbID),
				IsValid:    true,
				Status:     "ACTIVE",
			})
			servers = append(servers, model.StreamLink{
				MovieID:    movie.ID,
				Provider:   "SuperEmbed",
				ServerName: "Server 5 (Multi)",
				Quality:    "HD 720p",
				EmbedURL:   fmt.Sprintf("https://multiembed.mov/?video_id=%d&tmdb=1", tmdbID),
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
