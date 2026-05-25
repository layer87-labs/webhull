package i18n

import (
	"context"
	"testing"
)

func TestNewService(t *testing.T) {
	svc := NewService("de", []string{"de", "en"})
	if svc.Default() != "de" {
		t.Errorf("expected default 'de', got %q", svc.Default())
	}
	supported := svc.Supported()
	if len(supported) != 2 {
		t.Fatalf("expected 2 supported languages, got %d", len(supported))
	}
}

func TestDetectFromHeader_Empty(t *testing.T) {
	svc := NewService("de", []string{"de", "en"})
	if lang := svc.DetectFromHeader(""); lang != "de" {
		t.Errorf("expected default 'de' for empty header, got %q", lang)
	}
}

func TestDetectFromHeader_ExactMatch(t *testing.T) {
	svc := NewService("de", []string{"de", "en"})
	if lang := svc.DetectFromHeader("en"); lang != "en" {
		t.Errorf("expected 'en', got %q", lang)
	}
}

func TestDetectFromHeader_WithRegionAndQuality(t *testing.T) {
	svc := NewService("de", []string{"de", "en"})
	lang := svc.DetectFromHeader("en-US,en;q=0.9,de-DE;q=0.8,de;q=0.7")
	if lang != "en" {
		t.Errorf("expected 'en' from 'en-US,...', got %q", lang)
	}
}

func TestDetectFromHeader_FallsBackToDefault(t *testing.T) {
	svc := NewService("de", []string{"de", "en"})
	lang := svc.DetectFromHeader("fr-FR,fr;q=0.9,es;q=0.8")
	if lang != "de" {
		t.Errorf("expected default 'de' for unsupported languages, got %q", lang)
	}
}

func TestDetectFromHeader_FirstSupported(t *testing.T) {
	svc := NewService("de", []string{"de", "en", "fr"})
	lang := svc.DetectFromHeader("ja,fr;q=0.9,en;q=0.8")
	if lang != "fr" {
		t.Errorf("expected 'fr' as first supported, got %q", lang)
	}
}

func TestDetectFromCookie_Valid(t *testing.T) {
	svc := NewService("de", []string{"de", "en"})
	lang, ok := svc.DetectFromCookie("en")
	if !ok {
		t.Error("expected ok=true for supported cookie value")
	}
	if lang != "en" {
		t.Errorf("expected 'en', got %q", lang)
	}
}

func TestDetectFromCookie_Invalid(t *testing.T) {
	svc := NewService("de", []string{"de", "en"})
	lang, ok := svc.DetectFromCookie("fr")
	if ok {
		t.Error("expected ok=false for unsupported cookie value")
	}
	if lang != "de" {
		t.Errorf("expected default 'de', got %q", lang)
	}
}

func TestDetectFromCookie_Empty(t *testing.T) {
	svc := NewService("de", []string{"de", "en"})
	lang, ok := svc.DetectFromCookie("")
	if ok {
		t.Error("expected ok=false for empty cookie value")
	}
	if lang != "de" {
		t.Errorf("expected default 'de', got %q", lang)
	}
}

func TestIsSupported(t *testing.T) {
	svc := NewService("de", []string{"de", "en"})
	tests := []struct {
		lang     Language
		expected bool
	}{
		{"de", true}, {"en", true}, {"fr", false}, {"", false},
	}
	for _, tt := range tests {
		if got := svc.IsSupported(tt.lang); got != tt.expected {
			t.Errorf("IsSupported(%q) = %v, want %v", tt.lang, got, tt.expected)
		}
	}
}

func TestWithLanguage_FromContext(t *testing.T) {
	lc := &LanguageContext{
		Current:    "de",
		Available:  []Language{"de", "en"},
		Alternates: map[Language]string{"de": "start", "en": "home"},
	}
	ctx := WithLanguage(context.Background(), lc)
	result := FromContext(ctx)
	if result == nil {
		t.Fatal("expected non-nil language context")
	}
	if result.Current != "de" {
		t.Errorf("expected current 'de', got %q", result.Current)
	}
	if result.Alternates["en"] != "home" {
		t.Errorf("expected alternate 'home' for 'en', got %q", result.Alternates["en"])
	}
}

func TestFromContext_Missing(t *testing.T) {
	if result := FromContext(context.Background()); result != nil {
		t.Error("expected nil when no language context set")
	}
}

func TestLanguage_String(t *testing.T) {
	if l := Language("de"); l.String() != "de" {
		t.Errorf("expected 'de', got %q", l.String())
	}
}

func TestLanguage_IsValid(t *testing.T) {
	l := Language("en")
	if !l.IsValid([]string{"de", "en"}) {
		t.Error("expected 'en' to be valid")
	}
	if l.IsValid([]string{"de", "fr"}) {
		t.Error("expected 'en' to be invalid")
	}
}
