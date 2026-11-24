package stackin

import (
	"fmt"
	"regexp"
	"strings"
)

var countryISO2Pattern = regexp.MustCompile(`^[A-Z]{2}$`)

// AbroadState is the State of a buyer abroad on an NF-e. Such an address
// may leave ZipCode and CityCode out and names its country with
// CountryCode (BACEN code) and CountryName instead.
const AbroadState = "EX"

type Address struct {
	State        string `json:"state,omitempty"`
	CityCode     string `json:"city_code,omitempty"`
	Street       string `json:"street,omitempty"`
	Number       string `json:"number,omitempty"`
	Neighborhood string `json:"neighborhood,omitempty"`
	City         string `json:"city,omitempty"`
	ZipCode      string `json:"zip_code,omitempty"`
	CountryCode  string `json:"country_code,omitempty"`
	CountryName  string `json:"country_name,omitempty"`
	// Complement is the apartment, suite or block, written as xCpl. A
	// complement too long for the layout is left out by the API with the
	// warning RECIPIENT_COMPLEMENT_OMITTED.
	Complement string `json:"complement,omitempty"`
	// CountryISO2 is the ISO 3166 alpha-2 code of an NFS-e tomador abroad
	// (State EX), never BR; PostalCode and Province complete its address.
	// NFS-e only.
	CountryISO2 string `json:"country_iso2,omitempty"`
	PostalCode  string `json:"postal_code,omitempty"`
	Province    string `json:"province,omitempty"`
}

func (a *Address) abroad() bool {
	return a != nil && strings.EqualFold(strings.TrimSpace(a.State), AbroadState)
}

// Export is where the goods of an NF-e to a buyer abroad leave Brazil (the
// exporta group): ExitState is a Brazilian state (UFSaidaPais, never EX),
// Place the place of shipment or border crossing (xLocExporta) and
// DispatchPlace the optional customs facility of dispatch (xLocDespacho).
type Export struct {
	ExitState     string `json:"exit_state"`
	Place         string `json:"place"`
	DispatchPlace string `json:"dispatch_place,omitempty"`
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
	if a.CountryISO2 != "" && !countryISO2Pattern.MatchString(a.CountryISO2) {
		return fmt.Errorf("country_iso2 must be an ISO 3166 alpha-2 code")
	}
	if len([]rune(a.PostalCode)) > 11 {
		return fmt.Errorf("postal_code must be at most 11 characters")
	}
	if len([]rune(a.Province)) > 60 {
		return fmt.Errorf("province must be at most 60 characters")
	}
	return nil
}
