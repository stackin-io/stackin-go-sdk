<div align="center">

<img src="https://raw.githubusercontent.com/stackin-io/stackin-go-sdk/main/docs/assets/stackin.png" width="120" />

**Integrate once. Issue everywhere.**

[![Go](https://img.shields.io/badge/go-1.21%2B-blue?style=flat-square)](go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/stackin-io/stackin-go-sdk.svg)](https://pkg.go.dev/github.com/stackin-io/stackin-go-sdk)
[![License](https://img.shields.io/badge/license-MIT-informational?style=flat-square)](https://github.com/stackin-io/stackin-go-sdk)

[API Reference](https://docs.stackin.io) · [Go SDK guide](https://docs.stackin.io/blog/go-sdk)

</div>

---

# stackin

Go SDK for issuing, consulting and cancelling electronic invoices — a handful of business fields, nothing about certificates, XML, XSD, signing or SOAP. The API resolves all of that from the issuer's own configuration, identified by `api_key`.

**One struct, `Invoice`** — `Issue()`/`Consult()`/`Cancel()`, nothing else to instantiate. Each line item is a `br.Product` — `Description`/`Amount` apply to any document type, `NCM`/`CFOP` (plus everything else on `Product`: `CEST`, tax groups, presumed credits...) are Brazil-specific and required per item for NFE, ignored for NFSE.

## Install

```bash
go get github.com/stackin-io/stackin-go-sdk
```

## Usage

Get an `api_key` from the [stackin dashboard](https://app.stackin.io) — select the issuing company, then Settings → API key (context `sdk`). One key per issuing company, shown once at creation. The API resolves the issuer (CNPJ, state, address, certificate, environment) entirely from it; nothing about the issuer is ever passed on a call.

```go
package main

import (
	"fmt"

	stackin "github.com/stackin-io/stackin-go-sdk"
	"github.com/stackin-io/stackin-go-sdk/br"
)

func main() {
	client := stackin.NewInvoice(stackin.WithAPIKey("COMPANY_API_KEY"))

	invoice, err := client.Issue(stackin.IssueRequest{
		DocumentType: stackin.NFSE,
		ClientName:   "John Doe",
		TaxID:        "00000000000",
		Items: []br.Product{
			{Description: "Software development", Amount: 5000.00},
		},
	})

	status, err := client.Consult("ACCESS_KEY...", stackin.NFSE)
	_, err = client.Cancel("ACCESS_KEY...", stackin.NFSE, "Typo")

	ncm, cfop := "84713012", "5102"
	invoice, err = client.Issue(stackin.IssueRequest{
		DocumentType: stackin.NFE,
		ClientName:   "Buyer Company Ltd",
		TaxID:        "11111111111111",
		Items: []br.Product{
			{Description: "Test product", Amount: 100.00, NCM: &ncm, CFOP: &cfop},
		},
		RecipientAddress: &stackin.Address{State: "RJ"},
	})
	_ = status
	fmt.Println(invoice, err)
}
```

`RecipientAddress` is an `Address`, but despite the name only `.State` is read — the rest of the fields aren't sent anywhere yet. It's the actual customer's state, used only to set `idDest` (interstate vs internal) on NFE — optional, omitting it always produces `idDest=1` (internal).

## Errors

- `*stackin.APIError` — the API responded with a non-2xx status (`StatusCode`, `Detail`) — a 401 here means `api_key` is missing, wrong, or was rotated.
- `*stackin.ConnectionFailedError` — the API didn't respond (network/DNS/timeout).
- `*stackin.InvoiceError` — `Issue()`'s `Items` is empty, or missing `NCM`/`CFOP` on an item for NFE.

Building the full fiscal document (issuer data, service code, tax groups, schema-accurate XML) is the API's job — configured once per company, not passed on every call.

## Examples

Runnable end-to-end programs in [`examples/`](examples/) — `simple_issue_nfe` and `simple_issue_nfse`, each with a catalog of realistic line items covering every optional field.
