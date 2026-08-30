package services

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
	ximgdraw "golang.org/x/image/draw"
)

type QRErrorCorrection string

const (
	ECLow      QRErrorCorrection = "L"
	ECMedium   QRErrorCorrection = "M"
	ECQuartile QRErrorCorrection = "Q"
	ECHigh     QRErrorCorrection = "H"
)

var ecLevels = map[QRErrorCorrection]qrcode.RecoveryLevel{
	ECLow:      qrcode.Low,
	ECMedium:   qrcode.Medium,
	ECQuartile: qrcode.High,    // library naming: High == ~25% ("Q" in the standard)
	ECHigh:     qrcode.Highest, // ~30% ("H" in the standard)
}

// PresetIcons lists the built-in icon keys available for the QR center badge
// when the user doesn't want to upload their own logo. Drawn procedurally
// (see drawPresetIcon/svgPresetIcon) — no external assets involved.
var PresetIcons = []string{"coffee", "heart", "matcha", "star", "gift"}

type QRCustomization struct {
	Content         string
	ForegroundColor string // hex, e.g. "#000000"
	BackgroundColor string
	ModuleStyle     string // square | dots | rounded
	EyeStyle        string // square | circular | rounded
	ErrorCorrection QRErrorCorrection
	HasLogo         bool
	SizePx          int // size of the scannable QR window itself, defaults to 512

	// LogoImage is the decoded uploaded logo, used for PNG compositing.
	// LogoBytes/LogoMimeType carry the original file for SVG data-URI embedding.
	// Mutually exclusive with PresetIcon.
	LogoImage    image.Image
	LogoBytes    []byte
	LogoMimeType string

	// LogoStyle controls how LogoImage is rendered: "color" (the original
	// image, unmodified), "monochrome" (recolored to the QR's own ink color
	// as a solid silhouette — the recommended choice, since it reads as
	// part of the code rather than a sticker placed on top of it), or
	// "dots" (a halftone dot pattern sampled from its luminance). Ignored
	// when LogoImage is nil (has no effect on PresetIcon).
	LogoStyle string

	// EyeColorFromLogo tints the three finder-pattern eyes with a dominant
	// color sampled from LogoImage instead of ForegroundColor — e.g. a
	// green logo gets green eyes. Only meaningful with LogoStyle "color"
	// (a monochrome/dotted logo is already drawn in the ink color, so
	// there'd be nothing distinct to sample). Falls back to
	// ForegroundColor if the sampled color doesn't have enough contrast
	// against BackgroundColor to stay reliably scannable.
	EyeColorFromLogo bool

	// PresetIcon is one of PresetIcons, drawn procedurally instead of a logo.
	PresetIcon string

	// FrameShape is either empty (plain QR) or "custom_logo" — the only
	// remaining option after a set of decorative preset outer-frame
	// silhouettes (heart, coffee, star, ...) was retired following repeated
	// feedback that they never looked right. "custom_logo" makes every dark
	// module (and finder eye) sample its color from LogoImage instead of
	// flat ForegroundColor — see sampleLogoModuleColor.
	FrameShape string

	// ShapeFill, when true alongside FrameShape "custom_logo", bumps the
	// effective error-correction level to H (see Validate) — the per-module
	// color sampling reduces contrast versus a flat-ink QR, so the extra EC
	// budget compensates for that reduced scan margin.
	ShapeFill bool
}

func (c QRCustomization) hasCenterBadge() bool {
	return c.LogoImage != nil || c.PresetIcon != ""
}

type QRValidation struct {
	Warnings         []string          `json:"warnings"`
	EffectiveECLevel QRErrorCorrection `json:"effective_error_correction"`
	ContrastRatio    float64           `json:"contrast_ratio"`
}

const quietZoneModules = 4 // fixed, non-configurable, per spec minimum

