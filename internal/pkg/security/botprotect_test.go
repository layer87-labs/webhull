package security_test

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/layer87-labs/webhull/internal/pkg/security"
)

func TestTextShielder_GeneratePNG(t *testing.T) {
	ts := security.NewTextShielder()

	tests := []struct {
		name string
		text string
	}{
		{"single line", "kontakt@layer87.de"},
		{"multi line", "Layer87\nMusterstrasse 1\n12345 Musterstadt"},
		{"with empty lines", "Line 1\n\nLine 3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := ts.GeneratePNG(&buf, tt.text)
			if err != nil {
				t.Fatalf("GeneratePNG() error = %v", err)
			}

			if buf.Len() == 0 {
				t.Fatal("GeneratePNG() produced empty output")
			}

			// Verify it's a valid PNG.
			img, err := png.Decode(&buf)
			if err != nil {
				t.Fatalf("output is not valid PNG: %v", err)
			}

			bounds := img.Bounds()
			if bounds.Dx() == 0 || bounds.Dy() == 0 {
				t.Fatal("PNG has zero dimensions")
			}
		})
	}
}

func TestTextShielder_Deterministic(t *testing.T) {
	ts := security.NewTextShielder()
	text := "kontakt@layer87.de"

	var buf1, buf2 bytes.Buffer
	if err := ts.GeneratePNG(&buf1, text); err != nil {
		t.Fatal(err)
	}
	if err := ts.GeneratePNG(&buf2, text); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		t.Error("same input should produce identical output (deterministic RNG)")
	}
}

func TestHash(t *testing.T) {
	h1 := security.Hash("kontakt@layer87.de")
	h2 := security.Hash("kontakt@layer87.de")
	h3 := security.Hash("other@example.com")

	if h1 != h2 {
		t.Error("same input should produce same hash")
	}
	if h1 == h3 {
		t.Error("different input should produce different hash")
	}
	if len(h1) != 8 {
		t.Errorf("hash length = %d, want 8", len(h1))
	}
}
