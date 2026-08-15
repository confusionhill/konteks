package service

import (
	"testing"
)

func TestCalculateShannonEntropy(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		minScore float64
		maxScore float64
	}{
		{
			name:     "Low entropy repeated text",
			text:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			minScore: 0.0,
			maxScore: 0.1,
		},
		{
			name:     "Normal English prose",
			text:     "The user moved to San Francisco to work on backend distributed systems.",
			minScore: 3.5,
			maxScore: 4.5,
		},
		{
			name:     "High entropy random string",
			text:     "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=[]{}|;:,.<>?",
			minScore: 4.8,
			maxScore: 6.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateShannonEntropy(tt.text)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("CalculateShannonEntropy(%q) = %v; want between %v and %v", tt.text, score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestIsWhitelistedTechnicalContent(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "Code block",
			text:     "```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```",
			expected: true,
		},
		{
			name:     "API Key",
			text:     "Here is my key: sk-proj-1234567890abcdefghijklmnopqrstuvwxyz",
			expected: true,
		},
		{
			name:     "UUID",
			text:     "5ba87043-412b-4c81-92d9-b613608658d5",
			expected: true,
		},
		{
			name:     "Base64 String",
			text:     "eyJhbGciOiJBRVMyNTYiLCJjdHgiOiJhbGciOiJBRVMyNTYiLCJjdHgiOiJ",
			expected: true,
		},
		{
			name:     "CJK Text",
			text:     "ユーザーは東京からサンフランシスコに引っ越しました。",
			expected: true,
		},
		{
			name:     "Long plain prose text",
			text:     "This is a long normal English text about database schema design and context retrieval strategies.",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsWhitelistedTechnicalContent(tt.text)
			if result != tt.expected {
				t.Errorf("IsWhitelistedTechnicalContent(%q) = %v; want %v", tt.text, result, tt.expected)
			}
		})
	}
}
