package services

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	ximgdraw "golang.org/x/image/draw"
)

// halftoneGrid is the resolution (cells per side) the logo is sampled down
// to before being redrawn as dots. Small enough that each cell reads as a
// single dot at typical QR print sizes, big enough that simple logos (a
// wordmark initial, an icon) stay recognizable.
const halftoneGrid = 16

// monochromeGrid is finer than halftoneGrid: a solid silhouette (see
// drawMonochromeLogo) has no dot-size shading to suggest detail with, so it
// needs more cells to still read as the logo rather than a blob.
const monochromeGrid = 30

// inkAmount turns one sampled RGBA pixel into a 0..1 "how much ink here"
// value: darker and more opaque means more ink, fully transparent means
// none. c's channels are alpha-premultiplied (the standard image.RGBA
// convention), so they're unpremultiplied first — otherwise a transparent
// pixel and an opaque black one could look identical (both premultiply to
// near-zero channel values).
func inkAmount(c color.RGBA) float64 {
	if c.A == 0 {
		return 0
	}
	a := float64(c.A) / 255
	r := float64(c.R) / float64(c.A)
	g := float64(c.G) / float64(c.A)
	b := float64(c.B) / float64(c.A)
	luminance := 0.2126*r + 0.7152*g + 0.0722*b
	return a * (1 - luminance)
}

// colorSampleWeight favors opaque, saturated pixels over near-white/gray
// background noise when picking a representative "brand color" for
// dominantLogoColor — unlike inkAmount (which favors darkness, for
// silhouette masking), a light but vivid color like bright yellow should
// still count here.
func colorSampleWeight(c color.RGBA) float64 {
	if c.A == 0 {
		return 0
	}
	a := float64(c.A) / 255
	r := float64(c.R) / float64(c.A)
	g := float64(c.G) / float64(c.A)
	b := float64(c.B) / float64(c.A)
	maxc := math.Max(r, math.Max(g, b))
	minc := math.Min(r, math.Min(g, b))
	sat := 0.0
	if maxc > 0 {
		sat = (maxc - minc) / maxc
	}
	return a * (0.3 + 0.7*sat)
}

// dominantLogoColor samples logo for a representative brand color: the
// alpha+saturation-weighted average of its pixels (see colorSampleWeight).
// ok is false when there's nothing meaningful to sample — a fully
// transparent logo, or one that's entirely gray/white/black.
func dominantLogoColor(logo image.Image) (c color.RGBA, ok bool) {
	const sampleSize = 24
	grid := sampleGrid(logo, sampleSize)
	var rSum, gSum, bSum, wSum float64
	for y := 0; y < sampleSize; y++ {
		for x := 0; x < sampleSize; x++ {
			px := grid.RGBAAt(x, y)
			w := colorSampleWeight(px)
			if w <= 0 {
				continue
			}
			rSum += float64(px.R) / float64(px.A) * 255 * w
			gSum += float64(px.G) / float64(px.A) * 255 * w
			bSum += float64(px.B) / float64(px.A) * 255 * w
			wSum += w
		}
	}
	if wSum < 0.5 {
		return color.RGBA{}, false
	}
	return color.RGBA{
		R: uint8(rSum / wSum),
		G: uint8(gSum / wSum),
		B: uint8(bSum / wSum),
		A: 255,
	}, true
}

// haltoneCellRadius converts ink amount into a dot radius: area (not
// radius) should scale linearly with ink for the halftone to read as
// proportional shading, hence the sqrt — the classic AM-halftone relation.
func halftoneCellRadius(ink, cellSize float64) float64 {
	if ink <= 0.04 {
		return 0
	}
	return cellSize / 2 * math.Sqrt(ink) * 0.92
}

// sampleGrid downscales logo to size x size so each cell's pixel represents
// that region's average color — the same downsampling PNG and SVG
// rendering both sample from, so the two outputs always agree.
func sampleGrid(logo image.Image, size int) *image.RGBA {
	grid := image.NewRGBA(image.Rect(0, 0, size, size))
	ximgdraw.BiLinear.Scale(grid, grid.Bounds(), logo, logo.Bounds(), ximgdraw.Over, nil)
	return grid
}

