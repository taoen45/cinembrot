package scraper

import (
	"html"
	"regexp"
	"strings"
	"unicode"
)

// ContainsNonLatin checks if a string contains CJK, Hiragana, Katakana, or Hangul characters
func ContainsNonLatin(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r) {
			return true
		}
	}
	return false
}


var (
	scriptRegex    = regexp.MustCompile(`(?i)<(script|style)[^>]*>[\s\S]*?</(script|style)>`)
	supRegex       = regexp.MustCompile(`(?i)<sup[^>]*>[\s\S]*?</sup>`)
	headingRegex   = regexp.MustCompile(`(?i)<(h[1-6]|div|p|br|tr)[^>]*>`)
	tagRegex       = regexp.MustCompile(`<[^>]+>`)
	citationRegex  = regexp.MustCompile(`\[\d+\]|\[cite[^\]]*\]`)
	multiSpace     = regexp.MustCompile(`[ \t]+`)
	multiNewline   = regexp.MustCompile(`\n{3,}`)
)

// CleanHTMLToPlainText converts dirty HTML text (like Wikipedia/Archive.org dumps) into clean readable plain text
func CleanHTMLToPlainText(dirtyHTML string) string {
	if strings.TrimSpace(dirtyHTML) == "" {
		return ""
	}

	text := dirtyHTML

	// 1. Remove script, style, and citation superscripts (e.g. Wikipedia reference tags)
	text = scriptRegex.ReplaceAllString(text, "")
	text = supRegex.ReplaceAllString(text, "")

	// 2. Replace paragraph/break/heading tags with double newlines
	text = headingRegex.ReplaceAllString(text, "\n\n")

	// 3. Strip all remaining HTML tags
	text = tagRegex.ReplaceAllString(text, "")

	// 4. Decode HTML entities (e.g. &amp;, &quot;, &#39;, &nbsp;)
	text = html.UnescapeString(text)

	// 5. Remove leftover citation brackets like [1], [2], [cite-bracket]
	text = citationRegex.ReplaceAllString(text, "")

	// 6. Clean up whitespace and newlines
	lines := strings.Split(text, "\n")
	var cleanedLines []string
	for _, line := range lines {
		line = multiSpace.ReplaceAllString(strings.TrimSpace(line), " ")
		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}

	cleanResult := strings.Join(cleanedLines, "\n\n")
	cleanResult = multiNewline.ReplaceAllString(cleanResult, "\n\n")

	return strings.TrimSpace(cleanResult)
}
