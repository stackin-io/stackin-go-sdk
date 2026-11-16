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

**One struct, `Invoice`** — `Issue()`/`Consult()`/`Cancel()`/`Reissue()`/`Correct()`/`Invalidate()`/`Pdf()`/`Received()`/`Manifest()`, nothing else to instantiate. Each line item is a `br.Product` — `Description`/`Amount` apply to any document type, `NCM`/`CFOP` (plus everything else on `Product`: `CEST`, tax groups, presumed credits...) are Brazil-specific and required per item for NFE, ignored for NFSE.

## What a line item is worth

`UnitPrice` is the price of **one unit**. `Amount` is the **gross total of
the line's products**, before discount, freight, insurance and other
expenses. Send either; sending both asserts that they agree.

```go
price := 120.00

// More than one unit — the note's line is 2 x 120.00 = 240.00
br.Product{Description: "Teclado", Quantity: 2, UnitPrice: &price, Unit: "UN"}

// Legacy: Amount alone still means the line's gross total
br.Product{Description: "Servico", Quantity: 3, Amount: 150.00}
```

Amounts that do not add up are refused before the authorizer sees them,
with a `422` naming the line and both numbers (`ITEM_TOTAL_MISMATCH`).

### Migrating

```text
Before:  Quantity: 3, Amount: 150.00
After:   Quantity: 3, UnitPrice: &price   // price := 50.00
```

Nothing has to migrate. `Amount` keeps the meaning it always had and is
not deprecated in this release.

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
			{Description: "Software development", UnitPrice: ptr(5000.00)},
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
		},
		ExternalReference: "order-1234",
	})
	// The authorizer's PDF, as raw bytes. NFS-e only; the XML stays the
	// legally valid document, and a 502 here means the authorizer is down.
	document, err := client.Pdf("ACCESS_KEY...", stackin.NFSE)
	err = os.WriteFile("nota.pdf", document, 0o644)

	_ = status
	fmt.Println(invoice, err)
}
```

`ExternalReference` is your own id for the document — an order number, a ticket. It is sent as `external_reference` (left out when empty), stored as sent (at most 255 characters), returned on every read and webhook, and `GET /invoices?external_reference=` finds the document by it.

`RecipientAddress` is an `Address` — the buyer's address, **required for NFE**. On NFE `Street`, `Number`, `Neighborhood`, `City`, `State` and `ZipCode` are required: it becomes `enderDest` on the wire and the SEFAZ rejects a partial one. `CityCode` (the 7-digit IBGE municipality code) may be left out — the API resolves it from `City` and `State`, and answers `CITY_NOT_FOUND` (422) when it cannot; send it yourself when the name is ambiguous. `State` is also what resolves `idDest` — a buyer in another state is emitted as an interstate operation automatically. On NFSE it is the tomador's address: optional, except when any item has `TaxRetained` (the tomador withholds the ISSQN), where it is required with every field except `CityCode`. A missing or incomplete NFE address, or a missing or incomplete one on an NFSE with a retained item, returns an `*InvoiceError` locally, before the request goes out.

### Discount, emission date, purpose and buyers abroad

Optional since API v0.8–v0.9. Each is left out of the body when unset, so a call that does not use them sends exactly what it sent before.

```go
issuedAt := time.Date(2026, 11, 14, 9, 30, 0, 0, time.FixedZone("BRT", -3*60*60))
discount := 25.00

