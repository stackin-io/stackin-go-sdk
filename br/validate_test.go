package br

import "testing"

// gs1CheckDigit computes the GS1 modulo-10 check digit of body
// independently of the code under test, to build vectors in the test.
func gs1CheckDigit(body string) byte {
	sum := 0
	weight := 3
	for i := len(body) - 1; i >= 0; i-- {
		sum += int(body[i]-'0') * weight
		weight = 4 - weight
	}
	return byte('0' + (10-sum%10)%10)
}

func TestGS1CheckDigitHelperMatchesPublishedVectors(t *testing.T) {
	for _, gtin := range []string{"4006381333931", "96385074", "036000291452", "7891000100103"} {
		body := gtin[:len(gtin)-1]
		if got := gs1CheckDigit(body); got != gtin[len(gtin)-1] {
			t.Errorf("gs1CheckDigit(%q) = %c, want %c", body, got, gtin[len(gtin)-1])
		}
	}
}

func TestValidateAcceptsAValidGTINCheckDigit(t *testing.T) {
	gtins := []string{"4006381333931", "96385074", "036000291452", "7891000100103"}
	gtin14 := "1789123456789"
	gtins = append(gtins, gtin14+string(gs1CheckDigit(gtin14)))
	for _, gtin := range gtins {
		if err := (Product{Description: "Item", Barcode: ptr(gtin)}).Validate(); err != nil {
			t.Errorf("Validate(barcode=%q) = %v, want nil", gtin, err)
		}
	}
}

func TestValidateRejectsAWrongGTINCheckDigit(t *testing.T) {
	// 17891234567894 is the issue's example; its check digit is 2.
	for _, gtin := range []string{"17891234567894", "4006381333932", "96385075", "036000291453", "7891000100100"} {
		err := (Product{Description: "Item", Barcode: ptr(gtin)}).Validate()
		if err == nil || err.Error() != "barcode has an invalid GTIN check digit" {
			t.Errorf("Validate(barcode=%q) = %v, want the check digit error", gtin, err)
		}
	}
}

func TestValidateSkipsTheCheckDigitOnSemGTIN(t *testing.T) {
	if err := (Product{Description: "Item", Barcode: ptr("SEM GTIN")}).Validate(); err != nil {
		t.Errorf("Validate(SEM GTIN) = %v, want nil", err)
	}
}
