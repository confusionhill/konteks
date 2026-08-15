package service

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

type TimeRange struct {
	StartTime time.Time
	EndTime   time.Time
	IsRecall  bool
	Label     string
}

var (
	// Matches relative months: e.g. "8 months ago", "8 month ago", "8m ago"
	monthsAgoRegex = regexp.MustCompile(`(?i)\b(\d+)\s*(?:months?|mth|mths)\s*ago\b`)

	// Matches relative weeks: e.g. "2 weeks ago"
	weeksAgoRegex = regexp.MustCompile(`(?i)\b(\d+)\s*(?:weeks?|wks?)\s*ago\b`)

	// Matches relative days: e.g. "30 days ago"
	daysAgoRegex = regexp.MustCompile(`(?i)\b(\d+)\s*(?:days?|d)\s*ago\b`)

	// Matches "last year" or "1 year ago"
	lastYearRegex = regexp.MustCompile(`(?i)\b(?:last\s*year|1\s*year\s*ago|\d+\s*years?\s*ago)\b`)

	// Matches "back in ..." or "previously" or "history"
	historicalKeywordRegex = regexp.MustCompile(`(?i)\b(?:previously|history|earlier|back\s*when|in\s*the\s*past)\b`)
)

// ExtractTemporalBound parses natural text queries for explicit temporal expressions.
func ExtractTemporalBound(query string, now time.Time) (*TimeRange, bool) {
	if strings.TrimSpace(query) == "" {
		return nil, false
	}

	if match := monthsAgoRegex.FindStringSubmatch(query); len(match) > 1 {
		months, err := strconv.Atoi(match[1])
		if err == nil && months > 0 {
			targetTime := now.AddDate(0, -months, 0)
			return &TimeRange{
				StartTime: targetTime.AddDate(0, 0, -15),
				EndTime:   targetTime.AddDate(0, 0, 15),
				IsRecall:  true,
				Label:     match[0],
			}, true
		}
	}

	if match := weeksAgoRegex.FindStringSubmatch(query); len(match) > 1 {
		weeks, err := strconv.Atoi(match[1])
		if err == nil && weeks > 0 {
			targetTime := now.AddDate(0, 0, -weeks*7)
			return &TimeRange{
				StartTime: targetTime.AddDate(0, 0, -4),
				EndTime:   targetTime.AddDate(0, 0, 4),
				IsRecall:  true,
				Label:     match[0],
			}, true
		}
	}

	if match := daysAgoRegex.FindStringSubmatch(query); len(match) > 1 {
		days, err := strconv.Atoi(match[1])
		if err == nil && days > 0 {
			targetTime := now.AddDate(0, 0, -days)
			return &TimeRange{
				StartTime: targetTime.AddDate(0, 0, -2),
				EndTime:   targetTime.AddDate(0, 0, 2),
				IsRecall:  true,
				Label:     match[0],
			}, true
		}
	}

	if lastYearRegex.MatchString(query) {
		targetTime := now.AddDate(-1, 0, 0)
		return &TimeRange{
			StartTime: targetTime.AddDate(0, -2, 0),
			EndTime:   targetTime.AddDate(0, 2, 0),
			IsRecall:  true,
			Label:     "historical_year",
		}, true
	}

	if historicalKeywordRegex.MatchString(query) {
		return &TimeRange{
			StartTime: now.AddDate(-5, 0, 0),
			EndTime:   now,
			IsRecall:  true,
			Label:     "historical_keyword",
		}, true
	}

	return nil, false
}
