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

**One struct, `Invoice`** — `Issue()`/`Consult()`/`Cancel()`/`Reissue()`/`Correct()`/`Invalidate()`/`Pdf()`/`History()`/`Submissions()`/`Received()`/`SyncReceived()`/`Manifest()`/`Usage()`/`Readiness()`/`Rejection()`/`Stats()`/`Classify()`/`Invalidations()`, nothing else to instantiate. Each line item is a `br.Product` — `Description`/`Amount` apply to any document type, `NCM`/`CFOP` (plus everything else on `Product`: `CEST`, tax groups, presumed credits...) are Brazil-specific and required per item for NFE, ignored for NFSE.

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
	// The PDF, as raw bytes, drawn by the API from the authorized XML: the
	// DANFE for an NF-e, the DANFSe for an NFS-e. The XML stays the legally
	// valid document.
	document, err := client.Pdf("ACCESS_KEY...", stackin.NFSE)
	err = os.WriteFile("nota.pdf", document, 0o644)

	_ = status
	fmt.Println(invoice, err)
}
```

`ExternalReference` is your own id for the document — an order number, a ticket. It is sent as `external_reference` (left out when empty), stored as sent (at most 255 characters), returned on every read and webhook, and `History(stackin.HistoryQuery{ExternalReference: "order-1234"})` finds the document by it (exact match).

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

An item's IBS/CBS group (Reforma Tributária) is `br.IbsCbs`, sent under the item's `br.ibs_cbs` key. Every field except `CST` and `Classification` (cClassTrib) is sent only when set.

- **On an NF-e** send `RateState`, `RateCity` and `RateFederal` (percent, 0–100), plus `Base`, `ReductionIbs` and `ReductionCbs` when they apply. `ReductionIbs` is the IBS pRedAliq (state and municipality) and `ReductionCbs` the CBS pRedAliq, both in percent: the cClassTrib's `pRedIBS` and `pRedCBS` in the official classification table. The API asks for the reductions on CST 011, 200 and 515 and refuses them on any other.
- **On an NFS-e** send only `CST`, `Classification` and `OperationIndicator` (cIndOp, 6 digits). The API mandates `OperationIndicator` on a production NFS-e and refuses the rates there. An ME/EPP also needs `SimplesApuracaoIbsCbs` 3 on the company (`Company.Update`, below) for the NFS-e group.

Which fields each document type takes is the API's rule, not the SDK's: the SDK sends what is set and the API answers by name.

```go
reduction := 60.0
state, city, federal := 0.1, 0.0, 0.9
nfeItem := br.Product{
	Description: "Medicamento", Amount: 100.00, NCM: &ncm, CFOP: &cfop,
	IbsCbs: &br.IbsCbs{
		CST: "200", Classification: "200009",
		RateState: &state, RateCity: &city, RateFederal: &federal,
		ReductionIbs: &reduction, ReductionCbs: &reduction,
	},
}

indicator := "100301"
nfseItem := br.Product{
	Description: "Consultoria", Amount: 100.00,
	IbsCbs: &br.IbsCbs{CST: "000", Classification: "000001", OperationIndicator: &indicator},
}
```

**Breaking since v0.23.0:** `RateState`, `RateCity` and `RateFederal` are `*float64`, like `ReductionIbs`, so a group can leave them out (an NFS-e) and still send a real `0` (an NF-e with `RateCity` 0). A `float64` field cannot say both. Existing code that assigned a value must take its address:

```text
Before:  IbsCbs{RateState: 0.1, RateCity: 0, RateFederal: 0.9}
After:   state, city, federal := 0.1, 0.0, 0.9
         IbsCbs{RateState: &state, RateCity: &city, RateFederal: &federal}
