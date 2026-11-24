package stackin

import (
	"errors"
	"regexp"
	"strings"

	"github.com/stackin-io/stackin-go-sdk/br"
)

var (
	ncmPattern     = regexp.MustCompile(`^(00|\d{8})$`)
	barcodePattern = regexp.MustCompile(`^(SEM GTIN|\d{8}|\d{12,14})$`)
	recopiPattern  = regexp.MustCompile(`^\d{20}$`)

	municipalServiceCodePattern = regexp.MustCompile(`^\d{3}$`)
	internalCodePattern         = regexp.MustCompile(`^[A-Za-z0-9]{1,20}$`)
)

// validateProduct checks the item fields whose shape is fixed by the NF-e
// 4.00 schema, whatever the document type. NCM takes the full 8 digits, or
// 00 for a service line; the SEFAZ rejects any other short code with 777.
// Barcode (cEAN) is either "SEM GTIN" or a GTIN-8, -12, -13 or -14 whose
// GS1 check digit holds. RecopiNumber (nRECOPI) is 20 digits whose two
// modulo-11 check digits (MOC 7.0 §8.5) hold. Fuel needs its three
// required fields and is never sent beside ExtraGroups["comb"], the same
// group. Issue calls it for every item and returns the failure as an
// *InvoiceError naming the item.
func validateProduct(p br.Product) error {
	if p.NetWeight != nil && *p.NetWeight < 0 {
		return errors.New("net_weight can't be negative")
	}
	if p.GrossWeight != nil && *p.GrossWeight < 0 {
		return errors.New("gross_weight can't be negative")
	}
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
	if p.RecopiNumber != nil {
		if !recopiPattern.MatchString(*p.RecopiNumber) {
			return errors.New("recopi_number must have exactly 20 digits")
		}
		if !validRecopiCheckDigits(*p.RecopiNumber) {
			return errors.New("recopi_number has invalid check digits")
		}
	}
	if p.Fuel != nil {
		if err := validateFuel(*p.Fuel, p.ExtraGroups); err != nil {
			return err
		}
	}
	if p.MunicipalServiceCode != "" && !municipalServiceCodePattern.MatchString(p.MunicipalServiceCode) {
		return errors.New("municipal_service_code must be 3 digits")
	}
	if p.InternalCode != "" && !internalCodePattern.MatchString(p.InternalCode) {
		return errors.New("internal_code must be 1 to 20 letters or digits")
	}
	if p.ConditionalDiscount != nil {
		value := *p.ConditionalDiscount
		if value <= 0 || value > 999999999999999.99 || decimalPlaces(value) > 2 {
			return errors.New("conditional_discount must be above 0 with at most 2 decimal places")
		}
	}
	if p.IbsCbs != nil {
		if err := p.IbsCbs.Validate("product.br.ibs_cbs"); err != nil {
			return err
		}
	}
	return nil
}

func validateFuel(fuel br.Fuel, extraGroups map[string]any) error {
	if _, ok := extraGroups["comb"]; ok {
		return errors.New("fuel can't be sent together with extra_groups.comb: send the fuel group once")
	}
	required := []struct {
		name  string
		value string
	}{
		{"anp_code", fuel.AnpCode},
		{"anp_description", fuel.AnpDescription},
		{"consumer_state", fuel.ConsumerState},
	}
	for _, field := range required {
		if field.value == "" {
			return errors.New("fuel." + field.name + " is required")
		}
	}
	return nil
}

// validRecopiCheckDigits applies the RECOPI modulo-11 check (MOC 7.0
// §8.5): DV1 over the first 18 digits with weights 18 down to 1, DV2 over
// the first 19 (18 + DV1) with weights 19 down to 1. A remainder of 0 or 1
// gives 0, any other r gives 11 - r. The date part is not checked.
func validRecopiCheckDigits(recopi string) bool {
	for n := 18; n <= 19; n++ {
		sum := 0
		for i := 0; i < n; i++ {
			sum += int(recopi[i]-'0') * (n - i)
		}
		dv := 0
		if r := sum % 11; r > 1 {
			dv = 11 - r
		}
		if int(recopi[n]-'0') != dv {
			return false
		}
	}
	return true
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

var registrationPattern = regexp.MustCompile(`^[A-Za-z0-9]{1,15}$`)

func unmaskedRegistration(value string) string {
	return strings.NewReplacer(" ", "", ".", "", "/", "", "-", "").Replace(strings.TrimSpace(value))
}

func validateRecipientFields(req IssueRequest) error {
	if req.RecipientMunicipalRegistration != "" &&
		!registrationPattern.MatchString(unmaskedRegistration(req.RecipientMunicipalRegistration)) {
		return &InvoiceError{Message: "recipient_municipal_registration takes up to 15 letters or digits"}
	}
	if req.ForeignIDMissingReason != nil && *req.ForeignIDMissingReason != 1 && *req.ForeignIDMissingReason != 2 {
		return &InvoiceError{Message: "foreign_id_missing_reason takes 1 (exempt from a NIF) or 2 (the country does not require one); 0 is refused"}
	}
	return nil
}
