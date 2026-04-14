package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name       string
		header     string
		wantToken  string
		wantReason string
	}{
		{name: "missing header", header: "", wantReason: "missing_header"},
		{name: "invalid scheme", header: "Token abc123", wantReason: "invalid_scheme"},
		{name: "missing token", header: "Bearer   ", wantReason: "missing_token"},
		{name: "valid token", header: "Bearer abc123", wantToken: "abc123", wantReason: "ok"},
		{name: "lowercase scheme", header: "bearer abc123", wantToken: "abc123", wantReason: "ok"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/send-push", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			gotToken, gotReason := extractBearerToken(req)
			if gotToken != tt.wantToken {
				t.Fatalf("token = %q, want %q", gotToken, tt.wantToken)
			}
			if gotReason != tt.wantReason {
				t.Fatalf("reason = %q, want %q", gotReason, tt.wantReason)
			}
		})
	}
}

func TestIsBearerAuthorized(t *testing.T) {
	originalToken := inboundBearerToken
	t.Cleanup(func() {
		inboundBearerToken = originalToken
	})

	tests := []struct {
		name           string
		configured     string
		header         string
		wantAuthorized bool
		wantReason     string
	}{
		{name: "disabled", configured: "", header: "", wantAuthorized: false, wantReason: "disabled"},
		{name: "missing header", configured: "secret", header: "", wantAuthorized: false, wantReason: "missing_header"},
		{name: "invalid token", configured: "secret", header: "Bearer nope", wantAuthorized: false, wantReason: "invalid_token"},
		{name: "valid token", configured: "secret", header: "Bearer secret", wantAuthorized: true, wantReason: "authorized"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inboundBearerToken = tt.configured
			req := httptest.NewRequest(http.MethodPost, "/send-push", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			gotAuthorized, gotReason := isBearerAuthorized(req)
			if gotAuthorized != tt.wantAuthorized {
				t.Fatalf("authorized = %v, want %v", gotAuthorized, tt.wantAuthorized)
			}
			if gotReason != tt.wantReason {
				t.Fatalf("reason = %q, want %q", gotReason, tt.wantReason)
			}
		})
	}
}

func TestRequestAuthorizationMatrix(t *testing.T) {
	originalToken := inboundBearerToken
	originalAllowedIPs := allowedIPs
	t.Cleanup(func() {
		inboundBearerToken = originalToken
		allowedIPs = originalAllowedIPs
	})

	tests := []struct {
		name            string
		configuredToken string
		authorization   string
		allowedList     []string
		remoteAddr      string
		wantAuthorized  bool
		wantReason      string
	}{
		{
			name:            "valid bearer with disallowed ip",
			configuredToken: "secret",
			authorization:   "Bearer secret",
			allowedList:     []string{"10.0.0.1"},
			remoteAddr:      "203.0.113.10:1234",
			wantAuthorized:  true,
			wantReason:      "authorized",
		},
		{
			name:            "allowed ip without bearer",
			configuredToken: "secret",
			allowedList:     []string{"203.0.113.10"},
			remoteAddr:      "203.0.113.10:1234",
			wantAuthorized:  true,
			wantReason:      "missing_header",
		},
		{
			name:            "invalid bearer and disallowed ip",
			configuredToken: "secret",
			authorization:   "Bearer nope",
			allowedList:     []string{"10.0.0.1"},
			remoteAddr:      "203.0.113.10:1234",
			wantAuthorized:  false,
			wantReason:      "invalid_token",
		},
		{
			name:            "malformed authorization header",
			configuredToken: "secret",
			authorization:   "Basic abc",
			allowedList:     []string{"10.0.0.1"},
			remoteAddr:      "203.0.113.10:1234",
			wantAuthorized:  false,
			wantReason:      "invalid_scheme",
		},
		{
			name:            "bearer disabled falls back to ip",
			configuredToken: "",
			allowedList:     []string{"203.0.113.10"},
			remoteAddr:      "203.0.113.10:1234",
			wantAuthorized:  true,
			wantReason:      "disabled",
		},
		{
			name:            "bearer enabled without allowlist still requires token",
			configuredToken: "secret",
			authorization:   "Bearer nope",
			allowedList:     nil,
			remoteAddr:      "203.0.113.10:1234",
			wantAuthorized:  false,
			wantReason:      "invalid_token",
		},
		{
			name:            "no auth configured keeps open access",
			configuredToken: "",
			allowedList:     nil,
			remoteAddr:      "203.0.113.10:1234",
			wantAuthorized:  true,
			wantReason:      "open_access",
		},
		{
			name:            "ip denied when bearer disabled",
			configuredToken: "",
			allowedList:     []string{"10.0.0.1"},
			remoteAddr:      "203.0.113.10:1234",
			wantAuthorized:  false,
			wantReason:      "ip_denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inboundBearerToken = tt.configuredToken
			allowedIPs = tt.allowedList

			req := httptest.NewRequest(http.MethodPost, "/send-push", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}

			clientIP := getClientIP(req)
			gotAuthorized, gotReason := isRequestAuthorized(req, clientIP)

			if gotAuthorized != tt.wantAuthorized {
				t.Fatalf("authorized = %v, want %v", gotAuthorized, tt.wantAuthorized)
			}
			if gotReason != tt.wantReason {
				t.Fatalf("reason = %q, want %q", gotReason, tt.wantReason)
			}
		})
	}
}
