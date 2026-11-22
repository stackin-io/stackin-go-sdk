package stackin

import (
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/stackin-io/stackin-go-sdk/br"
)

const itemID = "5c1e9a7d-3b24-4f60-8a1d-2e7b9c0d4f13"

func newItems(t *testing.T, status int, reply any, opts ...Option) (*Items, *[]seenRequest) {
	t.Helper()
	baseURL, seen := captureServer(t, status, reply)
	return NewItems(append([]Option{WithBaseURL(baseURL), WithAPIKey("k")}, opts...)...), seen
}

func TestItemsListSendsNoParamsWithoutAFilter(t *testing.T) {
	items, seen := newItems(t, http.StatusOK, map[string]any{})

	if _, err := items.List(ItemQuery{}); err != nil {
		t.Fatal(err)
	}

	got := (*seen)[0]
	if got.Method != http.MethodGet || got.Path != "/api/v1/items" {
		t.Errorf("%s %s", got.Method, got.Path)
	}
	if len(got.Query) != 0 || got.Raw != "" {
		t.Errorf("query %v body %q", got.Query, got.Raw)
	}
}

func TestItemsListSendsEveryFilterByItsAPIName(t *testing.T) {
	items, seen := newItems(t, http.StatusOK, map[string]any{})
	inactive := false

	_, err := items.List(ItemQuery{
		Kind: "product", Search: "cafe", Source: "nuvemshop", IsActive: &inactive,
		Limit: 10, Offset: 20, SortBy: "unit_price", OrderBy: "desc",
	})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string][]string{
		"kind": {"product"}, "search": {"cafe"}, "source": {"nuvemshop"},
		"is_active": {"false"}, "limit": {"10"}, "offset": {"20"},
		"sort_by": {"unit_price"}, "order_by": {"desc"},
	}
	if got := (*seen)[0].Query; !reflect.DeepEqual(got, want) {
		t.Errorf("query %v", got)
	}
}