// drawHalftoneLogo renders logo inside rect as a grid of variable-radius
// dots sized by each cell's ink amount — the classic print halftone
// technique — so an uploaded photo/logo reads as part of the QR's own dot
// texture instead of clashing with it as a smooth image.
func drawHalftoneLogo(img *image.RGBA, rect image.Rectangle, logo image.Image, fg color.RGBA) {
	grid := sampleGrid(logo, halftoneGrid)
	cell := float64(rect.Dx()) / float64(halftoneGrid)

	for row := 0; row < halftoneGrid; row++ {
		for col := 0; col < halftoneGrid; col++ {
			r := halftoneCellRadius(inkAmount(grid.RGBAAt(col, row)), cell)
			if r <= 0 {
				continue
			}
			cx := float64(rect.Min.X) + (float64(col)+0.5)*cell
			cy := float64(rect.Min.Y) + (float64(row)+0.5)*cell
			drawFilledCircle(img, cx, cy, r, fg)
		}
	}
}

// drawMonochromeLogo renders logo inside rect as a solid single-color
// silhouette — every cell whose ink amount clears the threshold is filled
// completely with fg, no shading or dots. This is the recommended way to
// carry a logo: recoloring it into the QR's own ink color (instead of
// pasting it in full color) is what makes it read as part of the code
// rather than a sticker placed on top of it.
func drawMonochromeLogo(img *image.RGBA, rect image.Rectangle, logo image.Image, fg color.RGBA) {
	grid := sampleGrid(logo, monochromeGrid)
	cell := float64(rect.Dx()) / float64(monochromeGrid)

	for row := 0; row < monochromeGrid; row++ {
		for col := 0; col < monochromeGrid; col++ {
			if inkAmount(grid.RGBAAt(col, row)) < 0.45 {
				continue
			}
			x0 := rect.Min.X + int(float64(col)*cell)
			y0 := rect.Min.Y + int(float64(row)*cell)
			x1 := rect.Min.X + int(float64(col+1)*cell) + 1
			y1 := rect.Min.Y + int(float64(row+1)*cell) + 1
			for py := y0; py < y1; py++ {
				for px := x0; px < x1; px++ {
					img.Set(px, py, fg)
				}
			}
		}
	}
}

func drawFilledCircle(img *image.RGBA, cx, cy, r float64, c color.Color) {
	x0, x1 := int(math.Floor(cx-r)), int(math.Ceil(cx+r))
	y0, y1 := int(math.Floor(cy-r)), int(math.Ceil(cy+r))
	for py := y0; py <= y1; py++ {
		for px := x0; px <= x1; px++ {
			dx, dy := float64(px)-cx, float64(py)-cy
			if dx*dx+dy*dy <= r*r {
				img.Set(px, py, c)
			}
		}
	}
}

// writeSVGHalftoneLogo is drawHalftoneLogo's SVG counterpart: logoR is the
// half-width of the (square) logo area, centered on cx,cy in module units.
func writeSVGHalftoneLogo(b *strings.Builder, logo image.Image, cx, cy, logoR float64, fg string) {
	grid := sampleGrid(logo, halftoneGrid)
	cell := logoR * 2 / float64(halftoneGrid)
	originX, originY := cx-logoR, cy-logoR

	for row := 0; row < halftoneGrid; row++ {
		for col := 0; col < halftoneGrid; col++ {
			r := halftoneCellRadius(inkAmount(grid.RGBAAt(col, row)), cell)
			if r <= 0 {
				continue
			}
			px := originX + (float64(col)+0.5)*cell
			py := originY + (float64(row)+0.5)*cell
			fmt.Fprintf(b, `<circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s"/>`, px, py, r, fg)
		}
	}
}

// writeSVGMonochromeLogo is drawMonochromeLogo's SVG counterpart. Adjacent
// cells are drawn slightly oversized (+0.5 unit) so sub-pixel rounding in
// the renderer can't leave visible hairline gaps between them.
func writeSVGMonochromeLogo(b *strings.Builder, logo image.Image, cx, cy, logoR float64, fg string) {
	grid := sampleGrid(logo, monochromeGrid)
	cell := logoR * 2 / float64(monochromeGrid)
	originX, originY := cx-logoR, cy-logoR

	for row := 0; row < monochromeGrid; row++ {
		for col := 0; col < monochromeGrid; col++ {
			if inkAmount(grid.RGBAAt(col, row)) < 0.45 {
				continue
			}
			px := originX + float64(col)*cell
			py := originY + float64(row)*cell
			fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s"/>`, px, py, cell+0.5, cell+0.5, fg)
		}
	}
}