```

NF-e sale terms (all optional, NF-e only, sent only when set): `Presence` (`"0"`-`"5"`, `"9"`), `Intermediary`, `Transport`, `Payments` and `Change`. `Product.NetWeight` and `GrossWeight` are one unit's kg. Pointers (`TransportVolume.Quantity *int`, the weights and `Change *float64`) let a real `0` go out; `Payment.Amount` is always sent. A nil `Transport.Volumes` is left out and an empty non-nil one is sent as `[]`. The SDK checks only the code sets of `Presence` and `FreightMode`, a two-digit `Method`, `Description` on method `"99"`, weights of at least 0 (volume weights at most 3 decimal places, `Payment.Amount` and `Change` at most 2), 1 to 100 payments (an empty non-nil `Payments` is refused) and a 2-60 character `Intermediary.Identifier`; the fiscal rules are the API's. `Reissue` takes none of them: it reuses the stored terms, and `SaleTermsOf(detail)` reads them back.

```go
zero := 0.0
inv.Issue(stackin.IssueRequest{ /* ... */
	Presence:     "2",
	Intermediary: &stackin.Intermediary{TaxID: "11222333000181", Identifier: "loja-1"},
	Transport:    &stackin.Transport{FreightMode: "1", Volumes: []stackin.TransportVolume{{Species: "CAIXA", NetWeight: &zero}}},
	Payments:     []stackin.Payment{{Method: "17", Amount: 100}},
	Change:       &zero,
})
```

Catalog items take `NetWeight`, `GrossWeight`, `IbsCbsCST`, `IbsCbsClass` and `Nbs` (`ItemFields` and `ItemChanges`). Catalog weights are checked for at least 0 and at most 6 decimal places. `ItemChanges.ClearNetWeight` and `ClearGrossWeight` send null to clear a weight.

`Contingency` — NF-e only: `true` sends the document through the UF's contingency authorizer (SVC-AN or SVC-RS, `tpEmis` 6/7) for when its own is down. Left `false`, nothing is sent:

```go
inv.Issue(stackin.IssueRequest{ /* ... */ Contingency: true})
```

The fiscal rules about these fields — the emission window, which purpose needs a referenced key, an export group on a note to Brazil — are the API's, and it answers them by name.

### NBS code on an NFS-e

`br.Product.Nbs` is the NBS code of the service (cNBS), nine digits or dotted (`1.1502.10.00`). It sits on the line beside `ServiceCode`, NFS-e only, and is left out when empty. The API requires it on a line that carries `IbsCbs` (`422 NBS_REQUIRED`) and refuses an unknown code (`422 NBS_UNKNOWN`); both come back as the ordinary `*APIError`. Catalog items take it too: `ItemFields.Nbs` and `ItemChanges.Nbs` (pointing at `""` clears it).

```go
inv.Issue(stackin.IssueRequest{
	DocumentType: stackin.NFSE,
	// ...
	Items: []br.Product{{Description: "Consultoria", UnitPrice: ptr(1500.00), ServiceCode: ptr("17.01"), Nbs: "1.1502.10.00"}},
})
```

### Activity warnings on issue

`Issue` returns the same result map as before with a `warnings` key added: always present, a `[]any` of `map[string]any` with `code`, `message` and `field` (`nil` when the finding has no field), empty when there is none. The codes are `MEI_ACTIVITY_NOT_ALLOWED`, `ACTIVITY_NOT_REGISTERED` and `MEI_NO_ICMS_ACTIVITY`. The document is issued either way, unless the owner turned on `BlockActivityMismatch` (see Fixing the issuer).

```go
result, err := inv.Issue(req)
for _, w := range result["warnings"].([]any) {
	fmt.Println(w.(map[string]any)["code"])
}
```

`IssueRequest.AcceptLanguage` (`pt-BR`, `en` or `es`; the API falls back to English) localizes the warning messages of `Issue` and `Validate`. It is sent as the `Accept-Language` header only, never in the body, and omitted when empty.

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
- On NFSE, an item with `TaxRetained: ptr(true)` needs a `RecipientAddress` with `Street`, `Number`, `Neighborhood`, `City`, `State` and `ZipCode` (`CityCode` optional): the ISSQN is retained by the tomador, and the city authorizer refuses the document without their address (`E0237`).

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

Ranges already reported are listed, a page at a time, with the SEFAZ `status` and
`protocol` of each — read it before reporting the same range twice:

```go
page, err := inv.Invalidations(stackin.PageQuery{Limit: 20})
```

## Before the first document, and after a refusal

```go
ready, err := inv.Readiness()          // GET /company
if ready["ready"] == false {
	fmt.Println(ready["missing"])       // certificate, certificate_expired, certificate_unreadable, address, state_registration, plan
	fmt.Println(ready["by_document_type"], ready["fix_url"])
}

