package torrentmgr

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// MagnetValidationResult merepresentasikan output JSON dari Rust media-worker
type MagnetValidationResult struct {
	Valid bool   `json:"valid"`
	URI   string `json:"uri"`
}

// ValidateMagnetWithRust memvalidasi magnet link menggunakan Rust media-worker CLI jika binary tersedia
// Jika binary tidak ditemukan, fungsi ini melakukan fallback otomatis ke native Go parser tanpa error
func ValidateMagnetWithRust(workerPath string, magnetURI string) (bool, error) {
	cleanURI := strings.TrimSpace(magnetURI)
	if cleanURI == "" {
		return false, fmt.Errorf("empty magnet URI")
	}

	// Default fallback path ke target release atau debug
	if workerPath == "" {
		workerPath = filepath.Join("crates", "media-worker", "target", "debug", "media_worker.exe")
	}

	cmd := exec.Command(workerPath, "validate-magnet", cleanURI)
	output, err := cmd.Output()
	if err != nil {
		// Fallback ke Go native parsing
		return strings.HasPrefix(cleanURI, "magnet:?") && strings.Contains(cleanURI, "xt=urn:btih:"), nil
	}

	var res MagnetValidationResult
	if err := json.Unmarshal(output, &res); err != nil {
		// Fallback ke Go native parsing jika unmarshal JSON gagal
		return strings.HasPrefix(cleanURI, "magnet:?") && strings.Contains(cleanURI, "xt=urn:btih:"), nil
	}

	return res.Valid, nil
}

