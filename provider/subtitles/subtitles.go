package subtitles

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cinembrot/model"
)

// GenerateSubtitleDownloadLinks generates clean download & search links for Indonesian and English subtitles
func GenerateSubtitleDownloadLinks(title string, year int) []model.DownloadLink {
	query := title
	if year > 0 {
		query = fmt.Sprintf("%s %d", title, year)
	}
	escapedQuery := url.QueryEscape(query)

	return []model.DownloadLink{
		{
			Provider:   "Subtitle Indonesia (SRT)",
			Quality:    "SRT",
			Resolution: "Sub",
			Format:     "SRT",
			URL:        fmt.Sprintf("https://subdl.com/search/%s", escapedQuery),
			IsValid:    true,
			Status:     "ACTIVE",
		},
		{
			Provider:   "Subtitle English (SRT)",
			Quality:    "SRT",
			Resolution: "Sub",
			Format:     "SRT",
			URL:        fmt.Sprintf("https://www.opensubtitles.org/en/search/sublanguageid-eng/moviename-%s", escapedQuery),
			IsValid:    true,
			Status:     "ACTIVE",
		},
	}
}

// PrepareCandidateSubtitles writes initial candidate subtitles (Indonesian & English) and creates WebVTT
func PrepareCandidateSubtitles(title, slug, subDir string) []model.SubtitleOption {
	_ = os.MkdirAll(subDir, 0755)

	candidates := []model.SubtitleOption{
		{
			ID:        "sub-id-1",
			Title:     "Pilihan 1: Bahasa Indonesia (Official / Terjemahan Baku)",
			Source:    "SubDL Community",
			Language:  "Indonesian",
			SRTPath:   filepath.Join(subDir, fmt.Sprintf("%s_id.srt", slug)),
			VTTPath:   fmt.Sprintf("/downloads/%s_id.vtt", slug),
			IsDefault: true,
		},
		{
			ID:        "sub-en-2",
			Title:     "Pilihan 2: English (Official / Sync)",
			Source:    "OpenSubtitles v3",
			Language:  "English",
			SRTPath:   filepath.Join(subDir, fmt.Sprintf("%s_en.srt", slug)),
			VTTPath:   fmt.Sprintf("/downloads/%s_en.vtt", slug),
			IsDefault: false,
		},
	}

	upperTitle := strings.ToUpper(title)

	// Indonesian Subtitle Template
	subIDContent := fmt.Sprintf("1\n00:00:05,000 --> 00:00:10,000\n<b>CINEMBROT MEMPERSEMBAHKAN</b>\n\n"+
		"2\n00:00:12,000 --> 00:00:18,000\n<b>\"%s\"</b>\n\n"+
		"3\n00:00:20,000 --> 00:00:26,000\n<i>Subtitle Bahasa Indonesia Resmi</i>\n\n"+
		"4\n00:00:28,000 --> 00:00:35,000\nSelamat menyaksikan. Kunjungi cinembrot.my.id untuk rilisan terbaru lainnya.\n",
		upperTitle)

	// English Subtitle Template
	subENContent := fmt.Sprintf("1\n00:00:05,000 --> 00:00:10,000\n<b>CINEMBROT PRESENTS</b>\n\n"+
		"2\n00:00:12,000 --> 00:00:18,000\n<b>\"%s\"</b>\n\n"+
		"3\n00:00:20,000 --> 00:00:26,000\n<i>Official English Subtitles</i>\n\n"+
		"4\n00:00:28,000 --> 00:00:35,000\nEnjoy the show. Visit cinembrot.my.id for more streaming and releases.\n",
		upperTitle)

	_ = os.WriteFile(candidates[0].SRTPath, []byte(subIDContent), 0644)
	_ = os.WriteFile(candidates[1].SRTPath, []byte(subENContent), 0644)

	// Convert all .srt to .vtt via FFmpeg if available
	for _, c := range candidates {
		vttFile := filepath.Join(subDir, filepath.Base(c.VTTPath))
		cmd := exec.Command("ffmpeg", "-i", c.SRTPath, "-y", vttFile)
		_ = cmd.Run()
	}

	return candidates
}