why, err := inv.Rejection(invoiceID)   // a rejected document, by its local id
fmt.Println(why["code"], why["explanation"], why["steps"], why["suggested"])

stats, err := inv.Stats(7)             // totals, by type and status, this week against the last
```

`Readiness()` says whether the company can issue right now and, when it cannot,
what is missing, plus the certificate's expiry, the tax regime (CRT) and the
default série per document type. `by_document_type` says it per `nfe`/`nfse`,
with the address fields at fault; `fix_url` is the dashboard screen for the first
blocker; `suggested_fiscal_fields` is what the CNPJ registry suggests, never
applied. `Rejection()` reads the authorizer's answer
(`Submissions()` has it raw) in plain words, with the field to fix and, when there
is one, a suggested value; a document that is not rejected is an `*APIError` 409.
`Stats()` takes `days` from 1 to 365, checked locally; `0` sends nothing and the
API answers for 30.

## Fixing the issuer

`Company.Update()` (`PATCH /company`) fixes what `Readiness()` reports missing and
answers the same fields, plus the stored `address` and `fiscal_fields`:

```go
company := stackin.NewCompany(stackin.WithAPIKey(os.Getenv("STACKIN_API_KEY")))

result, err := company.Update(stackin.CompanyUpdate{
	Address: &stackin.CompanyAddress{
		Street:           "Av. Paulista",
		Number:           "1000",
		Neighborhood:     "Bela Vista",
		MunicipalityCode: "3550308", // 7-digit IBGE code, in the company's state
		ZipCode:          "01310-100",
	},
	FiscalFields: &stackin.FiscalFields{TaxRegime: "1", SimplesNacional: "3"}, // owner only
})
fmt.Println(result["ready"], result["missing"])
```

Only what is set is sent. There is no complement, and the state changes only in
the dashboard. `MunicipalityName` is derived from the code when left out. The
municipality code (7 digits), the CEP (8 digits, mask stripped) and the
`TaxRegime` (1–4), `SimplesNacional` and `SimplesApuracao` (1–3) and
`SimplesApuracaoIbsCbs` (1 or 3) codes are checked locally. `FiscalFields` are the company owner's alone: anyone else gets an
`*APIError` 403. An OAuth credential needs `company:write`
(`*InsufficientPermissionError` without it). A 503 means the IBGE table could not
be read: nothing was written, retry after `Retry-After`.

`CompanyUpdate.BlockActivityMismatch` (`*bool`, owner only, anyone else gets an `*APIError` 403) makes the API refuse an issuance with `MEI_ACTIVITY_NOT_ALLOWED` or `ACTIVITY_NOT_REGISTERED` as a 422 instead of a warning. It is sent only when set, so `ptr(false)` turns it off:

```go
block := true
company.Update(stackin.CompanyUpdate{BlockActivityMismatch: &block})
```

## Suggesting a classification

```go
suggestion, err := inv.Classify(stackin.ClassifyRequest{
	Description:    "Cerveja artesanal IPA 500ml",
	DocumentType:   stackin.NFE,
	RecipientState: "SP",
	Answers:        map[string]any{"purpose": "resale"}, // optional, from missing_information
})
best := suggestion["primary_candidate"] // ncm, cfop, cst/csosn, cest, confidence
```

**A suggestion, never a decision.** Every code in a candidate is read from the
published tables, but the NCM and CFOP decide the tax owed: show it, have someone
confirm it, and only then put it on an item. `missing_information` lists what
would settle it; answer through `Answers` and ask again. `Description` is 3 to
500 characters, checked locally.

## Documents issued against you

Everything above serves the **issuer**. These two serve the **recipient**: what
suppliers billed to this CNPJ, and the formal answer to it.

Reading the list never calls the SEFAZ. Collecting runs on a schedule on the API
side, every six hours, and `SyncReceived()` asks for it now. After a read that
finds nothing more the SEFAZ makes the CNPJ wait an hour; a sync inside that wait
comes back as a `*stackin.RateLimitedError` with `Code` `DISTRIBUTION_COOLDOWN`,
`RetryAt` and `RetryAfter` (seconds).

The first 50 received documents a month are included in every plan; each one
after that takes R$ 0,05 from the account credit. Some existing accounts are
exempt, and `Usage().ReceivedFree` says whether this one is.

```go
sync, err := client.SyncReceived()
var limited *stackin.RateLimitedError
if errors.As(err, &limited) {
    fmt.Println("next read allowed at", limited.RetryAt)
} else if err != nil {
    log.Fatal(err)
} else {
    fmt.Println(sync.Stored, "new documents")
}

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

