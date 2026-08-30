package services

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func makeImage(t *testing.T, w, h int, format string) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 120, 255})
		}
	}
	var buf bytes.Buffer
	var err error
	if format == "png" {
		err = png.Encode(&buf, img)
	} else {
		err = jpeg.Encode(&buf, img, nil)
	}
	if err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	return buf.Bytes()
}

func TestDownscaleShrinksOversizedImage(t *testing.T) {
	// A typical phone photo: far wider than any screen shows it at.
	original := makeImage(t, 4000, 3000, "jpeg")

	out, mime, ext, ok := downscaleImage(original, "image/jpeg")
	if !ok {
		t.Fatal("expected an oversized image to be downscaled")
	}
	if mime != "image/jpeg" || ext != ".jpg" {
		t.Errorf("got %s/%s, want image/jpeg/.jpg", mime, ext)
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if cfg.Width != maxStoredDimension {
		t.Errorf("width = %d, want %d", cfg.Width, maxStoredDimension)
	}
	// Aspect ratio must survive: 4000x3000 is 4:3, so 1600 wide is 1200 tall.
	if cfg.Height != 1200 {
		t.Errorf("height = %d, want 1200 (aspect ratio not preserved)", cfg.Height)
	}
	if len(out) >= len(original) {
		t.Errorf("downscaled to %d bytes, no smaller than the original %d", len(out), len(original))
	}
}

func TestDownscaleKeepsPNGAsPNG(t *testing.T) {
	// Logos arrive as PNG and often rely on transparency; re-encoding one to
	// JPEG would both flatten that and soften its edges.
	out, mime, ext, ok := downscaleImage(makeImage(t, 2400, 2400, "png"), "image/png")
	if !ok {
		t.Fatal("expected an oversized PNG to be downscaled")
	}
	if mime != "image/png" || ext != ".png" {
		t.Errorf("got %s/%s, want image/png/.png", mime, ext)
	}
	if _, err := png.Decode(bytes.NewReader(out)); err != nil {
		t.Errorf("result is not valid PNG: %v", err)
	}
}

func TestDownscaleLeavesReasonableImagesAlone(t *testing.T) {
	// Already within the cap — re-encoding would only lose quality.
	if _, _, _, ok := downscaleImage(makeImage(t, 800, 600, "jpeg"), "image/jpeg"); ok {
		t.Error("an image under the cap should be stored untouched")
	}
	// Exactly at the cap counts as within it.
	if _, _, _, ok := downscaleImage(makeImage(t, maxStoredDimension, 900, "jpeg"), "image/jpeg"); ok {
		t.Error("an image exactly at the cap should be stored untouched")
	}
}

func TestDownscaleSkipsNonReencodableTypes(t *testing.T) {
	// A PDF menu, and formats with no stdlib encoder or that would lose an
	// animation, must pass through byte-for-byte.
	for _, mime := range []string{"application/pdf", "image/webp", "image/gif"} {
		if _, _, _, ok := downscaleImage([]byte("not really an image"), mime); ok {
			t.Errorf("%s should never be re-encoded", mime)
		}
	}
}

func TestDownscaleSurvivesGarbage(t *testing.T) {
	// A corrupt or truncated upload must not panic; it just isn't scaled.
	if _, _, _, ok := downscaleImage([]byte{0xff, 0xd8, 0x00, 0x01}, "image/jpeg"); ok {
		t.Error("undecodable bytes should not report a successful downscale")
	}
}
