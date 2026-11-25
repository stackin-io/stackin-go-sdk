package stackin

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func languageServer(t *testing.T, reply string) (string, *http.Header, *map[string]any) {
	t.Helper()
	header := http.Header{}
	body := map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header = r.Header.Clone()
		clear(body)
		if raw, _ := io.ReadAll(r.Body); len(raw) > 0 {
			_ = json.Unmarshal(raw, &body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(server.Close)
	return server.URL, &header, &body
}

func TestNBSSuggestionReadsTheNewFields(t *testing.T) {
	client, _ := newReference(t, map[string]any{
		"service_code": "010700",
		"suggestions": []any{map[string]any{
			"nbs": "115021000", "description": "Suporte técnico", "operation_indicator": "100301",
			"ibs_cbs_classification": "200034", "classification_name": "Redução",
			"classification_cst": "200", "reduction_ibs": 60.0, "reduction_cbs": nil,
			"legal_reference": "Art. 129 da LC 214/2025", "incidence": "Local do fornecedor",
			"operation_variants": map[string]any{"free_abroad": "100401", "onerous_domestic": "100301"},
		}},
		"source": map[string]any{"path": "p", "date": nil},
	})

	got, err := client.NBSSuggestions("1.07")
	if err != nil {
		t.Fatal(err)
	}
	row := got.Suggestions[0]
	if row.ClassificationCST == nil || *row.ClassificationCST != "200" {
		t.Errorf("ClassificationCST = %v", row.ClassificationCST)
	}
	if row.ReductionIBS == nil || *row.ReductionIBS != 60 || row.ReductionCBS != nil {
		t.Errorf("reductions = %v, %v", row.ReductionIBS, row.ReductionCBS)
	}
	if *row.LegalReference != "Art. 129 da LC 214/2025" || *row.Incidence != "Local do fornecedor" {
		t.Errorf("row = %+v", row)
	}
	if row.OperationVariants["free_abroad"] != "100401" {
		t.Errorf("OperationVariants = %v", row.OperationVariants)
	}
}

func TestNBSSuggestionLeavesTheNewFieldsEmptyWhenAbsent(t *testing.T) {
	client, _ := newReference(t, map[string]any{
		"service_code": "010700",
		"suggestions":  []any{map[string]any{"nbs": "115021000"}},
		"source":       map[string]any{"path": "p", "date": nil},
	})

	got, err := client.NBSSuggestions("1.07")
	if err != nil {
		t.Fatal(err)
	}
	row := got.Suggestions[0]
	if row.ClassificationCST != nil || row.ReductionIBS != nil || row.OperationVariants != nil {
		t.Errorf("row = %+v", row)
	}
}

func TestIssueSendsTheContingencyMomentAndReason(t *testing.T) {
	req := nfeRequest(nfeItem())
	req.Contingency = true
	started := time.Date(2026, 11, 25, 9, 0, 0, 0, time.FixedZone("BRT", -3*60*60))
	req.ContingencyStartedAt = &started
	req.ContingencyReason = "SEFAZ authorizer unavailable"

	raw, _, err := captureIssue(t, req)
	if err != nil {
		t.Fatal(err)
	}
	body := sentFields(t, raw)
	if body["contingency"] != true ||
		body["contingency_started_at"] != "2026-11-25T09:00:00-03:00" ||
		body["contingency_reason"] != "SEFAZ authorizer unavailable" {
		t.Errorf("body = %v", body)
	}
}

func TestIssueLeavesTheContingencyMomentAndReasonOutWhenUnset(t *testing.T) {
	req := nfeRequest(nfeItem())
	req.Contingency = true

	raw, _, err := captureIssue(t, req)
	if err != nil {
		t.Fatal(err)
	}
	body := sentFields(t, raw)
	if _, ok := body["contingency_started_at"]; ok {
		t.Error("contingency_started_at sent")
	}
	if _, ok := body["contingency_reason"]; ok {
		t.Error("contingency_reason sent")
	}
}

func TestIssueRefusesAZeroContingencyMoment(t *testing.T) {
	req := nfeRequest(nfeItem())
	req.Contingency = true
	zero := time.Time{}
	req.ContingencyStartedAt = &zero
	expectRefusal(t, req, "contingency_started_at must be a date-time")
}

func TestClassifySendsTheLanguageAndTheOriginAnswer(t *testing.T) {
	url, header, body := languageServer(t, `{"result":{}}`)
	_, err := NewInvoice(WithBaseURL(url), WithAPIKey("k")).Classify(ClassifyRequest{
		Description:    "Camiseta de algodao",
		DocumentType:   NFE,
		Answers:        map[string]any{"origin": "resale"},
		AcceptLanguage: "en",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := header.Get("Accept-Language"); got != "en" {
		t.Errorf("Accept-Language = %q", got)
	}
	if answers, _ := (*body)["answers"].(map[string]any); answers["origin"] != "resale" {
		t.Errorf("body = %v", *body)
	}
	if _, ok := (*body)["AcceptLanguage"]; ok {
		t.Error("language reached the body")
	}
}

func TestReceivedSendsTheLanguageWhenAsked(t *testing.T) {
	url, header, _ := languageServer(t, `{"data":[],"sync":{"status":"ok"}}`)
	inv := NewInvoice(WithBaseURL(url), WithAPIKey("k"))

	page, err := inv.Received(5, 0, WithAcceptLanguage("es"))
	if err != nil {
		t.Fatal(err)
	}
	if got := header.Get("Accept-Language"); got != "es" {
		t.Errorf("Accept-Language = %q", got)
	}
	if sync, _ := page["sync"].(map[string]any); sync["status"] != "ok" {
		t.Errorf("page = %v", page)
	}

	if _, err := inv.Received(5, 0); err != nil {
		t.Fatal(err)
	}
	if got := header.Get("Accept-Language"); got != "" {
		t.Errorf("Accept-Language = %q, want none", got)
	}
}
