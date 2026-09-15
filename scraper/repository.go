package scraper

import (
	"log"
	"strings"
	"time"

	"cinembrot/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// UpsertMovie saves or updates movie and all related relations (genres, cast, links)
func (r *Repository) UpsertMovie(movie *model.Movie) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Process & Associate Genres (find or create)
		var resolvedGenres []model.Genre
		for _, g := range movie.Genres {
			if strings.TrimSpace(g.Name) == "" {
				continue
			}
			slug := g.Slug
			if slug == "" {
				slug = Slugify(g.Name)
			}
			var existingGenre model.Genre
			if err := tx.Where("slug = ? OR name = ?", slug, g.Name).First(&existingGenre).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					newGenre := model.Genre{Name: g.Name, Slug: slug}
					if err := tx.Create(&newGenre).Error; err == nil {
						resolvedGenres = append(resolvedGenres, newGenre)
					}
				}
			} else {
				resolvedGenres = append(resolvedGenres, existingGenre)
			}
		}
		movie.Genres = resolvedGenres

		// 2. Process & Associate Directors (find or create)
		var resolvedDirectors []model.Director
		for _, d := range movie.Directors {
			if strings.TrimSpace(d.Name) == "" {
				continue
			}
			slug := d.Slug
			if slug == "" {
				slug = Slugify(d.Name)
			}
			var existingDirector model.Director
			if err := tx.Where("slug = ? OR name = ?", slug, d.Name).First(&existingDirector).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					newDirector := model.Director{Name: d.Name, Slug: slug, PhotoURL: d.PhotoURL}
					if err := tx.Create(&newDirector).Error; err == nil {
						resolvedDirectors = append(resolvedDirectors, newDirector)
					}
				}
			} else {
				resolvedDirectors = append(resolvedDirectors, existingDirector)
			}
		}
		movie.Directors = resolvedDirectors

		// 3. Process & Associate Actors (find or create)
		var resolvedActors []model.Actor
		for _, a := range movie.Actors {
			if strings.TrimSpace(a.Name) == "" {
				continue
			}
			slug := a.Slug
			if slug == "" {
				slug = Slugify(a.Name)
			}
			var existingActor model.Actor
			if err := tx.Where("slug = ? OR name = ?", slug, a.Name).First(&existingActor).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					newActor := model.Actor{Name: a.Name, Slug: slug, CharacterName: a.CharacterName, PhotoURL: a.PhotoURL}
					if err := tx.Create(&newActor).Error; err == nil {
						resolvedActors = append(resolvedActors, newActor)
					}
				}
			} else {
				resolvedActors = append(resolvedActors, existingActor)
			}
		}
		movie.Actors = resolvedActors

		incomingDownloads := movie.DownloadLinks
		incomingStreams := movie.StreamLinks

		// 4. Save/Update Movie record
		var existingMovie model.Movie
		err := tx.Where("source_url = ? OR slug = ?", movie.SourceURL, movie.Slug).First(&existingMovie).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// Insert new movie without duplicate links
				movie.DownloadLinks = nil
				movie.StreamLinks = nil
				if err := tx.Create(movie).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			// 🛡️ PROTEKSI CMS: Jika film sudah diedit manual oleh admin/user, jangan timpa datanya!
			if existingMovie.IsManualEdit {
				log.Printf("[SCRAPER LOCK 🔒] Film '%s' (ID: %d) TIDAK ditimpa karena telah diedit manual oleh admin CMS.\n",
					existingMovie.Title, existingMovie.ID)
				return nil
			}

			// Pertahankan sinopsis lama jika sinopsis baru kosong
			if strings.TrimSpace(movie.Synopsis) == "" && strings.TrimSpace(existingMovie.Synopsis) != "" {
				movie.Synopsis = existingMovie.Synopsis
			}

			// Jika judul lama berkarakter non-Latin dan judul baru alfabet latin, prioritaskan judul baru
			if ContainsNonLatin(existingMovie.Title) && !ContainsNonLatin(movie.Title) {
				if movie.AlternativeTitles == "" {
					movie.AlternativeTitles = existingMovie.Title
				}
				if movie.OriginalTitle == "" {
					movie.OriginalTitle = existingMovie.Title
				}
			}

			// Update existing movie without cascading has-many
			movie.ID = existingMovie.ID
			movie.DownloadLinks = nil
			movie.StreamLinks = nil
			if err := tx.Omit("DownloadLinks", "StreamLinks").Save(movie).Error; err != nil {
				return err
			}
		}

		// Replace many2many relations
		if err := tx.Model(movie).Association("Genres").Replace(movie.Genres); err != nil {
			return err
		}
		if err := tx.Model(movie).Association("Directors").Replace(movie.Directors); err != nil {
			return err
		}
		if err := tx.Model(movie).Association("Actors").Replace(movie.Actors); err != nil {
			return err
		}

		// 5. Deduplicated Upsert for DownloadLinks
		for _, dl := range incomingDownloads {
			cleanURL := strings.TrimSpace(dl.URL)
			if cleanURL == "" {
				continue
			}
			var existingDL model.DownloadLink
			if err := tx.Where("movie_id = ? AND url = ?", movie.ID, cleanURL).First(&existingDL).Error; err != nil {
				dl.ID = 0
				dl.MovieID = movie.ID
				dl.URL = cleanURL
				_ = tx.Create(&dl).Error
			}
		}

		// 6. Deduplicated Upsert for StreamLinks
		for _, sl := range incomingStreams {
			cleanEmbed := strings.TrimSpace(sl.EmbedURL)
			if cleanEmbed == "" {
				continue
			}
			var existingSL model.StreamLink
			if err := tx.Where("movie_id = ? AND embed_url = ?", movie.ID, cleanEmbed).First(&existingSL).Error; err != nil {
				sl.ID = 0
				sl.MovieID = movie.ID
				sl.EmbedURL = cleanEmbed
				_ = tx.Create(&sl).Error
			}
		}

		return nil
	})
}

// LogScrape records a scraping operation history
func (r *Repository) LogScrape(sourceWebsite, targetURL, status string, itemsScraped int, errorMsg string, duration time.Duration) error {
	logEntry := model.ScrapeLog{
		SourceWebsite:   sourceWebsite,
		TargetURL:       targetURL,
		Status:          status,
		ItemsScraped:    itemsScraped,
		ErrorMessage:    errorMsg,
		ExecutionTimeMs: duration.Milliseconds(),
	}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&logEntry).Error
}

// Slugify generates a clean URL slug from title string
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var result []rune
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result = append(result, r)
		} else if r == ' ' || r == '-' || r == '_' {
			if len(result) > 0 && result[len(result)-1] != '-' {
				result = append(result, '-')
			}
		}
	}
	return strings.Trim(string(result), "-")
}
