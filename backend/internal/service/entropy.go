package service

import (
	"math"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	// Code block pattern (markdown code blocks with ```)
	codeBlockRegex = regexp.MustCompile("(?s)```.*?```")

	// UUID pattern
	uuidRegex = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

	// API key patterns (e.g. sk-..., ck_live_...)
	apiKeyRegex = regexp.MustCompile(`(?i)\b(sk-[a-zA-Z0-9]{20,}|ck_[a-zA-Z0-9_]{16,}|api_key_[a-zA-Z0-9]{16,})\b`)

	// Base64 pattern (continuous base64 strings of length 32+)
	base64Regex = regexp.MustCompile(`\b[A-Za-z0-9+/]{32,}={0,2}\b`)

	// CJK Unicode Range (Chinese, Japanese, Korean characters)
	cjkRegex = regexp.MustCompile(`[\x{4E00}-\x{9FFF}\x{3040}-\x{309F}\x{30A0}-\x{30FF}\x{AC00}-\x{D7AF}]`)
)

// CalculateShannonEntropy returns the Shannon entropy (bits per character) of a string.
func CalculateShannonEntropy(text string) float64 {
	if len(text) == 0 {
		return 0.0
	}

	charCounts := make(map[rune]int)
	totalRunes := 0

	for _, r := range text {
		charCounts[r]++
		totalRunes++
	}

	if totalRunes == 0 {
		return 0.0
	}

	entropy := 0.0
	for _, count := range charCounts {
		p := float64(count) / float64(totalRunes)
		entropy -= p * math.Log2(p)
	}

	return entropy
}

// IsWhitelistedTechnicalContent checks if text contains code, keys, UUIDs, Base64 strings, or CJK characters.
// Technical content is whitelisted to prevent false-positive entropy rejections.
func IsWhitelistedTechnicalContent(text string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}

	if codeBlockRegex.MatchString(text) {
		return true
	}
	if uuidRegex.MatchString(text) {
		return true
	}
	if apiKeyRegex.MatchString(text) {
		return true
	}
	if base64Regex.MatchString(text) {
		return true
	}
	if cjkRegex.MatchString(text) {
		return true
	}

	// Short texts (<30 chars) are also exempt from entropy flagging
	if utf8.RuneCountInString(text) < 30 {
		return true
	}

	return false
}