invoice, err := client.Issue(stackin.IssueRequest{
	DocumentType:     stackin.NFE,
	ClientName:       "Buyer Company Ltd",
	TaxID:            "11111111111111",
	Items:            items,
	RecipientAddress: address,
	Discount:         &discount,
	IssuedAt:         &issuedAt,
	Purpose:          stackin.PurposeReturn,
	Direction:        stackin.DirectionIn,
	ReferencedKeys:   []string{"35261111111111111111550010000000011000000010"},
})
```

| Field | Sent as | What it is |
|---|---|---|
| `Discount *float64` | `discount` (decimal string) | A discount on the whole document, in BRL, zero or more. The API spreads it over the lines. |
| `IssuedAt *time.Time` | `issued_at` | The emission moment, RFC 3339 in the offset of the `time.Time`'s own `Location` (`2026-11-14T09:30:00-03:00`; UTC goes out as `Z`). NF-e: at most 30 days back and 5 minutes ahead; NFS-e: never ahead. Defaults to now. |
| `Purpose` | `purpose` | NF-e only: `PurposeNormal` (the default), `PurposeComplementary`, `PurposeAdjustment` or `PurposeReturn`. |
| `Direction` | `direction` | NF-e only: `DirectionOut` (the default) or `DirectionIn`, an incoming NF-e the company issues itself, such as a sale returned by the buyer. |
| `ReferencedKeys []string` | `referenced_keys` | NF-e only: the 44-character access keys of the NF-e, NFC-e or CF-e this one refers to. Required for `PurposeComplementary` and, but for a few CFOPs, `PurposeReturn`. |
| `ForeignID string` | `foreign_id` | The document of a buyer abroad (idEstrangeiro, 5 to 20 characters). |
| `Export *Export` | `export` | Where the goods leave Brazil: `ExitState` (a Brazilian state, never `EX`), `Place` and an optional `DispatchPlace`. Required for an outgoing NF-e to a buyer abroad. |

A buyer abroad is a `RecipientAddress` with `State: stackin.AbroadState` (`"EX"`), NF-e only. `ZipCode` and `CityCode` may be left out; `CountryCode` (the BACEN code, 1 to 4 digits, never 1058) is required and `CountryName` names it. `TaxID` is left empty and the SDK does not send it:

```go
invoice, err = client.Issue(stackin.IssueRequest{
	DocumentType: stackin.NFE,
	ClientName:   "Globex Corporation",
	ForeignID:    "P1234567",
	Items:        items,
	RecipientAddress: &stackin.Address{
		Street: "Fifth Avenue", Number: "350", Neighborhood: "Midtown",
		City: "New York", State: stackin.AbroadState,
		CountryCode: "2496", CountryName: "Estados Unidos",
	},
	Export: &stackin.Export{ExitState: "SP", Place: "Porto de Santos"},
})
```

The SDK does not check the BACEN code's check digit: the official table has countries whose digit does not close.

An item on a fuel CFOP carries the fuel group (`comb`) as `br.Fuel`, sent under the item's `fuel` key. `AnpCode` (cProdANP, 9 digits), `AnpDescription` and `ConsumerState` are required; `GlpPercent`, `NaturalGasNationalPercent`, `NaturalGasImportedPercent`, `StartingValue`, `Codif`, `TemperatureQuantity` and `BiodieselPercent` are sent only when set. The groups it does not cover (CIDE, encerrante, origComb) still go through `ExtraGroups["comb"]` — one or the other, never both.

```go
biodiesel := 14.0
item := br.Product{
	Description: "Oleo diesel B S10", Amount: 6200.00, NCM: &ncm, CFOP: &cfop,
	Fuel: &br.Fuel{
		AnpCode: "820101034", AnpDescription: "OLEO DIESEL B S10",
		ConsumerState: "SP", BiodieselPercent: &biodiesel,
	},
}
```

The fiscal rules about these fields — the emission window, which purpose needs a referenced key, an export group on a note to Brazil — are the API's, and it answers them by name.

### Checked before the request goes out

`Issue()` refuses, with an `*InvoiceError` and no network call, anything the authorizer is certain to reject:

- `NCM`, when set, must be the full 8 digits, or `00` for a service line. Any other 2-digit chapter is not accepted: the SEFAZ rejects it with `777`.
- `Barcode` (cEAN), when set, must be `SEM GTIN` or a GTIN of 8, 12, 13 or 14 digits, and a GTIN's GS1 check digit must be right (a mistyped digit is caught here instead of by the SEFAZ).
- `RecopiNumber` (nRECOPI), when set, must be the 20-digit RECOPI number; the SDK checks its two modulo-11 check digits (MOC §8.5) before sending.
- On NFE, an item's `Description` (xProd) is at most 120 characters. NFSE descriptions are not bound by it.
- `Discount`, when set, is zero or more; `IssuedAt`, when set, is not the zero `time.Time`; `Purpose` and `Direction`, when set, are one of their constants.
- An item's `Fuel` needs `AnpCode`, `AnpDescription` and `ConsumerState`, and is refused beside `ExtraGroups["comb"]`.
- On NFE, a `RecipientAddress` with `State` `EX` (a buyer abroad) needs `CountryCode` instead of `ZipCode`.
- `RecipientAddress.Street` and `Neighborhood`, when set, need at least 2 characters after trimming, on either document type.
- On NFE, every free-text field sent — `ClientName`, each item's `Description`, `Unit`, `PurchaseOrder`, `PurchaseOrderItem` and `Observations`, the `Fuel` group's text fields, every `RecipientAddress` field, `ForeignID` and every `Export` field — is checked for characters the NF-e cannot carry. Anything the NF-e cannot carry (emoji, CJK, Cyrillic…) is refused, and the error names the field and the character. Accented Portuguese and anything else in ISO-8859-1 (latin1) is fine; tabs and line breaks are collapsed into a space, and typographic quotes, dashes, ellipsis, bullet, `€` and `™` are converted by the API (`‘ ’ ‚` to `'`, `“ ” „` to `"`, `– — −` and `•` to `-`, `…` to `...`, `€` to `EUR`, `™` to `TM`). The SDK sends them unchanged. NFSE is not checked.
- On NFSE, an item with `TaxRetained: true` needs a `RecipientAddress` with `Street`, `Number`, `Neighborhood`, `City`, `State` and `ZipCode` (`CityCode` optional): the ISSQN is retained by the tomador, and the city authorizer refuses the document without their address (`E0237`).

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
	Items:          []br.Product{{Description: "Consultoria", UnitPrice: ptr(1500.00)}},
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

The correction text is 15 to 1000 characters, and the access key must be an NF-e access key (44 characters) — both checked locally before the call. Any other key, an NFS-e's 50-digit one included, returns an `*InvoiceError`: NFS-e has no correction letter, so cancel and reissue instead.

