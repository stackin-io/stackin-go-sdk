package stackin

import (
	"fmt"
	"strings"
)

type Address struct {
	State        string `json:"state,omitempty"`
	CityCode     string `json:"city_code,omitempty"`
	Street       string `json:"street,omitempty"`
	Number       string `json:"number,omitempty"`
	Neighborhood string `json:"neighborhood,omitempty"`
	City         string `json:"city,omitempty"`
	ZipCode      string `json:"zip_code,omitempty"`
}

// validate checks the fields the NF-e 4.00 schema bounds, whatever the
// document type: xLgr and xBairro take at least 2 characters when sent.
func (a Address) validate() error {
	fields := []struct {
		name  string
		value string
	}{
		{"street", a.Street},
		{"neighborhood", a.Neighborhood},
	}
	for _, field := range fields {
		if field.value != "" && len([]rune(strings.TrimSpace(field.value))) < 2 {
			return fmt.Errorf("%s must have at least 2 characters", field.name)
		}
	}
	return nil
}