// Validate checks a requested customization for scannability risk and
// returns the error-correction level that will actually be used (bumped up
// automatically when a logo is present) plus any warnings to surface to the
// user. Always starts from the fixed ECMedium baseline rather than
// c.ErrorCorrection — that field is normally whatever got stored from the
// PREVIOUS render (see ToCustomization), and reading it back in here would
// make a bump one-directional: once a logo or shape-fill pushed a QR to EC
// Q/H, it would stay there forever even after the user removed whatever
// caused the bump, since every later call would see its own past bump as
// the new floor.
func Validate(c QRCustomization) QRValidation {
	warnings := []string{}

	effectiveEC := ECMedium

	// "custom_logo" never puts the logo in the center (drawCenterBadge is
	// skipped for it — see RenderPNG/RenderSVG) — it's either a full-color
	// decorative frame (plain mode, touches zero real modules) or a
	// shape-fill silhouette (its own EC-H bump below already covers it) — so
	// c.HasLogo alone shouldn't trigger the "compensating a center badge"
	// bump/warning here the way it does for an actual center logo.
	hasCenterLogo := c.HasLogo && c.FrameShape != "custom_logo"

	if hasCenterLogo {
		effectiveEC = ECQuartile
		warnings = append(warnings, "Se elevó el nivel de corrección de errores a Q para compensar el logo central.")
	}

	if c.ShapeFill && c.FrameShape != "" {
		effectiveEC = ECHigh
	}

	fg := parseHexColor(c.ForegroundColor)
	bg := parseHexColor(c.BackgroundColor)
	contrast := contrastRatio(fg, bg)

	if contrast < 4.5 {
		warnings = append(warnings, "El contraste entre el color de fondo y el de los módulos es bajo; el QR podría no ser legible por algunos lectores.")
	} else if relativeLuminance(fg) > relativeLuminance(bg) {
		// High contrast alone doesn't save this: most QR readers (including
		// the library powering a large share of real scanner apps) only
		// ever look for dark modules on a light background — they don't
		// attempt the inverted hypothesis even when asked to try harder, so
		// an inverted code can go completely undetected despite reading
		// fine to the human eye. Confirmed empirically against a real
		// decoder before writing this warning, not a guess.
		warnings = append(warnings, "El color del código es más claro que el fondo (colores invertidos). Muchos lectores de QR no reconocen este patrón aunque el contraste sea alto — usa un color de código más oscuro que el de fondo para máxima compatibilidad.")
	}

	if hasCenterLogo && effectiveEC != ECHigh && (c.ModuleStyle == "dots" || c.EyeStyle == "circular") {
		warnings = append(warnings, "Combinar logo con módulos tipo 'dots' reduce aún más el margen de corrección de errores; considera usar módulos cuadrados o subir a nivel H.")
	}

	return QRValidation{Warnings: warnings, EffectiveECLevel: effectiveEC, ContrastRatio: contrast}
}

