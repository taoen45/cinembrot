package torrentmgr_test

import (
	"cinembrot/torrentmgr"
	"testing"
)

func TestValidateMagnetWithRust(t *testing.T) {
	validMagnet := "magnet:?xt=urn:btih:c12fe1c06bba254a9dc9f519b335aa7c1367a88a&dn=Ubuntu"
	invalidMagnet := "https://cinembrot.web.id/sample.mp4"

	// Uji valid magnet (menguji fallback aman atau binary langsung)
	isValid, err := torrentmgr.ValidateMagnetWithRust("", validMagnet)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !isValid {
		t.Errorf("Expected valid magnet to return true")
	}

	// Uji invalid magnet
	isInvalid, err := torrentmgr.ValidateMagnetWithRust("", invalidMagnet)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if isInvalid {
		t.Errorf("Expected invalid URL to return false")
	}
}