func TestItemsListReturnsTheEnvelopeAsIs(t *testing.T) {
	items, _ := newItems(t, http.StatusOK, map[string]any{"items": []any{}, "total": 0})

	page, err := items.List(ItemQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if page["total"] != float64(0) {
		t.Errorf("page %v", page)
	}
}

func TestItemsGetEscapesTheID(t *testing.T) {
	items, seen := newItems(t, http.StatusOK, map[string]any{"id": itemID})

	if _, err := items.Get("a/b"); err != nil {
		t.Fatal(err)
	}
	if got := (*seen)[0].Path; got != "/api/v1/items/a%2Fb" {
		t.Errorf("path %s", got)
	}
	for _, bad := range []string{"", ".", ".."} {
		if _, err := items.Get(bad); err == nil {
			t.Errorf("Get(%q) accepted", bad)
		}
	}
}

func TestItemsUpsertReportsCreatedOn201(t *testing.T) {
	items, seen := newItems(t, http.StatusCreated, map[string]any{"id": itemID})

	price, no := 18.9, false
	item, created, err := items.Upsert(ItemFields{
		Kind: "product", Description: "Cafe", ExternalReference: "ns-1",
		Source: "nuvemshop", UnitPrice: &price, ISSRetained: &no,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !created || item["id"] != itemID {
		t.Errorf("created %v item %v", created, item)
	}
	got := (*seen)[0]
	if got.Method != http.MethodPost || got.Path != "/api/v1/items" {
		t.Errorf("%s %s", got.Method, got.Path)
	}
	want := map[string]any{
		"kind": "product", "description": "Cafe", "external_reference": "ns-1",
		"source": "nuvemshop", "unit_price": "18.9", "iss_retained": false,
	}
	if !reflect.DeepEqual(got.Body, want) {
		t.Errorf("body %v", got.Body)
	}
}

func TestItemsUpsertReportsUpdatedOn200(t *testing.T) {
	items, _ := newItems(t, http.StatusOK, map[string]any{"id": itemID})

	item, created, err := items.Upsert(ItemFields{Kind: "service", Description: "Consultoria"})
	if err != nil {
		t.Fatal(err)
	}
	if created || item["id"] != itemID {
		t.Errorf("created %v item %v", created, item)
	}
}

func TestItemsCreateReturnsTheItem(t *testing.T) {
	items, _ := newItems(t, http.StatusCreated, map[string]any{"id": itemID})

	yes := true
	item, err := items.Create(ItemFields{Kind: "service", Description: "Consultoria", ServiceCode: "1.07", ISSRetained: &yes})
	if err != nil {
		t.Fatal(err)
	}
	if item["id"] != itemID {
		t.Errorf("item %v", item)
	}
}

func TestItemsUpdateSendsOnlyTheFieldsSetAndNullToClear(t *testing.T) {
	items, seen := newItems(t, http.StatusOK, map[string]any{})
	price, retained := 0.5, false

	_, err := items.Update(itemID, ItemChanges{
		Description: text("Novo"), Notes: text(""), UnitPrice: &price, ISSRetained: &retained,
	})
	if err != nil {
		t.Fatal(err)
	}

	got := (*seen)[0]
	if got.Method != http.MethodPatch || got.Path != "/api/v1/items/"+itemID {
		t.Errorf("%s %s", got.Method, got.Path)
	}
	want := map[string]any{"description": "Novo", "notes": nil, "unit_price": "0.5", "iss_retained": false}
	if !reflect.DeepEqual(got.Body, want) {
		t.Errorf("body %v", got.Body)
	}
}

func TestItemsUpdateWithNoChangesSendsAnEmptyObject(t *testing.T) {
	items, seen := newItems(t, http.StatusOK, map[string]any{})

	if _, err := items.Update(itemID, ItemChanges{}); err != nil {
		t.Fatal(err)
	}
	if got := (*seen)[0].Raw; got != "{}" {
		t.Errorf("body %q", got)
	}
}

func TestItemsDeactivateDeletesAndDecodesNothing(t *testing.T) {
	items, seen := newItems(t, http.StatusNoContent, nil)

	if err := items.Deactivate(itemID); err != nil {
		t.Fatal(err)
	}
	got := (*seen)[0]
	if got.Method != http.MethodDelete || got.Path != "/api/v1/items/"+itemID || got.Raw != "" {
		t.Errorf("%s %s %q", got.Method, got.Path, got.Raw)
	}
	if err := items.Deactivate(".."); err == nil {
		t.Error("dot segment accepted")
	}
}

func TestItemsReactivatePatchesIsActive(t *testing.T) {
	items, seen := newItems(t, http.StatusOK, map[string]any{"id": itemID, "is_active": true})

	item, err := items.Reactivate(itemID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual((*seen)[0].Body, map[string]any{"is_active": true}) || item["is_active"] != true {
		t.Errorf("body %v item %v", (*seen)[0].Body, item)
	}
}

func TestItemsSendTheCompanyAndForCompanyBindsACopy(t *testing.T) {
	items, seen := newItems(t, http.StatusOK, map[string]any{}, WithCompany("11222333000181"))
	other := items.ForCompany("99887766000155")

	if _, err := items.List(ItemQuery{}); err != nil {
		t.Fatal(err)
	}
	if _, err := other.Get(itemID); err != nil {
		t.Fatal(err)
	}
	if _, err := items.Get(itemID); err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"11222333000181", "99887766000155", "11222333000181"} {
		if got := (*seen)[i].Header.Get("X-Stackin-Company"); got != want {
			t.Errorf("call %d company %q", i, got)
		}
	}
}

func itemErrorReply(code string, extra map[string]any) map[string]any {
	detail := map[string]any{"code": code, "message": "m", "item_id": itemID}
	for key, value := range extra {
		detail[key] = value
	}
	return map[string]any{"detail": detail}
}

func TestItemExistsIsATypedErrorWrappingAPIError(t *testing.T) {
	items, _ := newItems(t, http.StatusConflict, itemErrorReply("ITEM_EXISTS", map[string]any{"field": "code"}))

	_, _, err := items.Upsert(ItemFields{Kind: "product", Description: "Cafe", Code: "CAF"})

	var exists *ItemAlreadyExistsError
	if !errors.As(err, &exists) {
		t.Fatalf("got %T", err)
	}
	if exists.ItemID != itemID || exists.Field != "code" || exists.StatusCode != http.StatusConflict {
		t.Errorf("%+v", exists)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Error("errors.As does not find the *APIError")
	}
}

func TestItemNotFoundIsATypedError(t *testing.T) {
	items, _ := newItems(t, http.StatusNotFound, itemErrorReply("ITEM_NOT_FOUND", nil))

	_, err := items.Get(itemID)

	var missing *ItemNotFoundError
	if !errors.As(err, &missing) || missing.ItemID != itemID {
		t.Fatalf("got %T", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Error("errors.As does not find the *APIError")
	}
}

func TestIssueRefusalsAreTypedItemErrors(t *testing.T) {
	cases := []struct {
		code  string
		check func(error) bool
	}{
		{"ITEM_INACTIVE", func(err error) bool {
			var e *ItemInactiveError
			return errors.As(err, &e) && e.ItemID == itemID
		}},
		{"ITEM_KIND_MISMATCH", func(err error) bool {
			var e *ItemKindMismatchError
			return errors.As(err, &e) && e.ItemID == itemID && e.Index == 2
		}},
		{"ITEM_TAX_REGIME_MISMATCH", func(err error) bool {
			var e *ItemTaxRegimeMismatchError
			return errors.As(err, &e) && e.ItemID == itemID && e.Index == 2
		}},
		{"ITEM_TAX_REQUIRED", func(err error) bool {
			var e *ItemTaxRequiredError
			return errors.As(err, &e) && e.ItemID == itemID && e.Index == 2
		}},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			baseURL, _ := captureServer(t, http.StatusUnprocessableEntity,
				itemErrorReply(tc.code, map[string]any{"index": 2}))
			_, err := NewInvoice(WithBaseURL(baseURL), WithAPIKey("k")).Issue(IssueRequest{
				DocumentType: NFSE, ClientName: "ACME", TaxID: "11222333000181",
				Items: []br.Product{clientNFSEItem(false)},
			})
			if !tc.check(err) {
				t.Fatalf("got %T", err)
			}
			var apiErr *APIError
			if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnprocessableEntity {
				t.Error("errors.As does not find the *APIError")
			}
		})
	}
}

func TestAnotherItemCodeOrStatusStaysAPlainAPIError(t *testing.T) {
	items, _ := newItems(t, http.StatusUnprocessableEntity, map[string]any{"detail": "boom"})

	_, err := items.Get(itemID)
	if _, ok := err.(*APIError); !ok {
		t.Fatalf("got %T", err)
	}
}

func sentLine(t *testing.T, req IssueRequest) map[string]any {
	t.Helper()
	body, err := issueSeen(t, req)
	if err != nil {
		t.Fatal(err)
	}
	return body["items"].([]any)[0].(map[string]any)
}

func TestIssueLineWithItemIDSendsItAndOmitsTheDescription(t *testing.T) {
	line := sentLine(t, IssueRequest{
		DocumentType: NFE, ClientName: "ACME", TaxID: "11222333000181",
		RecipientAddress: &Address{Street: "Rua A", Number: "1", Neighborhood: "Centro", City: "Sao Paulo", State: "SP", ZipCode: "01001000"},
		Items:            []br.Product{{ItemID: itemID, Quantity: 2}},
	})

	if line["item_id"] != itemID {
		t.Errorf("item_id %v", line["item_id"])
	}
	for _, key := range []string{"description", "amount", "unit_price", "tax_retained"} {
		if _, ok := line[key]; ok {
			t.Errorf("%s sent: %v", key, line[key])
		}
	}
	product := line["product"].(map[string]any)
	if _, ok := product["unit"]; ok {
		t.Errorf("unit defaulted: %v", product["unit"])
	}
	if _, ok := product["br"]; ok {
		t.Errorf("br sent: %v", product["br"])
	}
	if product["quantity"] != float64(2) {
		t.Errorf("quantity %v", product["quantity"])
	}
}

func TestIssueLineWithItemIDSendsTheFieldsSetSoTheyWin(t *testing.T) {
	line := sentLine(t, IssueRequest{
		DocumentType: NFSE, ClientID: clientID,
		Items: []br.Product{{ItemID: itemID, Description: "Mine", Unit: "KG", Amount: 5, TaxRetained: ptr(true)}},
	})

	if line["description"] != "Mine" || line["amount"] != "5" || line["tax_retained"] != true {
		t.Errorf("line %v", line)
	}
	if unit := line["product"].(map[string]any)["unit"]; unit != "KG" {
		t.Errorf("unit %v", unit)
	}
}

func TestIssueNFEWithItemIDDoesNotRequireNCMAndCFOPLocally(t *testing.T) {
	_, err := issueSeen(t, IssueRequest{
		DocumentType: NFE, ClientName: "ACME", TaxID: "11222333000181",
		RecipientAddress: &Address{Street: "Rua A", Number: "1", Neighborhood: "Centro", City: "Sao Paulo", State: "SP", ZipCode: "01001000"},
		Items:            []br.Product{{ItemID: itemID}},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIssueNFEWithoutItemIDStillRequiresNCM(t *testing.T) {
	_, err := issueSeen(t, IssueRequest{
		DocumentType: NFE, ClientName: "ACME", TaxID: "11222333000181",
		Items: []br.Product{{Description: "x", Amount: 1}},
	})
	var invoiceErr *InvoiceError
	if !errors.As(err, &invoiceErr) {
		t.Fatalf("got %v", err)
	}
}

func TestIssueLineWithoutItemIDSendsTheSameBody(t *testing.T) {
	line := sentLine(t, IssueRequest{
		DocumentType: NFSE, ClientName: "ACME", TaxID: "11222333000181",
		Items: []br.Product{clientNFSEItem(false)},
	})

	if _, ok := line["item_id"]; ok {
		t.Error("item_id sent")
	}
	if line["description"] != "svc" || line["tax_retained"] != false {
		t.Errorf("line %v", line)
	}
	if unit := line["product"].(map[string]any)["unit"]; unit != "UN" {
		t.Errorf("unit %v", unit)
	}
}

func TestItemFieldsSendZeroAndFalseButNotUnset(t *testing.T) {
	zero, no := 0.0, false
	items, seen := newItems(t, http.StatusOK, map[string]any{"id": itemID})

	if _, _, err := items.Upsert(ItemFields{
		Kind: "service", Description: "Consultoria", UnitPrice: &zero, ISSRetained: &no,
	}); err != nil {
		t.Fatal(err)
	}
	sent := (*seen)[0].Body
	if v, ok := sent["unit_price"]; !ok || v != "0" {
		t.Errorf("unit_price 0 not sent: %v", sent)
	}
	if v, ok := sent["iss_retained"]; !ok || v != false {
		t.Errorf("iss_retained false not sent: %v", sent)
	}

	if _, _, err := items.Upsert(ItemFields{Kind: "service", Description: "Consultoria"}); err != nil {
		t.Fatal(err)
	}
	sent = (*seen)[1].Body
	for _, key := range []string{"unit_price", "iss_retained"} {
		if _, ok := sent[key]; ok {
			t.Errorf("%s sent although unset: %v", key, sent)
		}
	}
}

func TestItemFieldsSendAUnitPriceTextAsIs(t *testing.T) {
	items, seen := newItems(t, http.StatusCreated, map[string]any{"id": itemID})

	if _, _, err := items.Upsert(ItemFields{
		Kind: "product", Description: "Cafe", UnitPriceText: "18.9000000001",
	}); err != nil {
		t.Fatal(err)
	}
	if got := (*seen)[0].Body["unit_price"]; got != "18.9000000001" {
		t.Errorf("unit_price %v", got)
	}
}

func TestItemFieldsRefuseABadUnitPriceText(t *testing.T) {
	items, seen := newItems(t, http.StatusCreated, map[string]any{"id": itemID})

	for _, bad := range []string{"1,5", "-1", "1e3", "0.12345678901", " 1"} {
		_, _, err := items.Upsert(ItemFields{Kind: "product", Description: "x", UnitPriceText: bad})
		if err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	if len(*seen) != 0 {
		t.Errorf("%d requests went out", len(*seen))
	}
}

func TestItemChangesSendAPriceTextAndClearIt(t *testing.T) {
	items, seen := newItems(t, http.StatusOK, map[string]any{"id": itemID})
	text := "1234.5678901234"
	empty := ""

	if _, err := items.Update(itemID, ItemChanges{UnitPriceText: &text}); err != nil {
		t.Fatal(err)
	}
	if _, err := items.Update(itemID, ItemChanges{UnitPriceText: &empty}); err != nil {
		t.Fatal(err)
	}
	bad := "1,5"
	if _, err := items.Update(itemID, ItemChanges{UnitPriceText: &bad}); err == nil {
		t.Error("bad text accepted")
	}
	if got := (*seen)[0].Body["unit_price"]; got != text {
		t.Errorf("unit_price %v", got)
	}
	if v, ok := (*seen)[1].Body["unit_price"]; !ok || v != nil {
		t.Errorf("unit_price not cleared: %v", (*seen)[1].Body)
	}
	if len(*seen) != 2 {
		t.Errorf("%d requests", len(*seen))
	}
}

func TestIssueLineWithItemIDSendsTaxRetainedFalseWhenSet(t *testing.T) {
	no := false
	line := sentLine(t, IssueRequest{
		DocumentType: NFSE, ClientID: clientID,
		Items: []br.Product{{ItemID: itemID, TaxRetained: &no}},
	})
	if v, ok := line["tax_retained"]; !ok || v != false {
		t.Errorf("tax_retained false not sent: %v", line)
	}

	line = sentLine(t, IssueRequest{
		DocumentType: NFSE, ClientID: clientID,
		Items: []br.Product{{ItemID: itemID}},
	})
	if _, ok := line["tax_retained"]; ok {
		t.Errorf("tax_retained sent although unset: %v", line)
	}
}
