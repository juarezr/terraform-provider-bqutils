package sqlparse

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// IntervalValue canonical form used by BigQuery Tables API / google_bigquery_table.max_staleness.
var canonicalIntervalRE = regexp.MustCompile(`^\d+-\d+ \d+ \d+:\d+:\d+(\.\d+)?$`)

var simpleIntervalRE = regexp.MustCompile(`(?i)^INTERVAL\s+(\d+)\s+(YEAR|YEARS|MONTH|MONTHS|DAY|DAYS|HOUR|HOURS|MINUTE|MINUTES|SECOND|SECONDS)\s*$`)

var quotedHourToSecondRE = regexp.MustCompile(`(?i)^INTERVAL\s+["']([^"']+)["']\s+HOUR\s+TO\s+SECOND\s*$`)

// normalizeMaxStaleness converts SQL INTERVAL option values to IntervalValue encoding
// (Y-M D H:M:S) expected by the BigQuery Tables API.
func normalizeMaxStaleness(v string) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return s
	}
	if canonicalIntervalRE.MatchString(s) {
		return s
	}
	if m := simpleIntervalRE.FindStringSubmatch(s); m != nil {
		n, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			return v
		}
		return formatIntervalFromParts(0, 0, 0, n, strings.ToUpper(m[2]))
	}
	if m := quotedHourToSecondRE.FindStringSubmatch(s); m != nil {
		h, min, sec, ok := parseHMS(m[1])
		if !ok {
			return v
		}
		return fmt.Sprintf("0-0 0 %d:%d:%d", h, min, sec)
	}
	return v
}

func parseHMS(s string) (h, m, sec int64, ok bool) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) < 1 || len(parts) > 3 {
		return 0, 0, 0, false
	}
	nums := make([]int64, 3)
	for i, p := range parts {
		n, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return 0, 0, 0, false
		}
		nums[i] = n
	}
	// "4:0:0" => H:M:S; "4:0" => H:M; "4" => H
	switch len(parts) {
	case 1:
		return nums[0], 0, 0, true
	case 2:
		return nums[0], nums[1], 0, true
	default:
		return nums[0], nums[1], nums[2], true
	}
}

func formatIntervalFromParts(years, months, days, n int64, unit string) string {
	switch unit {
	case "YEAR", "YEARS":
		years += n
	case "MONTH", "MONTHS":
		months += n
	case "DAY", "DAYS":
		days += n
	case "HOUR", "HOURS":
		h, m, s := normalizeHMS(n, 0, 0)
		return fmt.Sprintf("%d-%d %d %d:%d:%d", years, months, days, h, m, s)
	case "MINUTE", "MINUTES":
		h, m, s := normalizeHMS(0, n, 0)
		return fmt.Sprintf("%d-%d %d %d:%d:%d", years, months, days, h, m, s)
	case "SECOND", "SECONDS":
		h, m, s := normalizeHMS(0, 0, n)
		return fmt.Sprintf("%d-%d %d %d:%d:%d", years, months, days, h, m, s)
	}
	return fmt.Sprintf("%d-%d %d 0:0:0", years, months, days)
}

func normalizeHMS(hours, minutes, seconds int64) (h, m, s int64) {
	total := hours*3600 + minutes*60 + seconds
	if total < 0 {
		total = 0
	}
	h = total / 3600
	rem := total % 3600
	m = rem / 60
	s = rem % 60
	return h, m, s
}
