package main

import (
	"strings"
	"testing"
)

func TestConvertTimeValid(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Valid test case with UTC timezone",
			input: "2024-01-01T00:00:00Z",
			want:  "1704067200000",
		},
		{
			name:  "Valid test case with timezone offset",
			input: "2025-08-20T19:59:23+00:00",
			want:  "1755719963000",
		},
		{
			name:  "Valid test case with nanoseconds",
			input: "2024-01-01T00:00:00.000000000Z",
			want:  "1704067200000",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := convertTime(test.input)
			if err != nil {
				t.Fatalf("convertTime(%q) returned an unexpected error: %v", test.input, err)
			}
			if got != test.want {
				t.Errorf("convertTime(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestConvertTimeInvalid(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError string
	}{
		{
			name:      "Invalid time string",
			input:     "not-a-time",
			wantError: `cannot parse "not-a-time" as "2006"`,
		},
		{
			name:      "Invalid date separator",
			input:     "2024/01/01T00:00:00Z",
			wantError: `cannot parse "/01/01T00:00:00Z" as "-"`,
		},
		{
			name:      "Month out of range",
			input:     "2024-13-01T00:00:00Z",
			wantError: `month out of range`,
		},
		{
			name:      "Day out of range",
			input:     "2024-02-30T00:00:00Z",
			wantError: `day out of range`,
		},
		{
			name:      "Hour out of range",
			input:     "2024-01-01T25:00:00Z",
			wantError: `hour out of range`,
		},
		{
			name:      "Minute out of range",
			input:     "2024-01-01T00:61:00Z",
			wantError: `minute out of range`,
		},
		{
			name:      "Second out of range",
			input:     "2024-01-01T00:00:61Z",
			wantError: `second out of range`,
		},
		{
			name:      "Missing T separator",
			input:     "2024-01-01 00:00:00Z",
			wantError: `cannot parse " 00:00:00Z" as "T"`,
		},
		{
			name:      "Missing timezone information",
			input:     "2024-01-01T00:00:00",
			wantError: `cannot parse "" as "Z07:00"`,
		},
		{
			name:      "Invalid timezone offset format",
			input:     "2024-01-01T00:00:00+00",
			wantError: `cannot parse "+00" as "Z07:00"`,
		},
		{
			name:      "Missing seconds",
			input:     "2024-01-01T00:00Z",
			wantError: `cannot parse "Z" as ":"`,
		},
		{
			name:      "Extra characters",
			input:     "2024-01-01T00:00:00Z-extra",
			wantError: `extra text: "-extra"`,
		},
		{
			name:      "Empty string",
			input:     "",
			wantError: `time string is empty`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := convertTime(test.input)
			if err == nil {
				t.Fatalf("convertTime(%q) returned no error, want error", test.input)
			}
			if !strings.Contains(err.Error(), test.wantError) {
				t.Errorf("convertTime(%q) returned error %q, want error %q", test.input, err, test.wantError)
			}
		})
	}

}
