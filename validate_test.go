package stackin

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
		"3524091234567800019955001000000001100000001A",
	}
	for _, key := range keys {
		inv, hits := countingInvoice(t)

		_, err := inv.Correct(key, NFE, "Transportadora corrigida para Rapido Ltda")

		wantInvoiceError(t, err, *hits,
			"Correct is NFe only. NFSe has no correction letter — cancel and reissue instead.")
	}
}

func TestCorrectSendsA44DigitKey(t *testing.T) {
	inv, hits := countingInvoice(t)

	if _, err := inv.Correct(nfeAccessKey, NFE, "Transportadora corrigida para Rapido Ltda"); err != nil {
		t.Fatalf("Correct() = %v, want nil", err)
	}
	if *hits != 1 {
		t.Errorf("hits = %d, want 1", *hits)
	}
}
