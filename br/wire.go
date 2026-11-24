package br

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	amountMax = 999999999999999.99
	dayLayout = "2006-01-02"
)

var (
	digits7        = regexp.MustCompile(`^\d{7}$`)
	digits8        = regexp.MustCompile(`^\d{8}$`)
	upperISO2      = regexp.MustCompile(`^[A-Z]{2}$`)
	maskCharacters = strings.NewReplacer(" ", "", ".", "", "/", "", "-", "")
)

type wire struct {
	path string
	out  map[string]any
	err  error
}

func newWire(path string) *wire {
	return &wire{path: path, out: map[string]any{}}
}

func (w *wire) at(key string) string {
	switch {
	case key == "":
		return w.path
	case w.path == "":
		return key
	}
	return w.path + "." + key
}

func (w *wire) fail(key, format string, args ...any) {
	if w.err == nil {
		w.err = fmt.Errorf("%s %s", w.at(key), fmt.Sprintf(format, args...))
	}
}

func (w *wire) failWhole(format string, args ...any) {
	if w.err == nil {
		w.err = fmt.Errorf("%s %s", w.path, fmt.Sprintf(format, args...))
	}
}

func (w *wire) text(key, value string, min, max int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if n := utf8.RuneCountInString(value); n < min || n > max {
		w.fail(key, "must be %d to %d characters", min, max)
		return ""
	}
	w.out[key] = value
	return value
}

func (w *wire) required(key, value string, min, max int) string {
	if strings.TrimSpace(value) == "" {
		w.fail(key, "is required")
		return ""
	}
	return w.text(key, value, min, max)
}

func (w *wire) pattern(key, value string, re *regexp.Regexp, want string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !re.MatchString(value) {
		w.fail(key, "must be %s", want)
		return ""
	}
	w.out[key] = value
	return value
}

func (w *wire) requiredPattern(key, value string, re *regexp.Regexp, want string) string {
	if strings.TrimSpace(value) == "" {
		w.fail(key, "is required")
		return ""
	}
	return w.pattern(key, value, re, want)
}

func (w *wire) choice(key, value string, allowed ...string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, option := range allowed {
		if value == option {
			w.out[key] = value
			return value
		}
	}
	w.fail(key, "must be one of %s, not %q", strings.Join(allowed, ", "), value)
	return ""
}

func (w *wire) requiredChoice(key, value string, allowed ...string) string {
	if strings.TrimSpace(value) == "" {
		w.fail(key, "is required")
		return ""
	}
	return w.choice(key, value, allowed...)
}

func (w *wire) integer(key string, value *int, allowed ...int) {
	if value == nil {
		return
	}
	for _, option := range allowed {
		if *value == option {
			w.out[key] = *value
			return
		}
	}
	w.fail(key, "must be one of %s, not %d", joinInts(allowed), *value)
}

func (w *wire) requiredInteger(key string, value int, allowed ...int) {
	w.integer(key, &value, allowed...)
}

func joinInts(values []int) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = fmt.Sprint(value)
	}
	return strings.Join(parts, ", ")
}

func (w *wire) flag(key string, value *bool) {
	if value != nil {
		w.out[key] = *value
	}
}

type bounds struct {
	lo, hi float64
	loOpen bool
	places int
}

func (b bounds) describe() string {
	floor := fmt.Sprintf("at least %s", DecimalString(b.lo))
	if b.loOpen {
		floor = fmt.Sprintf("above %s", DecimalString(b.lo))
	}
	text := fmt.Sprintf("%s and at most %s", floor, DecimalString(b.hi))
	if b.places > 0 {
		text += fmt.Sprintf(" with at most %d decimal places", b.places)
	}
	return text
}

func (w *wire) check(key string, value float64, b bounds) bool {
	below := value < b.lo || (b.loOpen && value == b.lo)
	if below || value > b.hi || (b.places > 0 && decimalPlaces(value) > b.places) {
		w.fail(key, "must be %s", b.describe())
		return false
	}
	return true
}

func (w *wire) decimal(key string, value *float64, b bounds) {
	if value != nil && w.check(key, *value, b) {
		w.out[key] = DecimalString(*value)
	}
}

func (w *wire) requiredDecimal(key string, value float64, b bounds) {
	if w.check(key, value, b) {
		w.out[key] = DecimalString(value)
	}
}

func (w *wire) number(key string, value *float64, b bounds) {
	if value != nil && w.check(key, *value, b) {
		w.out[key] = *value
	}
}

func (w *wire) requiredNumber(key string, value float64, b bounds) {
	if w.check(key, value, b) {
		w.out[key] = value
	}
}

func (w *wire) day(key string, value time.Time) bool {
	if value.IsZero() {
		return false
	}
	if year := value.Year(); year < 2000 || year > 2099 {
		w.fail(key, "must be a date from 2000 to 2099")
		return false
	}
	w.out[key] = value.Format(dayLayout)
	return true
}

func (w *wire) requiredDay(key string, value time.Time) {
	if value.IsZero() {
		w.fail(key, "is required")
		return
	}
	w.day(key, value)
}

func (w *wire) adopt(key string, body map[string]any, err error) {
	if err != nil {
		if w.err == nil {
			w.err = err
		}
		return
	}
	w.out[key] = body
}

func (w *wire) result() (map[string]any, error) {
	if w.err != nil {
		return nil, w.err
	}
	return w.out, nil
}

func decimalPlaces(value float64) int {
	text := DecimalString(value)
	dot := strings.IndexByte(text, '.')
	if dot < 0 {
		return 0
	}
	return len(text) - dot - 1
}

func unmasked(value string) string {
	return maskCharacters.Replace(strings.TrimSpace(value))
}

func presentCount(flags ...bool) int {
	count := 0
	for _, flag := range flags {
		if flag {
			count++
		}
	}
	return count
}

func has(value string) bool {
	return strings.TrimSpace(value) != ""
}
