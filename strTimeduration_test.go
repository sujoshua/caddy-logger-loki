package caddy_logger_loki

import (
	"encoding/json"
	"testing"
	"time"
)

func TestStrTimeDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"2h30m", 2*time.Hour + 30*time.Minute, false},
		{"1d", 24 * time.Hour, false},
		{"1w", 7 * 24 * time.Hour, false},
		{"45s", 45 * time.Second, false},
		{"100ms", 100 * time.Millisecond, false},
		{"2h30m500ms", 2*time.Hour + 30*time.Minute + 500*time.Millisecond, false},
		{"invalid", 0, true},
		{"5y", 0, true},                 // Invalid unit 'y'
		{"2h60m", 3 * time.Hour, false}, // 60m should be parsed as 1 hour
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			st := StrTimeDuration{}
			err := st.FromString(test.input)

			if test.hasError {
				if err == nil {
					t.Fatalf("expected error for input %q, got nil", test.input)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error for input %q: %v", test.input, err)
				}
				if st.TimeDuration() != test.expected {
					t.Fatalf("for input %q, expected %v, got %v", test.input, test.expected, st.TimeDuration())
				}
			}
		})
	}

	// Test UnmarshalJson
	jsonTests := []struct {
		input    string
		expected time.Duration
	}{
		{`{"time": "2h30m"}`, 2*time.Hour + 30*time.Minute},
		{`{"time": "1d"}`, 24 * time.Hour},
		{`{"time": "1w"}`, 7 * 24 * time.Hour},
		{`{"time": "45s"}`, 45 * time.Second},
		{`{"time": "100ms"}`, 100 * time.Millisecond},
		{`{"time": "2h30m500ms"}`, 2*time.Hour + 30*time.Minute + 500*time.Millisecond},
		{`{"time": "invalid"}`, -1},          // -1 means error
		{`{"time": "5y"}`, -1},               // Invalid unit 'y'
		{`{"time": "2h60m"}`, 3 * time.Hour}, // 60m should be parsed as 1 hour
		{`{"no_time": "114514"}`, 0},         // No time field
	}

	for _, test := range jsonTests {
		t.Run(test.input, func(t *testing.T) {
			st := struct {
				Time StrTimeDuration `json:"time"`
			}{}
			err := json.Unmarshal([]byte(test.input), &st)

			if err != nil && test.expected != -1 {
				t.Fatalf("unexpected error for input %q: %v", test.input, err)
			}
			if test.expected != -1 && st.Time.TimeDuration() != test.expected {
				t.Fatalf("for input %q, expected %v, got %v", test.input, test.expected, st.Time.TimeDuration())
			}
		})
	}
}
