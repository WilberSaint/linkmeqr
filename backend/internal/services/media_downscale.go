package services

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"

	xdraw "golang.org/x/image/draw"
)

// maxStoredDimension caps the longest side of a stored image.
//
// Uploads come straight off a phone camera — routinely 3000-4000px and
// several megabytes — while the page never shows an image wider than a
// full-bleed cover on a large screen. Every visitor was downloading all of
// those pixels over mobile data to look at a fraction of them.
//
// 1600px keeps a cover crisp on a high-density display with room to spare,
// and is where the file size stops being the thing a QR scan waits for.
const maxStoredDimension = 1600

// jpegQuality is the usual "indistinguishable at a glance" point; well below
// the size cliff that starts at 90+.
const jpegQuality = 82

// downscaleImage shrinks an oversized image, returning the re-encoded bytes
// and the mime type/extension they should be stored under.
//
// It returns ok=false whenever the input should be stored exactly as it came
// in — anything already small enough, and anything it can't safely re-encode
// (PDFs, WebP, and animated GIFs, which would lose their animation if
// flattened through a still-image encoder).
func downscaleImage(buf []byte, mimeType string) (out []byte, outMime string, outExt string, ok bool) {
	// Only the formats the standard library can both decode and re-encode
	// losslessly enough to be worth it. WebP decodes but has no stdlib
	// encoder; GIF risks flattening an animation.
	if mimeType != "image/jpeg" && mimeType != "image/png" {
		return nil, "", "", false
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(buf))
	if err != nil {
		return nil, "", "", false
	}
	if cfg.Width <= maxStoredDimension && cfg.Height <= maxStoredDimension {
		return nil, "", "", false
	}

	src, _, err := image.Decode(bytes.NewReader(buf))
	if err != nil {
		return nil, "", "", false
	}

	// Scale the longest side down to the cap, preserving aspect ratio.
	b := src.Bounds()
	scale := float64(maxStoredDimension) / float64(max(b.Dx(), b.Dy()))
	dstW := int(float64(b.Dx()) * scale)
	dstH := int(float64(b.Dy()) * scale)
	if dstW < 1 || dstH < 1 {
		return nil, "", "", false
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	// CatmullRom is the sharpest of the stdlib-adjacent kernels; this runs
	// once per upload, so quality matters more than speed here.
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)

	var encoded bytes.Buffer
	if mimeType == "image/png" {
		// PNG stays PNG: it's what logos are uploaded as, and re-encoding a
		// logo to JPEG would introduce artifacts around hard edges and drop
		// the transparency a logo usually depends on.
		if err := png.Encode(&encoded, dst); err != nil {
			return nil, "", "", false
		}
		return encoded.Bytes(), "image/png", ".png", true
	}

	if err := jpeg.Encode(&encoded, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, "", "", false
	}
	return encoded.Bytes(), "image/jpeg", ".jpg", true
}
