package server_test

import (
	"cinembrot/server"
	"testing"
)

func TestValidateSlug(t *testing.T) {
	validSlugs := []string{
		"avengers-endgame-2019",
		"isekai-quartet",
		"movie_123",
	}

	for _, s := range validSlugs {
		if !server.ValidateSlug(s) {
			t.Errorf("Expected valid slug for %q", s)
		}
	}

	invalidSlugs := []string{
		"../../../etc/passwd",
		"movie<script>",
		"movie; DROP TABLE movies;",
		"",
	}

	for _, s := range invalidSlugs {
		if server.ValidateSlug(s) {
			t.Errorf("Expected invalid slug for %q", s)
		}
	}
}

func TestSanitizeSearchQuery(t *testing.T) {
	input := "Naruto \x00 Shippuden"
	got := server.SanitizeSearchQuery(input)
	expected := "Naruto  Shippuden"
	if got != expected {
		t.Errorf("SanitizeSearchQuery(%q) = %q; want %q", input, got, expected)
	}
}
