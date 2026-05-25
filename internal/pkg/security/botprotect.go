package security

import (
	"crypto/md5"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math/rand"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// TextShielder generates anti-scraping PNG images from text.
// It renders text with subtle anti-OCR features (noise, color variation,
// positional jitter) so bots cannot easily extract email addresses or
// phone numbers while humans can still read them.
type TextShielder struct{}

// NewTextShielder creates a new TextShielder.
func NewTextShielder() *TextShielder {
	return &TextShielder{}
}

// GeneratePNG renders the given text lines into a PNG image and writes it to w.
// The image includes anti-OCR features (subtle noise, color variations, jitter).
func (ts *TextShielder) GeneratePNG(w io.Writer, text string) error {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	lineHeight := 22
	padding := 20
	width := 380
	height := len(lines)*lineHeight + padding*2

	img := ts.createImage(width, height, lines, lineHeight, padding)
	return png.Encode(w, img)
}

// Hash returns a stable short hash for a given text (for caching/filenames).
func Hash(text string) string {
	h := md5.Sum([]byte(text))
	return fmt.Sprintf("%x", h[:4])
}

// createImage builds the full image with noise, text and anti-OCR elements.
func (ts *TextShielder) createImage(width, height int, lines []string, lineHeight, padding int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Slightly off-white background (harder for OCR than pure white).
	bgColor := color.RGBA{252, 252, 252, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	addNoise(img)

	face := basicfont.Face7x13
	textColor := color.RGBA{45, 45, 45, 255} // Not pure black — harder for OCR.

	// Deterministic but varied RNG seeded from content length.
	rng := rand.New(rand.NewSource(int64(len(lines))))

	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		y := padding + i*lineHeight + 13
		xOffset := rng.Intn(3) - 1
		yOffset := rng.Intn(2)

		drawLine(img, padding+xOffset, y+yOffset, strings.TrimSpace(line), face, textColor, rng)
	}

	addDecoyElements(img, rng)
	return img
}

// addNoise adds subtle pixel-level noise patterns.
func addNoise(img *image.RGBA) {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			switch {
			case (x+y)%13 == 0:
				img.Set(x, y, color.RGBA{250, 250, 250, 255})
			case (x*2+y)%17 == 0:
				img.Set(x, y, color.RGBA{254, 254, 254, 255})
			case (x+y*3)%23 == 0:
				img.Set(x, y, color.RGBA{248, 248, 248, 255})
			}
		}
	}
}

// drawLine renders a single line of text with per-character color and position variations.
func drawLine(img draw.Image, x, y int, text string, face font.Face, baseColor color.Color, rng *rand.Rand) {
	currentX := x

	for i, char := range text {
		if char == ' ' {
			currentX += 8
			continue
		}

		// Subtle per-character color variation.
		variation := rng.Intn(10) - 5
		r, g, b, a := baseColor.RGBA()
		charColor := color.RGBA{
			clamp8(int(r>>8) + variation),
			clamp8(int(g>>8) + variation),
			clamp8(int(b>>8) + variation),
			uint8(a >> 8),
		}

		// Tiny vertical jitter on every 3rd character.
		yVar := 0
		if i%3 == 0 {
			yVar = rng.Intn(2) - 1
		}

		d := &font.Drawer{
			Dst:  img,
			Src:  &image.Uniform{charColor},
			Face: face,
			Dot: fixed.Point26_6{
				X: fixed.Int26_6(currentX * 64),
				Y: fixed.Int26_6((y + yVar) * 64),
			},
		}
		d.DrawString(string(char))

		currentX += charWidth(char)
	}
}

// addDecoyElements places nearly-invisible dots and scan lines to confuse OCR.
func addDecoyElements(img *image.RGBA, rng *rand.Rand) {
	bounds := img.Bounds()

	// Faint decoy dots.
	for i := 0; i < 15; i++ {
		x := rng.Intn(bounds.Max.X)
		y := rng.Intn(bounds.Max.Y)
		img.Set(x, y, color.RGBA{249, 249, 249, 255})

		if rng.Intn(3) == 0 && x+1 < bounds.Max.X && y+1 < bounds.Max.Y {
			img.Set(x+1, y, color.RGBA{251, 251, 251, 255})
			img.Set(x, y+1, color.RGBA{253, 253, 253, 255})
		}
	}

	// Faint horizontal scan lines (simulates bad scanner).
	for i := 0; i < 3; i++ {
		y := rng.Intn(bounds.Max.Y-40) + 20
		for x := bounds.Min.X; x < bounds.Max.X; x += 2 {
			if rng.Intn(8) == 0 {
				img.Set(x, y, color.RGBA{251, 251, 251, 255})
			}
		}
	}
}

// clamp8 clamps an int to valid uint8 range.
func clamp8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// charWidth returns the approximate rendering width for a character.
func charWidth(char rune) int {
	switch char {
	case 'i', 'l', '1', '.', ',', ':', ';', '|':
		return 4
	case 'f', 'j', 'r', 't', 'I':
		return 5
	case 'c', 's', 'v', 'x', 'z':
		return 6
	case 'a', 'b', 'd', 'e', 'g', 'h', 'k', 'n', 'o', 'p', 'q', 'u', 'y':
		return 7
	case 'm', 'w', 'M', 'W':
		return 10
	case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'J', 'K', 'L', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'X', 'Y', 'Z':
		return 8
	default:
		return 7
	}
}
