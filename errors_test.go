package stackin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAPIErrorMessage(t *testing.T) {
	err := &APIError{StatusCode: 404, Detail: "not found"}

	if got, want := err.Error(), "[404] not found"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestConnectionFailedErrorMessage(t *testing.T) {
	err := &ConnectionFailedError{Message: "could not connect"}

	if got, want := err.Error(), "could not connect"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func jsonServer(t *testing.T, status int, header http.Header, body string, seen *[]*http.Request) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if seen != nil {
			*seen = append(*seen, r.Clone(r.Context()))
		}
		for key, values := range header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func errorFrom(t *testing.T, status int, header http.Header, body string) error {
	t.Helper()
	url := jsonServer(t, status, header, body, nil)
	_, err := NewInvoice(WithBaseURL(url)).Consult("abc", NFE)
	if err == nil {
		t.Fatal("want an error")
	}
	return err
}

func assertWrapsAPIError(t *testing.T, err error, status int) {
	t.Helper()
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("errors.As(%T, *APIError) = false", err)
	}
	if apiErr.StatusCode != status {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, status)
	}
}

func TestAny402IsPaymentRequired(t *testing.T) {
	err := errorFrom(t, http.StatusPaymentRequired, nil, `{"detail":"No credit left."}`)

	var payment *PaymentRequiredError
	if !errors.As(err, &payment) {
		t.Fatalf("error = %T, want *PaymentRequiredError", err)
	}
	if payment.Detail != "No credit left." {
		t.Errorf("Detail = %q", payment.Detail)
	}
	assertWrapsAPIError(t, err, http.StatusPaymentRequired)
}

func TestAny429IsRateLimited(t *testing.T) {
	err := errorFrom(t, http.StatusTooManyRequests, http.Header{"Retry-After": {"30"}}, `{"detail":"Too many requests"}`)

	var limited *RateLimitedError
	if !errors.As(err, &limited) {
		t.Fatalf("error = %T, want *RateLimitedError", err)
	}
	if limited.RetryAfter != 30 || limited.RetryAt != nil || limited.Code != "" {
		t.Errorf("RateLimitedError = %+v, want RetryAfter 30 and nothing else", limited)
	}
	assertWrapsAPIError(t, err, http.StatusTooManyRequests)
}

func TestA429WithoutRetryAfterLeavesItZero(t *testing.T) {
	err := errorFrom(t, http.StatusTooManyRequests, nil, `not json`)

	var limited *RateLimitedError
	if !errors.As(err, &limited) {
		t.Fatalf("error = %T, want *RateLimitedError", err)
	}
	if limited.RetryAfter != 0 {
		t.Errorf("RetryAfter = %d, want 0", limited.RetryAfter)
	}
}

func TestA429RetryAfterAsAnHTTPDate(t *testing.T) {
	at := time.Now().Add(2 * time.Minute).UTC().Format(http.TimeFormat)
	err := errorFrom(t, http.StatusTooManyRequests, http.Header{"Retry-After": {at}}, `{"detail":"slow down"}`)

	var limited *RateLimitedError
	if !errors.As(err, &limited) {
		t.Fatalf("error = %T, want *RateLimitedError", err)
	}
	if limited.RetryAfter < 60 || limited.RetryAfter > 120 {
		t.Errorf("RetryAfter = %d, want about 120", limited.RetryAfter)
	}
}

func TestClientInactiveIsTyped(t *testing.T) {
	err := errorFrom(t, http.StatusUnprocessableEntity, nil, `{"detail":{
		"code":"CLIENT_INACTIVE","message":"This client is deactivated.","client_id":"c-9"}}`)

	var inactive *ClientInactiveError
	if !errors.As(err, &inactive) {
		t.Fatalf("error = %T, want *ClientInactiveError", err)
	}
	if inactive.ClientID != "c-9" || inactive.Message != "This client is deactivated." {
		t.Errorf("ClientInactiveError = %+v", inactive)
	}
	assertWrapsAPIError(t, err, http.StatusUnprocessableEntity)
}

func TestClientNotFoundIsTyped(t *testing.T) {
	err := errorFrom(t, http.StatusNotFound, nil, `{"detail":{
		"code":"CLIENT_NOT_FOUND","message":"No such client.","client_id":"c-9"}}`)

	var missing *ClientNotFoundError
	if !errors.As(err, &missing) {
		t.Fatalf("error = %T, want *ClientNotFoundError", err)
	}
	if missing.ClientID != "c-9" || missing.Message != "No such client." {
		t.Errorf("ClientNotFoundError = %+v", missing)
	}
	assertWrapsAPIError(t, err, http.StatusNotFound)
}

func TestAPlain404AndAnOther422StayAPIError(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
	}{
		{http.StatusNotFound, `{"detail":"invoice not found"}`},
		{http.StatusNotFound, `{"detail":{"code":"SOMETHING_ELSE"}}`},
		{http.StatusUnprocessableEntity, `{"detail":{"code":"CITY_NOT_FOUND"}}`},
	} {
		err := errorFrom(t, tc.status, nil, tc.body)
		if _, ok := err.(*APIError); !ok {
			t.Errorf("%d %s: error = %T, want *APIError", tc.status, tc.body, err)
		}
	}
}

func TestA429WithoutTheHeaderDerivesRetryAfterFromRetryAt(t *testing.T) {
	at := time.Now().Add(10 * time.Minute).UTC().Format(time.RFC3339)
	err := errorFrom(t, http.StatusTooManyRequests, nil,
		`{"detail":"wait","code":"DISTRIBUTION_COOLDOWN","retry_at":"`+at+`"}`)

	var limited *RateLimitedError
	if !errors.As(err, &limited) {
		t.Fatalf("error = %T, want *RateLimitedError", err)
	}
	if limited.RetryAfter < 540 || limited.RetryAfter > 600 {
		t.Errorf("RetryAfter = %d, want about 600", limited.RetryAfter)
	}
}
