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

**One struct, `Invoice`** — `Issue()`/`Consult()`/`Cancel()`/`Reissue()`, nothing else to instantiate. Each line item is a `br.Product` — `Description`/`Amount` apply to any document type, `NCM`/`CFOP` (plus everything else on `Product`: `CEST`, tax groups, presumed credits...) are Brazil-specific and required per item for NFE, ignored for NFSE.

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
		RecipientAddress: &stackin.Address{
			Street:       "Avenida Atlantica",
			Number:       "500",
			Neighborhood: "Copacabana",
			City:         "Rio de Janeiro",
			State:        "RJ",
			ZipCode:      "22010000",
			CityCode:     "3304557",
		},
	})
	_ = status
	fmt.Println(invoice, err)
}
```

`RecipientAddress` is an `Address` — the buyer's address, **required for NFE** and ignored for NFSE. Every field is required, `CityCode` (the 7-digit IBGE municipality code) included: it becomes `enderDest` on the wire and the SEFAZ rejects a partial one. `State` is also what resolves `idDest` — a buyer in another state is emitted as an interstate operation automatically. A missing or incomplete address returns an `*InvoiceError` locally, before the request goes out.

## Retrying safely

Issuing is the one call you must not repeat blindly. If the response is lost — a
timeout, a dropped connection — the document may well have been authorized, and a
second attempt issues a **second** fiscal document: another credit, another number
burned, and undoing it means cancelling, which has a deadline.

Pass an idempotency key to make the retry safe:

```go
key := uuid.NewString()

result, err := inv.Issue(stackin.IssueRequest{
	DocumentType:   stackin.NFSE,
	ClientName:     "Maria Silva",
	TaxID:          "12345678909",
	Items:          []br.Product{{Description: "Consultoria", Amount: 1500.00}},
	IdempotencyKey: key,
})
```

`Reissue` takes it as a call option instead, since it has no request struct:

```go
result, err := inv.Reissue("<invoice-id>", stackin.WithIdempotencyKey(key))
```

Retry with the **same key and the same body** and you get the first response back,
replayed — no second document, no credit consumed. Reissue takes the same argument.

| Situation | What the API does |
|---|---|
| New key | issues normally, records the response |
| Same key, same body | replays the recorded response |
| Same key, different body | API error 422 |
| Same key, first call still running | API error 409 |
| Previous attempt failed | key is released — the retry issues |
| Key older than 24 hours | treated as new |

Generate the key yourself and keep it for as long as you might retry — one UUID per
business event, not per HTTP call. The SDK never generates one, because a key minted
per call would protect nothing, and because two genuinely separate invoices for the
same customer and amount on the same day are a normal thing to issue.

## Correcting a document

Some mistakes don't need a cancellation. A wrong product name, wrong
transport details, a typo in the extra information — a **CC-e** (carta de
correção) fixes those, and it is free: no new credit, no burned series
number, no reissue.

```go
result, err := inv.Correct(
	"35240912345678000199550010000000011000000017",
	stackin.NFE,
	"Transportadora corrigida para Rapido Ltda",
)
```

The correction text is 15 to 1000 characters, checked locally before the call.

What a CC-e **cannot** fix: anything that changes the tax owed (base, rate,
price, quantity, totals), the buyer or the seller, or the issue date. Those
still mean cancelling and reissuing. The API sends the legally fixed wording
that says exactly this, attached to every correction.

The original document does not change — the CC-e is an event attached to it, and
the authorized XML stays as it was. A document accepts at most 20 of them, and
they are numbered for you.

**NF-e only.** NFS-e has no correction letter, and asking for one returns
a `409`.

## Invalidating unused numbers

NF-e numbering is sequential and the SEFAZ expects it to have no gaps. A number
gets reserved the moment issuing starts, so a submission that fails afterwards —
a rejection, a timeout — leaves a hole in the series. Reporting that range is how
you close it.

```go
result, err := inv.Invalidate(stackin.InvalidationRequest{
	Series:      "1",
	NumberStart: 10,
	NumberEnd:   12,
	Reason:      "Numeracao reservada e nao utilizada por falha no ERP",
})
```

The reason is 15 to 255 characters and the range is inclusive; both are checked
locally, as is `number_end` not being below `number_start`.

A number that already reached the authorizer can't be invalidated. The API checks
its own records first and answers `409` naming the offending numbers, without a
round trip — and the authorizer checks again for what we can't see from here.

**NF-e only**, and it takes no access key: there is no document to point at.

## Errors

- `*stackin.APIError` — the API responded with a non-2xx status (`StatusCode`, `Detail`) — a 401 here means `api_key` is missing, wrong, or was rotated.
- `*stackin.ConnectionFailedError` — the API didn't respond (network/DNS/timeout).
- `*stackin.InvoiceError` — `Issue()`'s `Items` is empty, missing `NCM`/`CFOP` on an item for NFE, or a missing/incomplete `RecipientAddress` on NFE.

Building the full fiscal document (issuer data, service code, tax groups, schema-accurate XML) is the API's job — configured once per company, not passed on every call.

## Examples

Runnable end-to-end programs in [`examples/nfe/`](examples/nfe/) and [`examples/nfse/`](examples/nfse/) — one program per field variant, from the bare minimum to every field filled. `examples/consult_invoice/`, `examples/cancel_invoice/`, and `examples/reissue_invoice/` cover the operations that act on an already-issued document.