// RenderPNG draws the QR code as a raster PNG, honoring module/eye style,
// colors and a fixed quiet zone. Returns the finished bytes. When
// FrameShape is "custom_logo", every dark module (and each of the 3 finder
// eyes) is colorized by sampling the business's own logo at that module's
// position instead of painting flat ForegroundColor — see
// sampleLogoModuleColor's doc comment for why this went through several
// design iterations before landing here (a set of decorative geometric
// "frame" shapes — heart, coffee cup, star, etc. — used to live alongside
// this, eroding real QR modules to trace a silhouette; retired after
// repeated rounds of feedback that the effect never looked right, and
// mostly duplicated what "custom_logo" already does better and more
// safely).
func RenderPNG(c QRCustomization) ([]byte, error) {
	validation := Validate(c)
	c.ErrorCorrection = validation.EffectiveECLevel

	bitmap, err := generateBitmap(c)
	if err != nil {
		return nil, err
	}

	canvas := c.SizePx
	if canvas <= 0 {
		canvas = 512
	}
	n := len(bitmap)
	isCustomLogo := c.FrameShape == "custom_logo" && c.LogoImage != nil

	bg := parseHexColor(c.BackgroundColor)
	fg := parseHexColor(c.ForegroundColor)

	img := image.NewRGBA(image.Rect(0, 0, canvas, canvas))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)

	totalModules := n + quietZoneModules*2
	qrRect := img.Bounds()

	moduleSize := float64(qrRect.Dx()) / float64(totalModules)
	originX, originY := float64(qrRect.Min.X), float64(qrRect.Min.Y)
	isEyeModule := eyeModuleMask(n)
	eyeColor := eyeInkColor(c, fg, bg)

	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if !bitmap[y][x] || isEyeModule[y][x] {
				continue // eyes are drawn as one unified shape each below
			}
			px := originX + float64(x+quietZoneModules)*moduleSize
			py := originY + float64(y+quietZoneModules)*moduleSize

			moduleColor := fg
			if isCustomLogo {
				moduleColor = sampleLogoModuleColor(c.LogoImage, float64(x)/float64(n), float64(y)/float64(n), fg)
			}
			drawModule(img, px, py, moduleSize, moduleColor, c.ModuleStyle)
		}
	}

	for _, eye := range [][2]int{{0, 0}, {0, n - 7}, {n - 7, 0}} {
		ex := originX + float64(eye[1]+quietZoneModules)*moduleSize
		ey := originY + float64(eye[0]+quietZoneModules)*moduleSize
		ec := eyeColor
		if isCustomLogo {
			// Sample from the eye's own 7x7-module center, not just its
			// top-left corner, so the ring's color reflects what's actually
			// under it rather than the pixel just outside its edge.
			nx := (float64(eye[1]) + 3.5) / float64(n)
			ny := (float64(eye[0]) + 3.5) / float64(n)
			ec = sampleLogoModuleColor(c.LogoImage, nx, ny, eyeColor)
		}
		drawEye(img, ex, ey, moduleSize, c.EyeStyle, ec, bg)
	}

	// A "custom_logo" frame already IS the logo — compositing it again as a
	// small center badge on top would be redundant.
	if c.hasCenterBadge() && c.FrameShape != "custom_logo" {
		drawCenterBadge(img, qrRect, bg, fg, c)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// drawCenterBadge composites the uploaded logo (scaled to fit, or recolored
// per LogoStyle) or a procedurally drawn preset icon into the middle of
// ref (the QR window). Deliberately does NOT clear a backing box first —
// for a logo, that box is what makes it read as a sticker pasted over the
// code instead of part of it; skipping it lets the surrounding modules (or,
// for a transparent-background logo, the page background right through its
// gaps) show all the way up to the logo's own silhouette. Preset icons get
// a small circular bg halo instead (see drawPresetIcon) since they're
// simple single-color glyphs with no silhouette of their own to blend into.
func drawCenterBadge(img *image.RGBA, ref image.Rectangle, bg, fg color.RGBA, c QRCustomization) {
	size := ref.Dx()
	logoSize := int(float64(size) * 0.20)
	cx, cy := ref.Min.X+size/2, ref.Min.Y+size/2

	innerRect := image.Rect(cx-logoSize/2, cy-logoSize/2, cx+logoSize/2, cy+logoSize/2)

	if c.LogoImage != nil {
		switch c.LogoStyle {
		case "monochrome":
			drawMonochromeLogo(img, innerRect, c.LogoImage, fg)
		case "dots":
			drawHalftoneLogo(img, innerRect, c.LogoImage, fg)
		default:
			ximgdraw.CatmullRom.Scale(img, innerRect, c.LogoImage, c.LogoImage.Bounds(), ximgdraw.Over, nil)
		}
		return
	}
	if c.PresetIcon != "" {
		drawPresetIcon(img, c.PresetIcon, innerRect, fg, bg)
	}
}

// sampleLogoModuleColor is the core of "immersive" logo QR art: instead of
// eroding/removing modules to trace a silhouette, EVERY real module stays —
// nothing is dropped, so there's no error-correction gamble at all — but
// each dark module's color is sampled from the logo image at that module's
// own position, so the finished code visibly IS a colorized version of the
// logo (exactly the "Burger King / Starbucks / KFC" QR-art technique: the
// dot pattern doubles as both real QR data and a low-res rendition of the
// artwork). nx,ny are normalized to [0,1] over the module grid.
//
// Two safety rules keep this scan-reliable no matter what the source image
// looks like at a given point:
//   - A pixel too transparent or too close to white to read as "ink" falls
//     back to the plain foreground color instead of vanishing into the
//     background (this is what makes a mostly-white/transparent logo still
//     produce a fully legible code — most of its modules simply end up in
//     plain ink, with color only where the logo actually has some).
//   - Every sampled color is blended toward black by a fixed margin, since
//     a scanner reads modules by LUMINANCE contrast against the (light)
//     background — a bright, saturated source color (e.g. yellow) can still
//     read as "too light" on its own even though it looks plenty dark to
//     the eye.
func sampleLogoModuleColor(logoImg image.Image, nx, ny float64, fg color.RGBA) color.RGBA {
	if logoImg == nil {
		return fg
	}
	b := squareCrop(logoImg.Bounds())
	px := b.Min.X + int(nx*float64(b.Dx()))
	py := b.Min.Y + int(ny*float64(b.Dy()))
	if px < b.Min.X {
		px = b.Min.X
	}
	if px >= b.Max.X {
		px = b.Max.X - 1
	}
	if py < b.Min.Y {
		py = b.Min.Y
	}
	if py >= b.Max.Y {
		py = b.Max.Y - 1
	}

	r, g, bl, a := logoImg.At(px, py).RGBA()
	if a < 0x2000 {
		return fg // essentially transparent — nothing meaningful to sample
	}
	// r/g/b come alpha-premultiplied out of At(); undo that so the hue is
	// the pixel's real color, not darkened toward transparency.
	r8 := uint8(float64(r) / float64(a) * 255)
	g8 := uint8(float64(g) / float64(a) * 255)
	b8 := uint8(float64(bl) / float64(a) * 255)

	if lum := 0.299*float64(r8) + 0.587*float64(g8) + 0.114*float64(b8); lum > 235 {
		return fg // near-white pixel — would disappear against the background
	}

	const blackMix = 0.3
	mix := func(v uint8) uint8 { return uint8(float64(v) * (1 - blackMix)) }
	return color.RGBA{R: mix(r8), G: mix(g8), B: mix(b8), A: 255}
}

// squareCrop returns the largest centered square within b — used anywhere
// an arbitrary (possibly non-square) uploaded image needs to fill a square
// slot without distorting its aspect ratio (a "cover" crop, matching the
// preserveAspectRatio="xMidYMid slice" the SVG side uses for the same job).
func squareCrop(b image.Rectangle) image.Rectangle {
	side := b.Dx()
	if b.Dy() < side {
		side = b.Dy()
	}
	cx, cy := (b.Min.X+b.Max.X)/2, (b.Min.Y+b.Max.Y)/2
	return image.Rect(cx-side/2, cy-side/2, cx-side/2+side, cy-side/2+side)
}

// RenderSVG draws the QR code as a scalable vector image with the same rules as RenderPNG.
func RenderSVG(c QRCustomization) (string, error) {
	validation := Validate(c)
	c.ErrorCorrection = validation.EffectiveECLevel

	bitmap, err := generateBitmap(c)
	if err != nil {
		return "", err
	}

	n := len(bitmap)
	totalModules := n + quietZoneModules*2
	canvasUnits := float64(totalModules)
	isCustomLogo := c.FrameShape == "custom_logo" && c.LogoImage != nil
	isEyeModule := eyeModuleMask(n)

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.2f %.2f" shape-rendering="crispEdges">`, canvasUnits, canvasUnits)
	fmt.Fprintf(&b, `<rect width="%.2f" height="%.2f" fill="%s"/>`, canvasUnits, canvasUnits, c.BackgroundColor)

	fg := parseHexColor(c.ForegroundColor)
	eyeColor := eyeInkColor(c, fg, parseHexColor(c.BackgroundColor))
	eyeColorHex := hexColor(eyeColor)

	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if !bitmap[y][x] || isEyeModule[y][x] {
				continue // eyes are drawn as one unified shape each below
			}
			px := float64(x + quietZoneModules)
			py := float64(y + quietZoneModules)

			moduleColorHex := c.ForegroundColor
			if isCustomLogo {
				moduleColorHex = hexColor(sampleLogoModuleColor(c.LogoImage, float64(x)/float64(n), float64(y)/float64(n), fg))
			}
			writeSVGModule(&b, c.ModuleStyle, px, py, moduleColorHex)
		}
	}

	for _, eye := range [][2]int{{0, 0}, {0, n - 7}, {n - 7, 0}} {
		ex := float64(eye[1] + quietZoneModules)
		ey := float64(eye[0] + quietZoneModules)
		ecHex := eyeColorHex
		if isCustomLogo {
			nx := (float64(eye[1]) + 3.5) / float64(n)
			ny := (float64(eye[0]) + 3.5) / float64(n)
			ecHex = hexColor(sampleLogoModuleColor(c.LogoImage, nx, ny, eyeColor))
		}
		writeSVGEye(&b, ex, ey, c.EyeStyle, ecHex, c.BackgroundColor)
	}

	// See RenderPNG's identical guard: a "custom_logo" frame already IS the
	// logo, so compositing it again as a center badge would be redundant.
	if c.hasCenterBadge() && c.FrameShape != "custom_logo" {
		writeSVGCenterBadge(&b, c, canvasUnits/2, canvasUnits/2, canvasUnits)
	}

	b.WriteString(`</svg>`)
	return b.String(), nil
}

