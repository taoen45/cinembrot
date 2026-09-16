package translator

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	cacheMu sync.RWMutex
	cache   = make(map[string]string)

	httpClient = &http.Client{
		Timeout: 12 * time.Second,
	}
)

// Translate translates a given text into targetLang ("id" or "en").
// Returns translated text, detected source language, and error.
func Translate(text, targetLang, sourceLang string) (string, string, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", "", nil
	}

	if sourceLang == "" {
		sourceLang = "auto"
	}

	cacheKey := fmt.Sprintf("%s:%s:%s", targetLang, sourceLang, trimmed)
	cacheMu.RLock()
	if val, ok := cache[cacheKey]; ok {
		cacheMu.RUnlock()
		return val, "", nil
	}
	cacheMu.RUnlock()

	// Jika teks sangat panjang (> 1800 char), pecah per paragraf/kalimat
	if len(trimmed) > 1800 {
		return translateLongText(trimmed, targetLang, sourceLang)
	}

	translated, detected, err := callTranslateAPI(trimmed, targetLang, sourceLang)
	if err != nil {
		return trimmed, "", err
	}

	cacheMu.Lock()
	cache[cacheKey] = translated
	cacheMu.Unlock()

	return translated, detected, nil
}

// callTranslateAPI executes translation via Google Clients5 Chrome Client API
func callTranslateAPI(text, targetLang, sourceLang string) (string, string, error) {
	endpoint := fmt.Sprintf("https://clients5.google.com/translate_a/t?client=dict-chrome-ex&sl=%s&tl=%s&q=%s",
		url.QueryEscape(sourceLang), url.QueryEscape(targetLang), url.QueryEscape(text))

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return text, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		return text, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return text, "", fmt.Errorf("translate API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return text, "", err
	}

	// 1. Coba parse nested array: [["Translated text 1", "en"], ["Translated text 2", "en"]]
	var nestedArray [][]interface{}
	if err := json.Unmarshal(body, &nestedArray); err == nil && len(nestedArray) > 0 {
		var parts []string
		detectedLang := ""
		for _, item := range nestedArray {
			if len(item) > 0 {
				if str, ok := item[0].(string); ok {
					parts = append(parts, str)
				}
			}
			if len(item) > 1 && detectedLang == "" {
				if lang, ok := item[1].(string); ok {
					detectedLang = strings.ToLower(strings.TrimSpace(lang))
				}
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, " "), detectedLang, nil
		}
	}

	// 2. Coba parse single array: ["Translated text", "en"]
	var singleList []interface{}
	if err := json.Unmarshal(body, &singleList); err == nil && len(singleList) > 0 {
		translatedStr := ""
		detectedLang := ""
		if str, ok := singleList[0].(string); ok {
			translatedStr = str
		}
		if len(singleList) > 1 {
			if lang, ok := singleList[1].(string); ok {
				detectedLang = strings.ToLower(strings.TrimSpace(lang))
			}
		}
		if translatedStr != "" {
			return translatedStr, detectedLang, nil
		}
	}

	return text, "", fmt.Errorf("failed to parse translate response: %s", string(body))
}

// translateLongText splits text into paragraphs and translates each
func translateLongText(text, targetLang, sourceLang string) (string, string, error) {
	paragraphs := strings.Split(text, "\n\n")
	var translatedParts []string
	detectedLang := ""

	for _, p := range paragraphs {
		pTrimmed := strings.TrimSpace(p)
		if pTrimmed == "" {
			continue
		}
		tPart, det, err := callTranslateAPI(pTrimmed, targetLang, sourceLang)
		if err != nil {
			translatedParts = append(translatedParts, pTrimmed)
		} else {
			translatedParts = append(translatedParts, tPart)
			if detectedLang == "" && det != "" {
				detectedLang = det
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	return strings.Join(translatedParts, "\n\n"), detectedLang, nil
}

// TranslateToIndonesian translates any text into Indonesian
func TranslateToIndonesian(text string) (string, error) {
	trans, _, err := Translate(text, "id", "auto")
	return trans, err
}

// TranslateToEnglish translates any text into English
func TranslateToEnglish(text string) (string, error) {
	trans, _, err := Translate(text, "en", "auto")
	return trans, err
}

// TranslateBilingual takes a raw synopsis in any language and produces:
// 1. Indonesian Synopsis (synopsisID)
// 2. English Synopsis (synopsisEN)
func TranslateBilingual(rawSynopsis string) (synopsisID, synopsisEN string, err error) {
	raw := strings.TrimSpace(rawSynopsis)
	if raw == "" {
		return "", "", nil
	}

	// 1. Coba terjemahkan ke Bahasa Indonesia
	transID, detectedLang, transErr := Translate(raw, "id", "auto")
	if transErr != nil {
		log.Printf("[WARN] Auto-translate synopsis ke ID error: %v\n", transErr)
		// Fallback jika gagal translate: gunakan raw untuk keduanya
		return raw, raw, transErr
	}

	synopsisID = transID

	// 2. Tentukan sinopsis Bahasa Inggris
	switch detectedLang {
	case "en":
		// Jika bahasa aslinya memang English, gunakan raw asli sebagai English
		synopsisEN = raw
	case "id":
		// Jika bahasa aslinya Bahasa Indonesia, buatkan versi English
		synopsisID = raw
		if en, err := TranslateToEnglish(raw); err == nil && en != "" {
			synopsisEN = en
		} else {
			synopsisEN = raw
		}
	default:
		// Jika bahasa aslinya non-Latin / non-English (Jepang, Mandarin, Korea, dll.)
		if en, err := TranslateToEnglish(raw); err == nil && en != "" {
			synopsisEN = en
		} else {
			synopsisEN = raw
		}
	}

	return synopsisID, synopsisEN, nil
}