## What was issued, and the plan behind it

`History` lists what this company issued, newest first. `ExternalReference` finds
the documents issued with exactly that reference, and `Fields: "summary"` returns a
lighter item per invoice.

```go
page, err := client.History(stackin.HistoryQuery{
	ExternalReference: "order-1234",
	Fields:            "summary",
	Limit:             20,
})

usage, err := client.Usage() // GET /api/v1/company/usage
fmt.Println(usage.PlanName, usage.UsedInvoices, "/", usage.IncludedInvoices)
fmt.Println(usage.UsedReceived, "/", usage.IncludedReceived, "received, free:", usage.ReceivedFree)
fmt.Println("credit (cents):", usage.CreditBalance)
```

`Usage()` returns a `*stackin.Usage`: `PlanName`, `PlanSlug`, `IncludedInvoices`,
`UsedInvoices`, `OveragePriceCents` (nil when the plan has no overage),
`IncludedReceived`, `UsedReceived`, `ReceivedPriceCents`, `ReceivedFree`,
`CurrentPeriodStart`, `CurrentPeriodEnd` (nil when open-ended), `CreditBalance`
(cents) and `CompaniesOnAccount`. The numbers are the paying account's, shared by
every company it holds.

## Saved clients

The company keeps the recipients it issues to. A successful issuance saves a new recipient on its own; `Clients` reads and edits them, and `IssueRequest.ClientID` issues to one without retyping it.

```go
clients := stackin.NewClients(stackin.WithAPIKey("..."))

page, err := clients.List(stackin.ClientQuery{Search: "acme", Limit: 20}) // active ones, by name
inactive := false
clients.List(stackin.ClientQuery{IsActive: &inactive})                  // the deactivated ones

acme, err := clients.Create(stackin.ClientFields{Name: "ACME Ltda", TaxID: "11.222.333/0001-81"})
var exists *stackin.ClientAlreadyExistsError
if errors.As(err, &exists) {
	acme, err = clients.Get(exists.ClientID) // it was there already
}
id := acme["id"].(string)

phone, empty := "+5511999999999", ""
clients.Update(id, stackin.ClientChanges{Phone: &phone, Notes: &empty}) // "" clears
clients.Deactivate(id)                                                  // kept, hidden from List
clients.Reactivate(id)

stackin.NewInvoice(stackin.WithAPIKey("...")).Issue(stackin.IssueRequest{
	DocumentType: stackin.NFSE,
	ClientID:     id, // name, tax id, address from it
	Items:        items,
})
```

- `List` filters by `Search` (part of the name, the tax id masked or not, or the foreign id), `Source` (`manual`, `issued`, `issuance`) and `IsActive` (nil lists the active clients, as the API does), and pages with `Limit`/`Offset`. `SortBy` takes `name`, `last_used_at`, `created_at` or `uses_count`.
- `Update` sends only the fields set in `ClientChanges`: a nil pointer is left alone, one pointing at `""` clears the field, `Address` replaces the whole address and `ClearAddress` removes it. The API checks the client as a whole, so clearing a Brazilian client's tax id is refused like a create without one.
- On `Issue`, the client fills whichever of `ClientName`, `TaxID`, `ForeignID` and `RecipientAddress` are empty; the ones set win, and the empty ones are left out of the request. The client's e-mail and phone likewise fill `ClientEmail` and `ClientPhone` when empty. With `ClientID` the address checks apply only to a `RecipientAddress` that is set: the API checks the merged request. A deactivated client is refused there with a `*stackin.ClientInactiveError` (422) until it is reactivated.
- `UpdateClient: true` rewrites an existing client with that request's recipient data. Left false, the issuance only counts one more use of it.
- `ClientEmail` and `ClientPhone` (8 to 15 digits, optional leading `+`) are saved on the client created after a successful issuance, or on an existing one only with `UpdateClient: true`, and only the ones sent. They are also kept on the invoice and sent again on a reissue (which does not touch the client), and written into the fiscal document: NF-e `dest/email` (up to 60 characters) and `dest/enderDest/fone` (digits only, up to 14, only with a `RecipientAddress`); NFS-e `toma/email` (up to 80) and `toma/fone` (up to 20 digits). The phone goes out digits only, area code first; a `+55` is dropped unless the recipient is abroad. A value the layout cannot carry (too long, an e-mail with characters beyond Latin-1) is left out of the document and the issuance goes on.
- Reading needs `invoice:read`; creating, editing and deactivating need `invoice:issue`. `WithCompany` and `ForCompany` work here as on `Invoice`.

