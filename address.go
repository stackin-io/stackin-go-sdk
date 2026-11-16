package stackin

import (
	"fmt"
	"strings"
)

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
	return nil
}
