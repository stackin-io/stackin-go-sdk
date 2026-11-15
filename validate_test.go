package stackin

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// countingInvoice returns a client whose server counts the requests that
// reach it, so a test can tell a local rejection from a round trip.
func countingInvoice(t *testing.T) (*Invoice, *int) {
	t.Helper()
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{}})
	}))
	t.Cleanup(server.Close)
	return NewInvoice(WithBaseURL(server.URL), WithAPIKey("test-key")), &hits
}

// wantInvoiceError fails unless err is an *InvoiceError carrying message
// and the server was never reached.
func wantInvoiceError(t *testing.T, err error, hits int, message string) {
	t.Helper()
	var invErr *InvoiceError
	if !errors.As(err, &invErr) {
		t.Fatalf("error = %v (%T), want *InvoiceError", err, err)
	}
	if invErr.Message != message {
		t.Errorf("message = %q, want %q", invErr.Message, message)
	}
	if hits != 0 {
		t.Errorf("request reached the network %d time(s), want a local rejection", hits)
	}
}

// nfeAccessKey is a well-formed 44-digit NF-e access key.
const nfeAccessKey = "35240912345678000199550010000000011000000017"

func TestCorrectRejectsAKeyThatIsNotAnNFEKey(t *testing.T) {
	keys := []string{
		"",
		"abc123",
		"3524091234567800019955001000000001100000001",        // 43 digits
		"352409123456780001995500100000000110000000170",      // 45 digits
		"35240912345678000199550010000000011000000017123456", // 50: an NFS-e key
		"3524091234567800019955001000000001100000001A",       // letter after the CNPJ slot
		"35240A12345678000199550010000000011000000017",       // letter before the CNPJ slot
		"352409abc45678000199550010000000011000000017",       // lowercase in the CNPJ slot
		"352409123456789ABCDE550010000000011000000017",       // letters at positions 19-20
		"35240912345678000199550010000000011000000017\n",     // trailing newline
	}
	for _, key := range keys {
		inv, hits := countingInvoice(t)

		_, err := inv.Correct(key, NFE, "Transportadora corrigida para Rapido Ltda")

		wantInvoiceError(t, err, *hits,
			"Correct is NFe only. NFSe has no correction letter — cancel and reissue instead.")
	}
}

func TestCorrectSendsAnNFEAccessKey(t *testing.T) {
	// 6 digits + 12 [0-9A-Z] (alphanumeric CNPJ) + 26 digits.
	alphanumeric := "352607" + "12ABC3450001" + strings.Repeat("9", 26)
	if len(alphanumeric) != 44 {
		t.Fatalf("test key has %d characters, want 44", len(alphanumeric))
	}
	for _, key := range []string{nfeAccessKey, alphanumeric} {
		inv, hits := countingInvoice(t)

		if _, err := inv.Correct(key, NFE, "Transportadora corrigida para Rapido Ltda"); err != nil {
			t.Fatalf("Correct(%q) = %v, want nil", key, err)
		}
		if *hits != 1 {
			t.Errorf("Correct(%q) hits = %d, want 1", key, *hits)
		}
	}
}

func TestCorrectLeavesAnNFSEDocumentTypeToTheAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "correction letters are NF-e only"})
	}))
	defer server.Close()
	inv := NewInvoice(WithBaseURL(server.URL), WithAPIKey("test-key"))

	_, err := inv.Correct(nfeAccessKey, NFSE, "Transportadora corrigida para Rapido Ltda")

	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("error = %v (%T), want *APIError with status 409", err, err)
	}
}
