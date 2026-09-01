package security

import "testing"

func TestBuildCSPReportURI(t *testing.T) {
	tests := []struct {
		name      string
		cfg       SecurityHeadersConfig
		wantHas   []string
		wantNotIn []string
	}{
		{
			name:      "no report-uri when unset",
			cfg:       SecurityHeadersConfig{},
			wantNotIn: []string{"report-uri"},
		},
		{
			name:    "relative report-uri is appended verbatim",
			cfg:     SecurityHeadersConfig{CSPReportURI: "/csp?key=abc"},
			wantHas: []string{"report-uri /csp?key=abc"},
		},
		{
			name: "error tracking origins land in the right directives",
			cfg: SecurityHeadersConfig{
				ErrorTrackingOrigin:    "https://glitchtip.example",
				ErrorTrackingSDKOrigin: "https://cdn.example",
			},
			wantHas: []string{
				"connect-src 'self' https://glitchtip.example",
				"script-src 'self' https://cdn.example",
			},
		},
		{
			name: "a same-origin SDK path adds nothing",
			cfg: SecurityHeadersConfig{
				ErrorTrackingOrigin:    "",
				ErrorTrackingSDKOrigin: "",
			},
			wantHas:   []string{"script-src 'self';"},
			wantNotIn: []string{"script-src 'self' ;"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildCSP(tt.cfg)
			for _, want := range tt.wantHas {
				if !contains(got, want) {
					t.Errorf("buildCSP() = %q, missing %q", got, want)
				}
			}
			for _, no := range tt.wantNotIn {
				if contains(got, no) {
					t.Errorf("buildCSP() = %q, should not contain %q", got, no)
				}
			}
		})
	}
}

// The ARCON policy is a separate code path and has silently drifted from the
// default one before; it gets the same treatment.
func TestBuildArconCSPCarriesReportURI(t *testing.T) {
	got := buildArconCSP(SecurityHeadersConfig{
		CSPReportURI:        "/csp",
		ErrorTrackingOrigin: "https://glitchtip.example",
	})
	if !contains(got, "report-uri /csp") {
		t.Errorf("arcon CSP lost the report-uri: %q", got)
	}
	if !contains(got, "connect-src 'self' https://glitchtip.example") {
		t.Errorf("arcon CSP does not allow the error tracker: %q", got)
	}
}

func TestBuildHSTS(t *testing.T) {
	tests := []struct {
		name string
		cfg  SecurityHeadersConfig
		want string
	}{
		{
			// The default matters more than the options. includeSubDomains
			// reaches subdomains nobody listed, preload is removable only in
			// months — neither may arrive by accident.
			name: "plain by default",
			cfg:  SecurityHeadersConfig{},
			want: "max-age=31536000",
		},
		{
			name: "subdomains on request",
			cfg:  SecurityHeadersConfig{HSTSIncludeSubdomains: true},
			want: "max-age=31536000; includeSubDomains",
		},
		{
			name: "preload on request",
			cfg:  SecurityHeadersConfig{HSTSIncludeSubdomains: true, HSTSPreload: true},
			want: "max-age=31536000; includeSubDomains; preload",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildHSTS(tt.cfg); got != tt.want {
				t.Errorf("buildHSTS() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOriginOf(t *testing.T) {
	tests := []struct{ in, want string }{
		{"https://key@glitchtip.example/1", "https://glitchtip.example"},
		{"/static/js/sentry.min.js", ""},
		{"", ""},
		{"not a url", ""},
	}
	for _, tt := range tests {
		if got := OriginOf(tt.in); got != tt.want {
			t.Errorf("OriginOf(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func contains(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) &&
		indexOf(haystack, needle) >= 0
}

func indexOf(h, n string) int {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}
