# Changelog

## v0.29.0 — 2026-11-25

### Features
- NBS reduction fields, contingency moment and reason, Accept-Language on classify and received (#64)

### Docs
- received sync object, classify origin answer and the new NBS fields (#64)

### CI
- **publish:** build and publish the release tag, commit nothing back (#61)
- **publish:** publish when Action Platform deploys a release (#61)

### Chores
- **platform:** declare scopes and the deploy targets the platform observes

## v0.27.0 — 2026-11-24

### Features
- **nfse:** DPS fields on the issuance request
- **company:** NFS-e settings in FiscalFields
- **reference:** NBSSuggestions for a service code

### Docs
- **readme:** document the NFS-e DPS fields and NBSSuggestions

### Tests
- **nfse:** cover DPS fields, company settings and NBS suggestions

## v0.26.0 — 2026-11-24

### Features
- **company:** add BlockActivityMismatch to CompanyUpdate
- **invoice:** add warnings to the issue result and AcceptLanguage to issue and validate
- **items:** send nbs on NFS-e lines and catalog items

### Refactoring
- **items:** rename Nbs to NBS to match NCM and CFOP

### Docs
- **readme:** document nbs, warnings, AcceptLanguage and BlockActivityMismatch

### Tests
- cover nbs, issue warnings, accept-language and block_activity_mismatch

### Style
- drop explanatory comments from the new fields

## v0.25.0 — 2026-11-23

### Features
- **items:** check catalog weights to 6 decimal places (#53)
- **invoice:** decimal-place checks on payments, change and volume weights; clear catalog weights (#53)
- **invoice:** NF-e sale terms, weights and catalog IBS/CBS codes (#53)

### Bug Fixes
- **invoice:** refuse empty payments and negative line weights, rename TransportVolume (#53)

### Docs
- **changelog:** drop stale Unreleased section (#51)

## v0.24.0 — 2026-11-22

### Features
- **invoice:** ReceivedXML, ConsultByDPS, Validate and structured API errors (#49)

### Bug Fixes
- **errors:** keep parsed fields when one detail value has an unexpected type (#49)

## v0.23.0 — 2026-11-22

### Breaking Changes
- **ibs-cbs:** optional rates and operation_indicator on IbsCbs, simples_apuracao_ibs_cbs on FiscalFields

### Features
- **reference:** CalculateIbsCbs simulates IBS and CBS (#47)

### Docs
- **readme:** IBS/CBS simulation, NFS-e ibs_cbs shape and the rate pointer migration (#47)
- **issue:** client_id also fills the recipient's e-mail and phone

## v0.22.1 — 2026-11-22

### Docs
- **readme:** client email and phone are written into the fiscal document (stackin-io/stackin-api#661)

## v0.22.0 — 2026-11-22

### Features
- **invoice:** ClientEmail and ClientPhone on IssueRequest (#43)

## v0.21.0 — 2026-11-22

### Features
- **items:** ITEM_TAX_REGIME_MISMATCH error (stackin-io/stackin-api#642)
- **items:** items catalog resource and item_id on issue (stackin-io/stackin-api#642)

### Bug Fixes
- **items:** TaxRetained is a *bool, the unit price goes out as a decimal string (stackin-io/stackin-api#642)
- **items:** UnitPrice and ISSRetained are pointers so 0 and false can be sent (stackin-io/stackin-api#642)

### Chores
- **gitignore:** block certificate, key and .env files (stackin-io/devops#47)

## v0.20.0 — 2026-11-21

### Features
- **company:** Update fixes the issuer address and fiscal fields (stackin-api#554)

### Docs
- **readme:** fixing the issuer and the new readiness fields (stackin-api#554)
- **invoice:** readiness by document type, fix_url and suggested fields (stackin-api#554)

### Tests
- **company:** update body, local refusals and API errors (stackin-api#554)

## v0.19.0 — 2026-11-20

### Features
- **invoice:** Stats, Readiness, Rejection, Classify and Invalidations
- **invoice:** Contingency on IssueRequest
- **reference:** Cities on FiscalReference
- **webhooks:** Webhooks client — Create, List, Delete and Deliveries

### Bug Fixes
- **invoice:** escape the invoice id in the Submissions and Rejection paths

### Docs
- **readme:** readiness, rejection, classification, cities, webhooks and contingency

### Tests
- **invoice:** public routes, contingency and webhooks

## v0.18.0 — 2026-11-18

### Features
- **history:** external_reference and fields filters (#36)
- **company:** Usage and SyncReceived (#36)
- **errors:** type 402, 429, CLIENT_INACTIVE and CLIENT_NOT_FOUND (#36)

### Bug Fixes
- **errors:** derive RetryAfter from retry_at when the header is missing (#36)

### Docs
- PDF for NF-e too, received-invoice wait and pricing, new methods and errors (#36)

## v0.17.0 — 2026-11-17

### Features
- **clients:** clients resource and client_id on issue (stackin-io/stackin-api#380)

### Docs
- **readme:** saved clients section (stackin-io/stackin-api#380)

## v0.16.0 — 2026-11-17

### Features
- **br:** add IBS/CBS rate reductions to the item group (stackin-io/stackin-api#326)

## v0.15.0 — 2026-11-16

### Features
- **errors:** type company_required, company_not_in_scope and INSUFFICIENT_PERMISSION (#31)
- **client:** list the companies a credential covers with Companies (#31)
- **client:** select the company per client and per call with X-Stackin-Company (#31)

### Docs
- **readme:** document company selection, Companies and the company errors (#31)

### Tests
- **client:** cover the company header, Companies and the three company errors (#31)

## v0.14.0 — 2026-11-15

### Features
- **invoice:** send the API v0.8-v0.9 issue fields
- **product:** send the fuel group (comb) as br.Fuel

### Docs
- **readme:** document the API v0.8-v0.9 issue fields

## v0.13.0 — 2026-11-15

### Features
- **product:** validate the RECOPI check digits

## v0.12.0 — 2026-11-15

### Features
- **client:** external_reference on Issue, CityCode optional on an NF-e address

### Docs
- **readme:** document ExternalReference and the optional CityCode

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
