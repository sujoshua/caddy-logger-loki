package caddy_logger_loki

import (
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type StrTimeDuration struct {
	Raw string
	T   time.Duration
}

func (t *StrTimeDuration) FromString(raw string) error {
	re := regexp.MustCompile(`(\d+)\s*(ms|s|m|h|d|w)`)
	matches := re.FindAllStringSubmatch(strings.ToLower(raw), -1)

	if len(matches) == 0 {
		return errors.New("invalid time duration format")
	}

	// Map of time units to time.Duration multipliers
	unitMap := map[string]time.Duration{
		"ms": time.Millisecond,
		"s":  time.Second,
		"m":  time.Minute,
		"h":  time.Hour,
		"d":  24 * time.Hour,     // 1 day
		"w":  7 * 24 * time.Hour, // 1 week
	}

	for _, match := range matches {
		value := match[1]
		unit := match[2]

		v, err := strconv.Atoi(value)
		if err != nil {
			return errors.New("invalid time duration format, unable to parse time value, get:" + value)
		}

		multiplier, exists := unitMap[unit]
		if !exists {
			return errors.New("invalid time unit " + unit + ", valid units are: s, m, h, d, w")
		}

		t.T += time.Duration(v) * multiplier
	}

	return nil
}

func (t *StrTimeDuration) UnmarshalJSON(data []byte) error {
	return t.FromString(string(data))
}

func (t *StrTimeDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.T)
}

func (t *StrTimeDuration) TimeDuration() time.Duration {
	return t.T
}
