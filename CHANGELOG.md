# Changelog — stackin (Go SDK)

## 0.1.0

### Added
- `Invoice` — `Issue()`/`Consult()`/`Cancel()`, HTTP client of stackin-api. Defaults to `https://sdk.stackin.io`; `WithBaseURL`/`WithEnvironment`/`STACKIN_BASE_URL`/`STACKIN_ENVIRONMENT`/`STACKIN_API_KEY` override it.
- `br.Product` — line item (`Description`/`Amount` universal, `NCM`/`CFOP`/tax groups Brazil-specific, required for NFE).
- `br.Tax`, `br.Icms00`/`Icms40`/`Icms60`/`IcmsSn101`/`IcmsSn102`/`IcmsUfDest`, `br.Ipi`/`IpiTrib`/`IpiNt`, `br.PisAliq`/`PisNt`/`PisOutr`, `br.CofinsAliq`/`CofinsNt`/`CofinsOutr` — already-computed tax groups, ported 1:1 from the Python SDK's field/alias set.
- `Address` — `RecipientAddress` on `Issue()`, only `.State` read (sets NFe's `idDest`).
- `APIError`, `ConnectionFailedError`, `InvoiceError`.
