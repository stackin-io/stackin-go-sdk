package stackin

type Address struct {
	State        string `json:"state,omitempty"`
	CityCode     string `json:"city_code,omitempty"`
	Street       string `json:"street,omitempty"`
	Number       string `json:"number,omitempty"`
	Neighborhood string `json:"neighborhood,omitempty"`
	City         string `json:"city,omitempty"`
	ZipCode      string `json:"zip_code,omitempty"`
}