What a CC-e **cannot** fix: anything that changes the tax owed (base, rate,
price, quantity, totals), the buyer or the seller, or the issue date. Those
still mean cancelling and reissuing. The API sends the legally fixed wording
that says exactly this, attached to every correction.

The original document does not change — the CC-e is an event attached to it, and
the authorized XML stays as it was. A document accepts at most 20 of them, and
they are numbered for you.

**NF-e only.** NFS-e has no correction letter. The key already says which one it is, so a key that is not an NF-e access key is refused locally with an `*InvoiceError`. What only the API can know stays an `*APIError` with status `409`: an NF-e-shaped key sent with `stackin.NFSE`, or a key for a document the API does not hold. The NF-e key is 44 characters: digits, except positions 7 to 18, which may carry the uppercase letters of an alphanumeric CNPJ.

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

## Documents issued against you

Everything above serves the **issuer**. These two serve the **recipient**: what
suppliers billed to this CNPJ, and the formal answer to it.

Reading the list never calls the SEFAZ. The authorizer caps how many times a CNPJ
may ask for its distribution per day, so collecting runs on a schedule on the API
side and a page refresh cannot spend that allowance.

```go
page, err := client.Received(20, 0)
if err != nil {
    log.Fatal(err)
}
fmt.Println(page["total"])

if _, err := client.Manifest(accessKey, stackin.Ciencia, ""); err != nil {
    log.Fatal(err)
}
_, err = client.Manifest(
    accessKey,
    stackin.OperacaoNaoRealizada,
    "Mercadoria nunca chegou ao endereco",
)
```

Before you answer a document the SEFAZ sends only a **summary** (`resNFe`): access
key, issuer, amount, date. The **full document** (`nfeProc`) arrives after a
manifestation, and the `schema` field on each row says which one you hold.

The four answers are `210200` Confirmação da Operação, `210210` Ciência da
Operação, `210220` Desconhecimento da Operação and `210240` Operação não
Realizada. Only the last one takes a reason, and it requires one — both rules are
checked locally, before the request goes out, because a round trip to be told a
fixed rule is a round trip wasted.

## Looking up a code, or who a CNPJ belongs to

Two more clients, for the tables an issuer reads while filling a document. Neither writes anything.

```go
ref := stackin.NewFiscalReference(stackin.WithAPIKey("..."))   // country BR by default

ref.NCM.Get("84716052")                                        // one code
ref.NCM.Search(stackin.SearchQuery{Term: "teclado", Limit: 5}) // a page of matches
ref.CFOP.Get("5102")
ref.Kinds()                                                    // what this country has
ref.Kind("ibs_cbs_class").Get("000001")                        // any kind, named or not
ref.Search(stackin.SearchQuery{Term: "teclado"})               // every kind at once

stackin.NewTaxpayer(stackin.WithAPIKey("...")).Get("00000000000191")
```

`CFOP`, `NCM`, `CEST`, `CST`, `CSOSN`, `IssService`, `IcmsFuel` and `IbsCbsClass` have accessors. **`Kind(name)` reaches anything else**, including a classification published after this release — ask `Kinds()` rather than trusting this list.

`metadata` comes back as the API sends it and differs per kind: `utrib` on an NCM, `ncm_code` on a CEST, `tax_type` on a CST, `nil` on an ISS service.

Three things worth knowing before you loop:

- **These share the invoice read allowance** — 600 calls a minute per key, the same bucket `Consult`, `History`, `Received` and `Pdf` draw from. One `Search` page beats N `Get` calls.
- **Ordering is fixed** (kind, then code, ascending). Unlike `History`, `SearchQuery` has no `SortBy`/`OrderBy`.
- **A 404 from `Taxpayer` does not mean the company does not exist.** That registry reloads monthly, so a recently registered CNPJ is simply not in it yet. Do not build a validation rule on it.

`Taxpayer` has one method and keeps one: the registry holds the names and addresses of real people, so there is no search over it, by design.

## Errors

- `*stackin.APIError` — the API responded with a non-2xx status (`StatusCode`, `Detail`) — a 401 here means `api_key` is missing, wrong, or was rotated.
- `*stackin.ConnectionFailedError` — the API didn't respond (network/DNS/timeout).
- `*stackin.InvoiceError` — `Issue()`'s `Items` is empty, missing `NCM`/`CFOP` on an item for NFE, a missing/incomplete `RecipientAddress` on NFE, or any other check listed under [Checked before the request goes out](#checked-before-the-request-goes-out).

Building the full fiscal document (issuer data, service code, tax groups, schema-accurate XML) is the API's job — configured once per company, not passed on every call.

## Examples

Runnable end-to-end programs in [`examples/nfe/`](examples/nfe/) and [`examples/nfse/`](examples/nfse/) — one program per field variant, from the bare minimum to every field filled. `examples/consult_invoice/`, `examples/cancel_invoice/`, and `examples/reissue_invoice/` cover the operations that act on an already-issued document. `examples/lookup_fiscal_reference/` and `examples/lookup_taxpayer/` cover the two read-only clients.