// writeSVGModule writes a single module cell in the requested style, shared
// between the real QR grid and the decorative texture so both read as one
// continuous pattern.
func writeSVGModule(b *strings.Builder, style string, px, py float64, fg string) {
	switch style {
	case "dots":
		// r=0.5 reaches exactly to the cell edge, so two adjacent "on"
		// modules' dots touch instead of leaving a gap between them —
		// matches drawModule's PNG rendering (r := (x1-x0)/2, the full
		// pixel radius) below. A smaller radius here left a real 10% gap
		// antialiasing then softened into a genuine contrast loss: at
		// print-card sizes this made "dots"-styled QRs fail to decode
		// entirely, confirmed with a real decoder, even though the same
		// QR in the default square style scanned fine at the same size.
		fmt.Fprintf(b, `<circle cx="%.2f" cy="%.2f" r="0.5" fill="%s"/>`, px+0.5, py+0.5, fg)
	case "rounded", "circular":
		fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="1" height="1" rx="0.3" fill="%s"/>`, px, py, fg)
	default:
		fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="1" height="1" fill="%s"/>`, px, py, fg)
	}
}

// writeSVGEye is drawEye's SVG counterpart — see its doc comment for why a
// finder pattern is drawn as three concentric solid shapes rather than 49
// individually-styled modules.
func writeSVGEye(b *strings.Builder, x0, y0 float64, style string, fg, bg string) {
	switch style {
	case "circular":
		cx, cy := x0+3.5, y0+3.5
		fmt.Fprintf(b, `<circle cx="%.2f" cy="%.2f" r="3.5" fill="%s"/>`, cx, cy, fg)
		fmt.Fprintf(b, `<circle cx="%.2f" cy="%.2f" r="2.5" fill="%s"/>`, cx, cy, bg)
		fmt.Fprintf(b, `<circle cx="%.2f" cy="%.2f" r="1.5" fill="%s"/>`, cx, cy, fg)
	case "rounded":
		fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="7" height="7" rx="1.4" fill="%s"/>`, x0, y0, fg)
		fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="5" height="5" rx="1" fill="%s"/>`, x0+1, y0+1, bg)
		fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="3" height="3" rx="0.6" fill="%s"/>`, x0+2, y0+2, fg)
	default:
		fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="7" height="7" fill="%s"/>`, x0, y0, fg)
		fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="5" height="5" fill="%s"/>`, x0+1, y0+1, bg)
		fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="3" height="3" fill="%s"/>`, x0+2, y0+2, fg)
	}
}

