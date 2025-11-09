package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMonthsInclusive(t *testing.T) {
	tests := []struct {
		name     string
		start    string
		end      string
		expected int
	}{
		{
			name:     "same day",
			start:    "2025-01-01",
			end:      "2025-01-01",
			expected: 1,
		},
		{
			name:     "less than 30 days",
			start:    "2025-01-01",
			end:      "2025-01-29",
			expected: 1,
		},
		{
			name:     "exactly 30 days",
			start:    "2025-01-01",
			end:      "2025-01-30",
			expected: 1,
		},
		{
			name:     "31 days (should count as 2 months)",
			start:    "2025-01-01",
			end:      "2025-01-31",
			expected: 2,
		},
		{
			name:     "60 days (2 months)",
			start:    "2025-01-01",
			end:      "2025-03-01",
			expected: 2,
		},
		{
			name:     "90 days (3 months)",
			start:    "2025-01-01",
			end:      "2025-03-31",
			expected: 3,
		},
		{
			name:     "end before start",
			start:    "2025-03-01",
			end:      "2025-01-01",
			expected: 0,
		},
	}

	layout := "2006-01-02"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, _ := time.Parse(layout, tt.start)
			b, _ := time.Parse(layout, tt.end)
			got := monthsInclusive(a, b)
			assert.Equal(t, tt.expected, got)
		})
	}
}