### Reading the XML of a received document

```go
xml, err := client.ReceivedXML(accessKey)
if err != nil {
    log.Fatal(err)
}
os.WriteFile(accessKey+".xml", xml, 0o644)
```

`ReceivedXML` returns the procNFe as raw bytes. A document held only as a summary answers 409 with `Code` `RECEIVED_XML_NOT_AVAILABLE` until it is manifested; 402 is a `*PaymentRequiredError` with `Code` `RECEIVED_PAYMENT_REQUIRED`. Downloading is never billed.

## Webhooks

`Webhooks` registers where the company's events are delivered: `invoice.authorized`,
`invoice.rejected`, `invoice.cancelled`, `invoice.corrected`, `received_invoice.stored` and
`received_invoice.full_document` (`stackin.WebhookEvents`).
An endpoint receives the events of the credential's own environment only.

```go
hooks := stackin.NewWebhooks(stackin.WithAPIKey("..."))

endpoint, err := hooks.Create(stackin.WebhookEndpoint{
	URL:    "https://shop.example/stackin",
	Secret: os.Getenv("STACKIN_WEBHOOK_SECRET"),                 // 16 to 255 characters
	Events: []string{"invoice.authorized", "invoice.rejected"}, // nil: every event
})
hooks.List()                                   // a plain list; secrets are never returned
hooks.Deliveries(stackin.PageQuery{Limit: 20}) // every attempt: status, attempts, last_error
hooks.Delete(endpoint["id"].(string))          // stops delivery, keeps the history
```

The secret signs every delivery and is not returned again, so keep it. An `https://`
URL, the secret's length and the event names are checked locally; the API also
refuses a URL that points at a private network. On an OAuth credential every call
needs `webhook:manage`. `WithCompany` and `ForCompany` work here as on `Invoice`.

## Items catalog

The company keeps the products and services it sells in one catalog, told apart by `Kind` (`product` for an NF-e, `service` for an NFS-e). `Items` reads and edits it, and `br.Product.ItemID` issues a line from one without retyping it.

```go
items := stackin.NewItems(stackin.WithAPIKey("..."))

page, err := items.List(stackin.ItemQuery{Kind: "product", Search: "cafe", Limit: 20}) // active ones, by description

// Upsert by ExternalReference (or Code): created is true for a 201, false for a 200.
// UnitPrice and ISSRetained are pointers: nil is not sent, &zero / &no are.
price := 18.9
coffee, created, err := items.Upsert(stackin.ItemFields{
	Kind: "product", Description: "Cafe torrado 500g", Code: "CAF-500",
	UnitPrice: &price, Unit: "UN", ExternalReference: "ns-variant-1", Source: "nuvemshop",
	NCM: "09012100", CFOP: "5102",
})
var exists *stackin.ItemAlreadyExistsError
if errors.As(err, &exists) {
	coffee, err = items.Get(exists.ItemID) // the code belongs to another item
}
id := coffee["id"].(string)

notes, empty := "fresh", ""
items.Update(id, stackin.ItemChanges{Notes: &notes, Unit: &empty}) // "" clears
items.Deactivate(id)                                               // kept, hidden from List
items.Reactivate(id)

stackin.NewInvoice(stackin.WithAPIKey("...")).Issue(stackin.IssueRequest{
	DocumentType: stackin.NFE,
	ClientID:     clientID,
	Items:        []br.Product{{ItemID: id, Quantity: 2}}, // description, price, unit and fiscal codes from it
})
```

