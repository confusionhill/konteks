package service

import (
	"testing"
	"time"
)

func TestExtractTemporalBound(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		query      string
		wantRecall bool
	}{
		{
			name:       "Relative months ago",
			query:      "What was the database schema we decided on 8 months ago?",
			wantRecall: true,
		},
		{
			name:       "Relative weeks ago",
			query:      "Can you retrieve our meeting notes from 2 weeks ago?",
			wantRecall: true,
		},
		{
			name:       "Last year query",
			query:      "Show me the project setup from last year",
			wantRecall: true,
		},
		{
			name:       "Standard active query",
			query:      "How do I configure postgres database connection?",
			wantRecall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, found := ExtractTemporalBound(tt.query, now)
			if found != tt.wantRecall {
				t.Errorf("ExtractTemporalBound(%q) found = %v; want %v", tt.query, found, tt.wantRecall)
			}
			if found && res == nil {
				t.Errorf("ExtractTemporalBound(%q) returned nil TimeRange", tt.query)
			}
		})
	}
}
