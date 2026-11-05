package pipeline

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"cinembrot/model"
	"cinembrot/provider/vorta"
	"cinembrot/scraper"

	"gorm.io/gorm"
)

// StealthOptions mendefinisikan batasan aman untuk scraper Vorta
type StealthOptions struct {
	MinDelaySec int
	MaxDelaySec int
	Platforms   []string
	Limit       int
}

// IngestDramaShortPolite mengambil data drama pendek secara sopan, random platform, dan random delay
// Dirancang secara khusus untuk mencegah deteksi bot, brute-force, dan limit ban pada Vorta API (Trial 600 req/hari).
func IngestDramaShortPolite(db *gorm.DB, repo *scraper.Repository, opts StealthOptions) error {
	client := vorta.NewClient()

	if len(opts.Platforms) == 0 {
		opts.Platforms = []string{"melolo", "dramabox", "shortmax", "storyreel", "vibeshort", "dramaverse"}
	}
	if opts.MinDelaySec <= 0 {
		opts.MinDelaySec = 5
	}
	if opts.MaxDelaySec <= opts.MinDelaySec {
		opts.MaxDelaySec = opts.MinDelaySec + 7
	}
	if opts.Limit <= 0 {
		opts.Limit = 1
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Acak urutan platform agar pola request tidak seragam
	platforms := make([]string, len(opts.Platforms))
	copy(platforms, opts.Platforms)
	rng.Shuffle(len(platforms), func(i, j int) {
		platforms[i], platforms[j] = platforms[j], platforms[i]
	})

	log.Printf("[VORTA STEALTH] Memulai siklus penyerapan drama pendek (Target Limit: %d judul)...\n", opts.Limit)
	log.Printf("[VORTA STEALTH] Konfigurasi Jeda: %d-%d detik antar request. Platform Pool: %v\n", opts.MinDelaySec, opts.MaxDelaySec, platforms)

	totalIngested := 0

	for _, platform := range platforms {
		if totalIngested >= opts.Limit {
			break
		}

		// Jeda acak sebelum request awal untuk meniru perilaku manusia
		randomPreDelay := time.Duration(opts.MinDelaySec+rng.Intn(opts.MaxDelaySec-opts.MinDelaySec+1)) * time.Second
		log.Printf("[VORTA STEALTH] Jeda keamanan sebelum memeriksa platform '%s': %v...\n", platform, randomPreDelay)
		time.Sleep(randomPreDelay)

		log.Printf("[VORTA STEALTH] Memeriksa daftar katalog beranda untuk platform: %s (Lang: id)...\n", platform)
		catalog, err := client.GetHomeFeed(platform, "id")
		if err != nil {
			log.Printf("[VORTA WARN] Gagal mengambil katalog platform '%s': %v. Melanjutkan ke platform berikutnya.\n", platform, err)
			continue
		}

		if len(catalog) == 0 {
			log.Printf("[VORTA INFO] Tidak ada item ditemukan di katalog beranda platform '%s'.\n", platform)
			continue
		}

		// Acak urutan item dalam katalog agar tidak selalu memproses item index 0
		shuffledItems := make([]vorta.DramaItem, len(catalog))
		copy(shuffledItems, catalog)
		rng.Shuffle(len(shuffledItems), func(i, j int) {
			shuffledItems[i], shuffledItems[j] = shuffledItems[j], shuffledItems[i]
		})

		for _, item := range shuffledItems {
			if totalIngested >= opts.Limit {
				break
			}

			if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.ID) == "" {
				continue
			}

			// Cek duplikasi di database berdasarkan judul atau slug
			slugBase := scraper.Slugify(item.Title)
			if slugBase == "" {
				slugBase = fmt.Sprintf("short-%s", item.ID)
			}
			slug := fmt.Sprintf("%s-2026", slugBase)
			var existing model.Movie
			if err := db.Where("slug = ? OR title = ?", slug, item.Title).First(&existing).Error; err == nil {
				log.Printf("[VORTA SKIP] Judul '%s' sudah ada di MariaDB (ID: %d, Slug: %s). Lewati.\n", item.Title, existing.ID, existing.Slug)
				continue
			}

			// Berikan jeda acak manusiawi sebelum mengambil detail drama
			interDelay := time.Duration(opts.MinDelaySec+rng.Intn(opts.MaxDelaySec-opts.MinDelaySec+1)) * time.Second
			log.Printf("[VORTA STEALTH] Mengambil detail drama ID '%s' ('%s') dengan jeda acak %v...\n", item.ID, item.Title, interDelay)
			time.Sleep(interDelay)

			detail, err := client.GetDetail(platform, item.ID, "id")
			if err != nil {
				log.Printf("[VORTA WARN] Gagal mengambil detail drama ID '%s': %v\n", item.ID, err)
				continue
			}

			synopsis := strings.TrimSpace(detail.Synopsis)
			if synopsis == "" {
				synopsis = fmt.Sprintf("Nonton drama pendek %s subtitle Indonesia legal kualitas HD.", detail.Title)
			}

			posterURL := detail.Cover
			if posterURL == "" {
				posterURL = item.Cover
			}

			now := time.Now()
			movie := model.Movie{
				Title:         detail.Title,
				OriginalTitle: detail.Title,
				Slug:          slug,
				Type:          "drama_pendek",
				Year:          2026,
				Status:        "ongoing",
				Rating:        8.5,
				Synopsis:      synopsis,
				PosterURL:     posterURL,
				BackdropURL:   posterURL,
				AgeRating:     "13+",
				Country:       "China / Asia",
				SourceWebsite: "vorta",
				SourceURL:     fmt.Sprintf("vorta://%s/%s", platform, detail.ID),
				ReleaseDate:   &now,
			}

			// Simpan data movie ke MariaDB
			if err := db.Create(&movie).Error; err != nil {
				log.Printf("[VORTA ERROR] Gagal menyimpan movie '%s' ke database: %v\n", movie.Title, err)
				continue
			}

			log.Printf("[VORTA SUCCESS] Berhasil insert drama baru ID: %d, Judul: '%s', Total Episode: %d\n", movie.ID, movie.Title, detail.TotalEpisodes)

			// Ambil HANYA Episode 1 (Chapter 1) sebagai sample stream awal on-demand (Anti-Harvester Policy)
			if len(detail.Chapters) > 0 {
				firstChapter := detail.Chapters[0]
				chapterID := firstChapter.ID
				if chapterID == "" {
					chapterID = "1"
				}

				// Jeda acak sebelum mengambil stream episode 1
				streamDelay := time.Duration(opts.MinDelaySec+rng.Intn(opts.MaxDelaySec-opts.MinDelaySec+1)) * time.Second
				log.Printf("[VORTA STEALTH] Mengambil sample stream episode 1 (Chapter %s) dengan jeda %v...\n", chapterID, streamDelay)
				time.Sleep(streamDelay)

				streamData, err := client.GetVideoStream(platform, detail.ID, chapterID, "id")
				if err != nil {
					log.Printf("[VORTA WARN] Gagal mengambil video stream episode 1: %v\n", err)
				} else if len(streamData.Streams) > 0 {
					streamURL := streamData.Streams[0].URL
					streamLink := model.StreamLink{
						MovieID:    movie.ID,
						Provider:   platform,
						ServerName: fmt.Sprintf("Server 1 - %s HD (M3U8)", strings.Title(platform)),
						EmbedURL:   streamURL,
						DirectURL:  streamURL,
						Quality:    "720p HLS",
						IsValid:    true,
						Status:     "ACTIVE",
					}
					if err := db.Create(&streamLink).Error; err != nil {
						log.Printf("[VORTA WARN] Gagal menyimpan stream link episode 1: %v\n", err)
					} else {
						log.Printf("[VORTA SUCCESS] Episode 1 Stream tersimpan untuk Movie ID %d (Link ID: %d)\n", movie.ID, streamLink.ID)
					}
				}
			}

			totalIngested++
			log.Printf("[VORTA PROGRESS] Terpenuhi %d dari batas %d judul.\n", totalIngested, opts.Limit)
		}
	}

	log.Printf("[VORTA FINISH] Siklus selesai. Total judul drama pendek baru yang berhasil diserap: %d\n", totalIngested)
	return nil
}
