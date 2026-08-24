package security

import (
	"strings"
	"testing"
)

func TestBuildCSP_ExtraImgSrc(t *testing.T) {
	csp := buildCSP(SecurityHeadersConfig{
		ExtraImgSrc: []string{"https://cdn.be.rentandtravel.de"},
	})

	if !strings.Contains(csp, "img-src 'self' data: https://cdn.be.rentandtravel.de") {
		t.Errorf("CSP missing plugin img-src host: %s", csp)
	}
}

func TestBuildCSP_NoExtraImgSrc(t *testing.T) {
	csp := buildCSP(SecurityHeadersConfig{})

	if !strings.Contains(csp, "img-src 'self' data:;") {
		t.Errorf("CSP img-src should be unchanged without plugins: %s", csp)
	}
}