// writeSVGCenterBadge is drawCenterBadge's SVG counterpart — see its doc
// comment for why a logo gets no backing box (svgPresetIcon draws its own
// small circular halo for preset icons instead).
func writeSVGCenterBadge(b *strings.Builder, c QRCustomization, cx, cy, windowUnits float64) {
	logoR := windowUnits * 0.10

	if c.LogoImage != nil && c.LogoStyle == "monochrome" {
		writeSVGMonochromeLogo(b, c.LogoImage, cx, cy, logoR, c.ForegroundColor)
		return
	}
	if c.LogoImage != nil && c.LogoStyle == "dots" {
		writeSVGHalftoneLogo(b, c.LogoImage, cx, cy, logoR, c.ForegroundColor)
		return
	}
	if len(c.LogoBytes) > 0 && c.LogoMimeType != "" {
		encoded := base64.StdEncoding.EncodeToString(c.LogoBytes)
		fmt.Fprintf(b, `<image x="%.2f" y="%.2f" width="%.2f" height="%.2f" href="data:%s;base64,%s" preserveAspectRatio="xMidYMid slice"/>`,
			cx-logoR, cy-logoR, logoR*2, logoR*2, c.LogoMimeType, encoded)
		return
	}
	if c.PresetIcon != "" {
		svgPresetIcon(b, c.PresetIcon, cx, cy, logoR, c.ForegroundColor, c.BackgroundColor)
	}
}

