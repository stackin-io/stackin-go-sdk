# Changelog

## Unreleased

### Features
- **invoice:** `ExternalReference` on `IssueRequest`, sent as `external_reference` — the caller's own id for the document, returned on every read and webhook
- **invoice:** `CityCode` is no longer required on an NFE `RecipientAddress`; the API resolves it from city and state

## v0.11.0 — 2026-11-15

### Features
- **invoice:** require recipient address for retained ISS on NFS-e
- **invoice:** require a 44-digit NF-e key in Correct
- **product:** validate the GTIN check digit
- **validation:** reject non-latin1 text on NF-e before sending
- **validation:** mirror NF-e 4.00 XSD rules for barcode, xProd and address

### Bug Fixes
- **product:** move the GTIN check digit out of the public surface
- **validation:** move the barcode check out of the public surface
- **product:** move the NCM check out of the public surface
- **validation:** refuse only what the API cannot convert to latin1
- **invoice:** require a complete tomador address for retained ISS on NFS-e
- **invoice:** accept alphanumeric-CNPJ NF-e keys in Correct and align docs
- **product:** accept only 8-digit NCM or 00

## v0.10.1 — 2026-11-15

### Chores
- ignore .DS_Store
- **platform:** install action-platform

# Changelog — stackin (Go SDK)

## 0.1.1

### Added
- `IcmsSn900`, `br.Product.ServiceCode`/`ServiceDiscount`/`TaxRetained`/`Observations` (NFSE-only fields).
- `Series`/`Number` on `IssueRequest` for manual or auto-calculated NFe/NFSe numbering.
- `Address.CityCode` — full recipient address instead of just `State`.
- `examples/nfe/` and `examples/nfse/` — one program per field variant, replacing the two catalog examples.
- Test coverage for `br.Product`/`br.Tax`, error types, and the remaining `client.go` branches (97%).

### Fixed
- `examples/nfse` used an invalid CPF `tax_id` and was missing the NFSE-only fields.

## 0.1.0

### Added
- `Invoice` — `Issue()`/`Consult()`/`Cancel()`, HTTP client for the platform. Defaults to `https://sdk.stackin.io`; `WithBaseURL`/`WithEnvironment`/`STACKIN_BASE_URL`/`STACKIN_ENVIRONMENT`/`STACKIN_API_KEY` override it.
- `br.Product` — line item (`Description`/`Amount` universal, `NCM`/`CFOP`/tax groups Brazil-specific, required for NFE).
- `br.Tax`, `br.Icms00`/`Icms40`/`Icms60`/`IcmsSn101`/`IcmsSn102`/`IcmsUfDest`, `br.Ipi`/`IpiTrib`/`IpiNt`, `br.PisAliq`/`PisNt`/`PisOutr`, `br.CofinsAliq`/`CofinsNt`/`CofinsOutr` — already-computed tax groups, ported 1:1 from the Python SDK's field/alias set.
- `Address` — `RecipientAddress` on `Issue()`, only `.State` read (sets NFe's `idDest`).
- `APIError`, `ConnectionFailedError`, `InvoiceError`.
