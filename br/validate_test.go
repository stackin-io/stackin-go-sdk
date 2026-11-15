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

func TestValidateAcceptsWellFormedBarcodes(t *testing.T) {
	for _, barcode := range []string{"SEM GTIN", "96385074", "036000291452", "4006381333931", "17891000100100"} {
		if err := (Product{Description: "Item", Barcode: ptr(barcode)}).Validate(); err != nil {
			t.Errorf("Validate(barcode=%q) = %v, want nil", barcode, err)
		}
	}
}

func TestValidateRejectsMalformedBarcodes(t *testing.T) {
	for _, barcode := range []string{"", "SEM", "sem gtin", "1234567", "123456789", "12345678901", "123456789012345", "40063813339A1"} {
		err := (Product{Description: "Item", Barcode: ptr(barcode)}).Validate()
		want := "barcode must be 'SEM GTIN' or a GTIN of 8, 12, 13 or 14 digits"
		if err == nil || err.Error() != want {
			t.Errorf("Validate(barcode=%q) = %v, want %q", barcode, err, want)
		}
	}
}