// libQuietZone is the size of the quiet-zone margin skip2/go-qrcode already
// bakes into the bitmap it returns. Every other position assumption in this
// file (finder patterns at row/col 0, timing pattern at row/col 6, ...)
// requires the raw module matrix with no margin — stripped below.
const libQuietZone = 4

func generateBitmap(c QRCustomization) ([][]bool, error) {
	level, ok := ecLevels[c.ErrorCorrection]
	if !ok {
		level = qrcode.Medium
	}

	qr, err := qrcode.New(c.Content, level)
	if err != nil {
		return nil, fmt.Errorf("generate qr: %w", err)
	}

	raw := qr.Bitmap()
	n := len(raw) - libQuietZone*2
	bitmap := make([][]bool, n)
	for i := range bitmap {
		bitmap[i] = raw[i+libQuietZone][libQuietZone : libQuietZone+n]
	}
	return bitmap, nil
}

// eyeModuleMask marks the three 7x7 finder-pattern ("eye") regions so they
// can be rendered with a distinct style from the data modules.
func eyeModuleMask(n int) [][]bool {
	mask := make([][]bool, n)
	for i := range mask {
		mask[i] = make([]bool, n)
	}

	mark := func(top, left int) {
		for y := top; y < top+7 && y < n; y++ {
			for x := left; x < left+7 && x < n; x++ {
				if y >= 0 && x >= 0 {
					mask[y][x] = true
				}
			}
		}
	}

	mark(0, 0)
	mark(0, n-7)
	mark(n-7, 0)

	return mask
}

// drawEye draws one 7x7-module finder pattern at (x0,y0) (its top-left
// corner) as three concentric solid shapes — outer ring, background gap,
// center block — rather than styling its 49 modules individually the way
// drawModule styles an ordinary data module. That distinction matters for
// scanning, not just looks: a finder pattern's outer ring and center block
// each have to read as ONE continuous run of dark pixels for a decoder's
// row-scan ratio test (the classic 1:1:3:1:1) to find the pattern at all.
// Rendering each of its 49 modules as an independent dot or rounded square
// — which is what happens if they're just run through the ordinary
// per-module style — breaks the ring and the center block into separate
// islands with background gaps between them, and the pattern stops being
// detectable at all rather than merely harder to read.
func drawEye(img *image.RGBA, x0, y0, moduleSize float64, style string, fg, bg color.RGBA) {
	switch style {
	case "circular":
		cx, cy := x0+3.5*moduleSize, y0+3.5*moduleSize
		drawFilledCircle(img, cx, cy, 3.5*moduleSize, fg)
		drawFilledCircle(img, cx, cy, 2.5*moduleSize, bg)
		drawFilledCircle(img, cx, cy, 1.5*moduleSize, fg)
	case "rounded":
		drawRoundedSquare(img, moduleRect(x0, y0, 7, moduleSize), fg, int(moduleSize*1.4))
		drawRoundedSquare(img, moduleRect(x0+moduleSize, y0+moduleSize, 5, moduleSize), bg, int(moduleSize))
		drawRoundedSquare(img, moduleRect(x0+2*moduleSize, y0+2*moduleSize, 3, moduleSize), fg, int(moduleSize*0.6))
	default:
		drawRoundedSquare(img, moduleRect(x0, y0, 7, moduleSize), fg, 0)
		drawRoundedSquare(img, moduleRect(x0+moduleSize, y0+moduleSize, 5, moduleSize), bg, 0)
		drawRoundedSquare(img, moduleRect(x0+2*moduleSize, y0+2*moduleSize, 3, moduleSize), fg, 0)
	}
}

// moduleRect is a size*size-module square with its top-left corner at
// (x0,y0), in pixel space.
func moduleRect(x0, y0 float64, sizeModules int, moduleSize float64) image.Rectangle {
	return image.Rect(int(math.Round(x0)), int(math.Round(y0)), int(math.Round(x0+float64(sizeModules)*moduleSize)), int(math.Round(y0+float64(sizeModules)*moduleSize)))
}

