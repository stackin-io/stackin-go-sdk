package br

import "testing"

func TestValidateAcceptsAnEightDigitNCMOrTheServiceCode(t *testing.T) {
	for _, ncm := range []string{"84713012", "00"} {
		if err := (Product{Description: "Notebook", NCM: ptr(ncm)}).Validate(); err != nil {
			t.Errorf("Validate(ncm=%q) = %v, want nil", ncm, err)
		}
	}
}

func TestValidateAcceptsAProductWithoutNCM(t *testing.T) {
	if err := (Product{Description: "Consultoria"}).Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestValidateRejectsAnyOtherNCM(t *testing.T) {
	for _, ncm := range []string{"84", "12", "000", "8471", "8471301", "847130120", "8471301A", " 84713012"} {
		err := (Product{Description: "Notebook", NCM: ptr(ncm)}).Validate()
		if err == nil || err.Error() != "ncm must be the full 8 digits, or 00 for a service line" {
			t.Errorf("Validate(ncm=%q) = %v, want the 8-digit error", ncm, err)
		}
	}
}
