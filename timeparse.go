package stackin

import (
	"encoding/json"
	"time"
)

var apiTimeLayouts = []string{
	time.RFC3339Nano,
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999",
}

func parseAPITime(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	for _, layout := range apiTimeLayouts {
		if at, err := time.Parse(layout, value); err == nil {
			return at, true
		}
	}
	return time.Time{}, false
}

type apiTime struct {
	value *time.Time
}

func (t *apiTime) UnmarshalJSON(raw []byte) error {
	var text *string
	if err := json.Unmarshal(raw, &text); err != nil {
		return err
	}
	if text == nil || *text == "" {
		t.value = nil
		return nil
	}
	at, ok := parseAPITime(*text)
	if !ok {
		return &time.ParseError{Layout: time.RFC3339, Value: *text, Message: ": not a date-time"}
	}
	t.value = &at
	return nil
}