func drawModule(img *image.RGBA, x, y, size float64, c color.Color, style string) {
	x0, y0 := int(math.Round(x)), int(math.Round(y))
	x1, y1 := int(math.Round(x+size)), int(math.Round(y+size))

	switch style {
	case "dots":
		cx, cy := (x0+x1)/2, (y0+y1)/2
		r := (x1 - x0) / 2
		for py := y0; py < y1; py++ {
			for px := x0; px < x1; px++ {
				dx, dy := px-cx, py-cy
				if dx*dx+dy*dy <= r*r {
					img.Set(px, py, c)
				}
			}
		}
	case "rounded", "circular":
		inset := int(float64(x1-x0) * 0.15)
		for py := y0; py < y1; py++ {
			for px := x0; px < x1; px++ {
				if px < x0+inset && py < y0+inset {
					continue
				}
				if px >= x1-inset && py < y0+inset {
					continue
				}
				if px < x0+inset && py >= y1-inset {
					continue
				}
				if px >= x1-inset && py >= y1-inset {
					continue
				}
				img.Set(px, py, c)
			}
		}
	default: // square
		for py := y0; py < y1; py++ {
			for px := x0; px < x1; px++ {
				img.Set(px, py, c)
			}
		}
	}
}

// inflateRect grows rect by factor around its own center (factor > 1 grows,
// < 1 shrinks).
func inflateRect(rect image.Rectangle, factor float64) image.Rectangle {
	cx := (rect.Min.X + rect.Max.X) / 2
	cy := (rect.Min.Y + rect.Max.Y) / 2
	hw := int(float64(rect.Dx()) / 2 * factor)
	hh := int(float64(rect.Dy()) / 2 * factor)
	return image.Rect(cx-hw, cy-hh, cx+hw, cy+hh)
}

// drawRoundedSquare fills rect with c, rounding all four corners to radius.
func drawRoundedSquare(img *image.RGBA, rect image.Rectangle, c color.Color, radius int) {
	x0, y0, x1, y1 := rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y
	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			cornerX, cornerY, inCorner := 0, 0, false
			switch {
			case px < x0+radius && py < y0+radius:
				cornerX, cornerY, inCorner = x0+radius, y0+radius, true
			case px >= x1-radius && py < y0+radius:
				cornerX, cornerY, inCorner = x1-radius, y0+radius, true
			case px < x0+radius && py >= y1-radius:
				cornerX, cornerY, inCorner = x0+radius, y1-radius, true
			case px >= x1-radius && py >= y1-radius:
				cornerX, cornerY, inCorner = x1-radius, y1-radius, true
			}
			if inCorner {
				dx, dy := px-cornerX, py-cornerY
				if dx*dx+dy*dy > radius*radius {
					continue
				}
			}
			img.Set(px, py, c)
		}
	}
}

func parseHexColor(s string) color.RGBA {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return color.RGBA{R: 0, G: 0, B: 0, A: 255}
	}
	var r, g, b uint8
	if _, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b); err != nil {
		return color.RGBA{R: 0, G: 0, B: 0, A: 255}
	}
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

func hexColor(c color.RGBA) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// eyeInkColor picks what color the three finder-pattern eyes draw in:
// ForegroundColor by default, or a color sampled from LogoImage when
// EyeColorFromLogo is set and that color has enough contrast against
// BackgroundColor to stay reliably scannable — finder patterns are how a
// decoder locates and calibrates against the code at all, so unlike a data
// module's color (which is always ForegroundColor, no per-module choice)
// this one silently falls back rather than risk a low-contrast eye.
func eyeInkColor(c QRCustomization, fg, bg color.RGBA) color.RGBA {
	if !c.EyeColorFromLogo || c.LogoImage == nil {
		return fg
	}
	sampled, ok := dominantLogoColor(c.LogoImage)
	if !ok || contrastRatio(sampled, bg) < 3 {
		return fg
	}
	return sampled
}

// contrastRatio implements the WCAG relative-luminance contrast formula,
// used here purely as a scannability heuristic (not for accessibility compliance).
func contrastRatio(a, b color.RGBA) float64 {
	l1 := relativeLuminance(a)
	l2 := relativeLuminance(b)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

func relativeLuminance(c color.RGBA) float64 {
	toLinear := func(v uint8) float64 {
		f := float64(v) / 255
		if f <= 0.03928 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	r, g, b := toLinear(c.R), toLinear(c.G), toLinear(c.B)
	return 0.2126*r + 0.7152*g + 0.0722*b
}
