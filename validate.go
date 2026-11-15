package stackin

import (
	"errors"
	"regexp"

	"github.com/stackin-io/stackin-go-sdk/br"
)

var gtinPattern = regexp.MustCompile(`^(\d{8}|\d{12,14})$`)

// validateProduct checks the item fields whose shape is fixed by the NF-e
// 4.00 schema, whatever the document type. A Barcode (cEAN) holding a
// GTIN-8, -12, -13 or -14 must carry a correct GS1 check digit; "SEM GTIN"
// skips the check. Issue calls it for every item and returns the failure
// as an *InvoiceError naming the item.
func validateProduct(p br.Product) error {
	if p.Barcode != nil && gtinPattern.MatchString(*p.Barcode) && !validGTINCheckDigit(*p.Barcode) {
		return errors.New("barcode has an invalid GTIN check digit")
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
