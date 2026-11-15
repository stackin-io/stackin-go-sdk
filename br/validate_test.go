package br

import "testing"

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
