package main

import (
	"testing"
	"time"
)

func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "valid timestamp",
			in:   "1678886400000",
			want: time.UnixMilli(1678886400000).Format(time.RFC1123),
		},
		{
			name: "invalid timestamp",
			in:   "not a number",
			want: "not a number",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatTimestamp(tc.in); got != tc.want {
				t.Errorf("formatTimestamp(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
