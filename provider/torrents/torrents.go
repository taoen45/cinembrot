package torrents

import (
	"fmt"
	"net/url"
	"strings"

	"cinembrot/model"
)

// GenerateTorrentLinks produces high-quality torrent candidates based on movie type and title
func GenerateTorrentLinks(movie *model.Movie) []model.DownloadLink {
	if movie == nil || strings.TrimSpace(movie.Title) == "" {
		return nil
	}

	cleanTitle := strings.TrimSpace(movie.Title)
	encodedTitle := url.QueryEscape(cleanTitle)
	yearStr := ""
	if movie.Year > 0 {
		yearStr = fmt.Sprintf("+%d", movie.Year)
	}

	var links []model.DownloadLink

	// 1. Khusus Anime: Nyaa.si & AnimeTosho (Sumber Torrent Anime Terbesar di Dunia)
	if movie.Type == "anime" {
		links = append(links, model.DownloadLink{
			MovieID:    movie.ID,
			Provider:   "Nyaa.si (Anime HD Torrent)",
			Quality:    "HD 1080p / 720p",
			Resolution: "Multi-Sub / Raw",
			Format:     "Torrent",
			URL:        fmt.Sprintf("https://nyaa.si/?f=0&c=1_2&q=%s", encodedTitle),
			IsValid:    true,
			Status:     "ACTIVE",
		})
		links = append(links, model.DownloadLink{
			MovieID:    movie.ID,
			Provider:   "AnimeTosho (Direct Torrents & DDL)",
			Quality:    "BDRip / WebRip",
			Resolution: "Full HD",
			Format:     "Torrent",
			URL:        fmt.Sprintf("https://animetosho.org/search?q=%s", encodedTitle),
			IsValid:    true,
			Status:     "ACTIVE",
		})
	} else if movie.Type == "drama_pendek" || movie.Type == "series" {
		// 2. Khusus Drama Asia & Serial TV: EZTV & 1337x
		links = append(links, model.DownloadLink{
			MovieID:    movie.ID,
			Provider:   "EZTV (TV & Drama Torrents)",
			Quality:    "HD 720p / 1080p",
			Resolution: "HD WebRip",
			Format:     "Torrent",
			URL:        fmt.Sprintf("https://eztvx.to/search/%s", encodedTitle),
			IsValid:    true,
			Status:     "ACTIVE",
		})
		links = append(links, model.DownloadLink{
			MovieID:    movie.ID,
			Provider:   "1337x (Drama Torrents)",
			Quality:    "HD 1080p",
			Resolution: "HDTV / Web-DL",
			Format:     "Torrent",
			URL:        fmt.Sprintf("https://1337x.to/search/%s%s/1/", encodedTitle, yearStr),
			IsValid:    true,
			Status:     "ACTIVE",
		})
	} else {
		// 3. Khusus Hollywood & Bioskop: 1337x & TorrentGalaxy
		links = append(links, model.DownloadLink{
			MovieID:    movie.ID,
			Provider:   "1337x (Verified Movies)",
			Quality:    "HD 1080p / 2160p 4K",
			Resolution: "Bluray / Web-DL",
			Format:     "Torrent",
			URL:        fmt.Sprintf("https://1337x.to/search/%s%s/1/", encodedTitle, yearStr),
			IsValid:    true,
			Status:     "ACTIVE",
		})
		links = append(links, model.DownloadLink{
			MovieID:    movie.ID,
			Provider:   "TorrentGalaxy (TGx Movies)",
			Quality:    "HD 1080p",
			Resolution: "Multi-Audio",
			Format:     "Torrent",
			URL:        fmt.Sprintf("https://torrentgalaxy.to/torrents.php?search=%s%s", encodedTitle, yearStr),
			IsValid:    true,
			Status:     "ACTIVE",
		})
	}

	return links
}
