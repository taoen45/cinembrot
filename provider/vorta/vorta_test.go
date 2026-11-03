package vorta_test

import (
	"cinembrot/provider/vorta"
	"testing"
)

func TestVortaKeyStatus(t *testing.T) {
	client := vorta.NewClient()
	status, err := client.GetKeyStatus()
	if err != nil {
		t.Fatalf("Gagal mengambil status Vorta key: %v", err)
	}

	if !status.Success {
		t.Errorf("Expected success = true, got false")
	}

	if status.Data.Limit <= 0 {
		t.Errorf("Expected limit > 0, got %d", status.Data.Limit)
	}

	t.Logf("Vorta API Key Valid: Label=%s, Remaining=%d/%d, ResetsIn=%s",
		status.Data.Label, status.Data.Remaining, status.Data.Limit, status.Data.ResetsIn)
}

func TestVortaGetCategories(t *testing.T) {
	client := vorta.NewClient()
	cats, err := client.GetCategories()
	if err != nil {
		t.Fatalf("Gagal mengambil kategori: %v", err)
	}

	if len(cats) == 0 {
		t.Errorf("Expected at least 1 category, got 0")
	}

	t.Logf("Kategori didukung: %d platform (contoh: %s)", len(cats), cats[0].DisplayName)
}

func TestVortaGetSingleDramaAndStream(t *testing.T) {
	client := vorta.NewClient()

	// 1. Ambil 1 judul dari home meloshort
	feed, err := client.GetHomeFeed("meloshort", "id")
	if err != nil {
		t.Fatalf("Gagal mengambil feed home: %v", err)
	}
	if len(feed) == 0 {
		t.Skip("Feed home kosong, lewati pengujian stream.")
	}

	target := feed[0]
	t.Logf("Judul uji: %s (ID: %s)", target.Title, target.ID)

	// 2. Ambil detail untuk mendapatkan daftar chapter
	detail, err := client.GetDetail("meloshort", target.ID, "id")
	if err != nil {
		t.Fatalf("Gagal mengambil detail drama: %v", err)
	}
	if len(detail.Chapters) == 0 {
		t.Skip("Daftar chapter drama kosong.")
	}

	firstChapter := detail.Chapters[0]
	t.Logf("Episode 1 ID: %s (Index: %d)", firstChapter.ID, firstChapter.Index)

	// 3. Ambil stream URL episode 1
	stream, err := client.GetVideoStream("meloshort", target.ID, firstChapter.ID, "id")
	if err != nil {
		t.Fatalf("Gagal mengambil video stream: %v", err)
	}

	if len(stream.Streams) == 0 {
		t.Fatalf("Tidak ada stream URL yang dikembalikan.")
	}

	t.Logf("Stream berhasil diambil: Resolusi=%s, URL=%s", stream.Streams[0].Resolution, stream.Streams[0].URL)
}