- `List` filters by `Kind`, `Search` (part of the description, code or external reference; digits also match the NCM and GTIN), `Source` (`manual`, `import`, `nuvemshop`, `invoice`) and `IsActive` (nil lists the active items, as the API does), and pages with `Limit`/`Offset`. `SortBy` takes `description`, `code`, `unit_price`, `last_used_at`, `created_at` or `uses_count`.
- `Upsert` is the public `POST /items` (`ItemFields.UnitPrice *float64` and `ISSRetained *bool` are pointers, so a price of `0` and `iss_retained: false` can be sent while nil is left out; the price goes out as a decimal string, and `UnitPriceText: "18.9000000001"` (up to 10 places, checked) keeps every digit a float would round. `ItemChanges` has the same `UnitPriceText *string`, `""` clearing the price): it updates the company's item with the same `ExternalReference` (or the same `Code` when none is sent) with the fields sent, and keeps its source and active flag; otherwise it creates one. Sending the same fields twice never makes two items. `Source` is used only when it creates. `Create` is the same call without the `created` flag (kept for parity with `Clients`).
- `Update` sends only the fields set in `ItemChanges`: a nil pointer is left alone and a text pointing at `""` clears the field. `ISSRetained *bool` tells `false` from unset. The API checks the item as a whole, so turning a product into a service while it still has an `NCM` is refused (422).
- `Deactivate` is `DELETE /items/{id}` (204, no body). A deactivated item still holds its code and is refused on `Issue` until `Reactivate`.
- On `Issue`, a line with `ItemID` may leave `Description` empty: it is left out of the request and the catalog fills it, with `Unit`, `Barcode`, `NCM`, `CFOP`, `CEST`, `Tax`, `ServiceCode`, `TaxRetained` and the price (when neither `Amount` nor `UnitPrice` is set). A field set on the line wins. With `ItemID`, the SDK does not default `Unit` to `UN`, sends `tax_retained` only when `TaxRetained` is set (it is a `*bool`, so `ptr(false)` turns off a retention the item has) and does not require `NCM`/`CFOP` locally on an NF-e: the API checks the filled line. An item whose tax situation needs an ICMS base and rate is refused (`*ItemTaxRequiredError`) unless the line sends its own `Tax`.
- Reading needs `invoice:read`; creating, editing and deactivating need `invoice:issue`. `WithCompany` and `ForCompany` work here as on `Invoice`. Responses are plain maps, as on `Clients`.

## Several companies on one credential

An API key is one company and needs nothing here: with no company set, requests are exactly what they were before. A credential authorized for several companies (an OAuth connection) has to say which one each call acts on, by the company id or its CNPJ (punctuation is ignored). It goes out as the `X-Stackin-Company` header on every request, lookups included.

```go
client := stackin.NewInvoice(
	stackin.WithAPIKey(token),
	stackin.WithCompany("00.000.000/0001-91"),
)

// One call on another company; client itself is unchanged.
client.ForCompany("6f1c2b8e-...").Consult("ACCESS_KEY...", stackin.NFE)

// Which companies this credential reaches, and what it may do on each.
access, err := client.Companies() // GET /api/v1/me/companies
for _, company := range access.Companies {
	fmt.Println(company.CompanyID, company.Name, company.TaxID, company.Permissions)
}
```

`Companies()` returns an `*AuthorizedCompanies` (`Environment`, `Companies []AuthorizedCompany`). An API key lists as its one company with every permission.

Three answers come back typed, and none of them is worth repeating unchanged:

| Error | When |
|---|---|
| `*stackin.CompanyRequiredError` | 422 `company_required`: the credential covers several companies and the call named none. `Companies` lists them. |
| `*stackin.CompanyNotInScopeError` | 403 `company_not_in_scope`: the company named is not covered. `Companies` lists the ones that are. |
| `*stackin.InsufficientPermissionError` | 403 `INSUFFICIENT_PERMISSION`: the company is covered but the permission is missing. `Description` reads `needs X, has Y, company=<id>`. |

Each wraps the `*APIError` it used to be, so `errors.As(err, &apiErr)` keeps working; a plain type assertion to `*APIError` does not match them.

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

