package stackin

import (
	"errors"
	"regexp"

	"github.com/stackin-io/stackin-go-sdk/br"
)

var (
	ncmPattern     = regexp.MustCompile(`^(00|\d{8})$`)
	barcodePattern = regexp.MustCompile(`^(SEM GTIN|\d{8}|\d{12,14})$`)
)

// validateProduct checks the item fields whose shape is fixed by the NF-e
// 4.00 schema, whatever the document type. NCM takes the full 8 digits, or
// 00 for a service line; the SEFAZ rejects any other short code with 777.
// Barcode (cEAN) is either "SEM GTIN" or a GTIN-8, -12, -13 or -14 whose
// GS1 check digit holds. Issue calls it for every item and returns the
// failure as an *InvoiceError naming the item.
func validateProduct(p br.Product) error {
	if p.NCM != nil && *p.NCM != "" && !ncmPattern.MatchString(*p.NCM) {
		return errors.New("ncm must be the full 8 digits, or 00 for a service line")
	}
	if p.Barcode != nil {
		if !barcodePattern.MatchString(*p.Barcode) {
			return errors.New("barcode must be 'SEM GTIN' or a GTIN of 8, 12, 13 or 14 digits")
		}
		if *p.Barcode != "SEM GTIN" && !validGTINCheckDigit(*p.Barcode) {
			return errors.New("barcode has an invalid GTIN check digit")
		}
	}
	return nil
}

// validGTINCheckDigit applies the GS1 modulo-10 check: from the rightmost
// digit before the check digit, weights alternate 3, 1, 3, 1...
func validGTINCheckDigit(gtin string) bool {
	last := len(gtin) - 1
	sum := 0
	for i := last - 1; i >= 0; i-- {
		digit := int(gtin[i] - '0')
		if (last-1-i)%2 == 0 {
			digit *= 3
		}
		sum += digit
	}
	return (10-sum%10)%10 == int(gtin[last]-'0')
}
