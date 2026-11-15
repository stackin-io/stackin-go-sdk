package stackin

import (
	"errors"
	"regexp"

	"github.com/stackin-io/stackin-go-sdk/br"
)

var barcodePattern = regexp.MustCompile(`^(SEM GTIN|\d{8}|\d{12,14})$`)

// validateProduct checks the item fields whose shape is fixed by the NF-e
// 4.00 schema, whatever the document type. Barcode (cEAN) is either
// "SEM GTIN" or a GTIN-8, -12, -13 or -14. Issue calls it for every item
// and returns the failure as an *InvoiceError naming the item.
func validateProduct(p br.Product) error {
	if p.Barcode != nil && !barcodePattern.MatchString(*p.Barcode) {
		return errors.New("barcode must be 'SEM GTIN' or a GTIN of 8, 12, 13 or 14 digits")
	}
	return nil
}