`Cities(name, CityQuery{State, Country})` lists the municipalities whose name starts with `name`, at most 20, each with its IBGE `code` — issuing resolves `CityCode` by itself, so this is for a name that matches more than one:

```go
ref.Cities("santa rita", stackin.CityQuery{State: "SP"}) // [{country: BR, code: ..., name: ..., state: SP}]
```

`CalculateIbsCbs(IbsCbsCalculationRequest)` simulates the IBS and CBS of 1 to 990 items (`POST /ibs-cbs/calculate`, scope `invoice:read`). It needs no company, issues nothing and stores nothing:

```go
amount, city := 100.0, 0.0
calc, err := ref.CalculateIbsCbs(stackin.IbsCbsCalculationRequest{
	DocumentType:  "nfe", // nfe (default), nfce or nfse: only picks the model the cClassTrib is checked for
	OperationDate: "2026-03-10",
	Items: []stackin.IbsCbsCalculationItem{
		{CST: "000", Classification: "000001", Amount: &amount}, // Amount or Base is required
	},
	Rates: &stackin.IbsCbsRates{RateCity: &city}, // optional; an item's own rate wins
})
fmt.Println(calc.Total.Ibs, calc.Total.Cbs, calc.Items[0].RatesSource)
```

Only what is set is sent. Left out, the official 2026 rates apply to a 2026 date (CBS 0.9%, IBS UF 0.1%, IBS Mun 0%); for any other year the API refuses the call with `IBS_CBS_RATES_REQUIRED` (an `*APIError` 422). The item count, the `CST` (3 digits) and `Classification` (6 digits) patterns, `Amount`/`Base` greater than zero, rates and reductions between 0 and 100 and the `OperationDate` format are checked locally. The result is typed (`IbsCbsCalculation`, `IbsCbsCalculatedItem`, `IbsCbsIndicators`, `IbsCbsAppliedRates`, `IbsCbsAmounts`) and keeps every amount and rate as the decimal string the API sends. The amounts are the NF-e's: the simulation does not compute an NFS-e's, which the Sefin Nacional does.

`metadata` comes back as the API sends it and differs per kind: `utrib` on an NCM, `ncm_code` on a CEST, `tax_type` on a CST, `nil` on an ISS service.

Three things worth knowing before you loop:

- **These share the invoice read allowance** — 600 calls a minute per key, the same bucket `Consult`, `History`, `Received` and `Pdf` draw from. One `Search` page beats N `Get` calls.
- **Ordering is fixed** (kind, then code, ascending). Unlike `History`, `SearchQuery` has no `SortBy`/`OrderBy`.
- **A 404 from `Taxpayer` does not mean the company does not exist.** That registry reloads monthly, so a recently registered CNPJ is simply not in it yet. Do not build a validation rule on it.

`Taxpayer` has one method and keeps one: the registry holds the names and addresses of real people, so there is no search over it, by design.

## Dry run and DPS lookup

`Validate` takes the same `IssueRequest` as `Issue` and runs the API's checks without transmitting, numbering or charging. A document that would be refused is a result, not an error:

```go
result, err := client.Validate(req)
if err != nil {
    log.Fatal(err)
}
if !result.Valid {
    for _, f := range result.Errors {
        fmt.Println(f.Code, f.Field, f.Message)
    }
}
for _, w := range result.Warnings {
    fmt.Println(w.Code, w.Field, w.Message)
}
```

Any other refusal (malformed body, unknown client or item, credentials, rate limit) comes back as the error `Issue` would return.

`ConsultByDPS` reads an NFS-e by the DPS id, the `DocumentID` of an `OUTCOME_UNKNOWN` error, and answers like `Consult` (it may carry `status: "cancelled"`). The API answers 400 when the id is not this company's and 404 when the authority holds no note for it, meaning nothing was issued.

```go
note, err := client.ConsultByDPS(apiErr.DocumentID)
```

## Errors

- `*stackin.APIError` — the API responded with a non-2xx status (`StatusCode`, `Detail`) — a 401 here means `api_key` is missing, wrong, or was rotated. Every error above wraps one, and it also carries what the body says about the failure, zero when absent: `Code` (a rule code, an authority `cStat` or `CONNECTION_FAILED`/`OUTCOME_UNKNOWN`), `Field` (the request path of the offending value), `Retryable` (`*bool`, nil when the API did not say), `DocumentID` (the DPS id of an `OUTCOME_UNKNOWN` NFS-e, for `ConsultByDPS`), `SefazRejection`, `Expected` and `Received`. `Detail` is the message, not the raw JSON, when the body has one.
- `*stackin.CompanyRequiredError`, `*stackin.CompanyNotInScopeError`, `*stackin.InsufficientPermissionError` — the company or permission answers described under [Several companies on one credential](#several-companies-on-one-credential). Each wraps an `*APIError`.
- `*stackin.ClientAlreadyExistsError` — 409 `CLIENT_EXISTS`: `Clients.Create` or `Update` with a tax id or foreign id the company already has on another client. `ClientID` names that client. It wraps an `*APIError`.
- `*stackin.PaymentRequiredError` — any 402: the plan's allowance is spent and neither an overage price nor the account credit covers the call; `Detail` says why. It wraps an `*APIError`.
- `*stackin.RateLimitedError` — any 429. `RetryAfter` is the `Retry-After` header in seconds (derived from `RetryAt` when the header is missing); `Code` and `RetryAt` are set when the body carries them (`DISTRIBUTION_COOLDOWN` on `SyncReceived`). It wraps an `*APIError`.
- `*stackin.ClientInactiveError` — 422 `CLIENT_INACTIVE`: the saved client `ClientID` is deactivated; reactivate it first. It wraps an `*APIError`.
- `*stackin.ClientNotFoundError` — 404 `CLIENT_NOT_FOUND`: no saved client of this company has that id. It wraps an `*APIError`.
- `*stackin.ItemAlreadyExistsError` — 409 `ITEM_EXISTS`: `Items.Create`, `Upsert` or `Update` with a code (or external reference) the company already has on another item, active or not. `ItemID` names it and `Field` says which. It wraps an `*APIError`.
- `*stackin.ItemNotFoundError` — 404 `ITEM_NOT_FOUND`: no item of this company has that id. It wraps an `*APIError`.
- `*stackin.ItemInactiveError`, `*stackin.ItemKindMismatchError`, `*stackin.ItemTaxRequiredError`, `*stackin.ItemTaxRegimeMismatchError` — the 422 answers of `Issue` for a line with `ItemID`: the item is deactivated (`ITEM_INACTIVE`), is a service on an NF-e or a product on an NFS-e (`ITEM_KIND_MISMATCH`), has a tax situation that needs the line's own `Tax` (`ITEM_TAX_REQUIRED`), or has a CST/CSOSN that does not fit the company's tax regime (a CSOSN is for Simples Nacional, a CST for the rest) or an export CFOP (`ITEM_TAX_REGIME_MISMATCH`). `ItemID` names the item and, on the last three, `Index` the line. Each wraps an `*APIError`.
- `*stackin.ConnectionFailedError` — the API didn't respond (network/DNS/timeout).
- `*stackin.InvoiceError` — `Issue()`'s `Items` is empty, missing `NCM`/`CFOP` on an item for NFE, a missing/incomplete `RecipientAddress` on NFE, or any other check listed under [Checked before the request goes out](#checked-before-the-request-goes-out). Also `Stats()` with `days` outside 1–365, `Classify()` with a `Description` outside 3–500 characters, a `DocumentType` other than NFE/NFSE or a `RecipientState` that is not two letters, `FiscalReference.Cities()` with a `name` outside 2–100 characters or a `State` that is not two letters, `Webhooks.Create()` with a URL that is not `https://`, a secret outside 16–255 characters or an event name not in `WebhookEvents`, and an invoice or endpoint id that is empty, `.` or `..` (ids are escaped as one path segment).

Building the full fiscal document (issuer data, service code, tax groups, schema-accurate XML) is the API's job — configured once per company, not passed on every call.

## Examples

Runnable end-to-end programs in [`examples/nfe/`](examples/nfe/) and [`examples/nfse/`](examples/nfse/) — one program per field variant, from the bare minimum to every field filled. `examples/consult_invoice/`, `examples/cancel_invoice/`, and `examples/reissue_invoice/` cover the operations that act on an already-issued document. `examples/lookup_fiscal_reference/` and `examples/lookup_taxpayer/` cover the two read-only clients.
