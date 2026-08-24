package services

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
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

type QRCustomization struct {
	Content         string
	ForegroundColor string // hex, e.g. "#000000"
	BackgroundColor string
	ModuleStyle     string // square | dots | rounded
	EyeStyle        string // square | circular | rounded
	ErrorCorrection QRErrorCorrection
	HasLogo         bool
	SizePx          int // output image size, defaults to 512
}

type QRValidation struct {
	Warnings         []string          `json:"warnings"`
	EffectiveECLevel QRErrorCorrection `json:"effective_error_correction"`
	ContrastRatio    float64           `json:"contrast_ratio"`
}

const quietZoneModules = 4 // fixed, non-configurable, per spec minimum

// Validate checks a requested customization for scannability risk and
// returns the error-correction level that will actually be used (bumped up
// automatically when a logo is present) plus any warnings to surface to the user.
func Validate(c QRCustomization) QRValidation {
	warnings := []string{}

	effectiveEC := c.ErrorCorrection
	if effectiveEC == "" {
		effectiveEC = ECMedium
	}

	if c.HasLogo && (effectiveEC == ECLow || effectiveEC == ECMedium) {
		effectiveEC = ECQuartile
		warnings = append(warnings, "Se elevó el nivel de corrección de errores a Q para compensar el logo central.")
	}

	fg := parseHexColor(c.ForegroundColor)
	bg := parseHexColor(c.BackgroundColor)
	contrast := contrastRatio(fg, bg)

	if contrast < 4.5 {
		warnings = append(warnings, "El contraste entre el color de fondo y el de los módulos es bajo; el QR podría no ser legible por algunos lectores.")
	}

	if c.HasLogo && effectiveEC != ECHigh && (c.ModuleStyle == "dots" || c.EyeStyle == "circular") {
		warnings = append(warnings, "Combinar logo con módulos tipo 'dots' reduce aún más el margen de corrección de errores; considera usar módulos cuadrados o subir a nivel H.")
	}

	return QRValidation{Warnings: warnings, EffectiveECLevel: effectiveEC, ContrastRatio: contrast}
}

// RenderPNG draws the QR code as a raster PNG, honoring module/eye style,
// colors and a fixed quiet zone. Returns the finished bytes.
func RenderPNG(c QRCustomization) ([]byte, error) {
	validation := Validate(c)
	c.ErrorCorrection = validation.EffectiveECLevel

	bitmap, err := generateBitmap(c)
	if err != nil {
		return nil, err
	}

	size := c.SizePx
	if size <= 0 {
		size = 512
	}

	n := len(bitmap)
	totalModules := n + quietZoneModules*2
	moduleSize := float64(size) / float64(totalModules)

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	bg := parseHexColor(c.BackgroundColor)
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)

	fg := parseHexColor(c.ForegroundColor)
	isEyeModule := eyeModuleMask(n)

	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if !bitmap[y][x] {
				continue
			}
			px := (float64(x+quietZoneModules) * moduleSize)
			py := (float64(y+quietZoneModules) * moduleSize)

			style := c.ModuleStyle
			if isEyeModule[y][x] {
				style = eyeRenderStyle(c.EyeStyle)
			}
			drawModule(img, px, py, moduleSize, fg, style)
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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
	isEyeModule := eyeModuleMask(n)

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="crispEdges">`, totalModules, totalModules)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="%s"/>`, totalModules, totalModules, c.BackgroundColor)

	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if !bitmap[y][x] {
				continue
			}
			px := x + quietZoneModules
			py := y + quietZoneModules

			style := c.ModuleStyle
			if isEyeModule[y][x] {
				style = eyeRenderStyle(c.EyeStyle)
			}

			switch style {
			case "dots":
				fmt.Fprintf(&b, `<circle cx="%.2f" cy="%.2f" r="0.45" fill="%s"/>`, float64(px)+0.5, float64(py)+0.5, c.ForegroundColor)
			case "rounded", "circular":
				fmt.Fprintf(&b, `<rect x="%d" y="%d" width="1" height="1" rx="0.3" fill="%s"/>`, px, py, c.ForegroundColor)
			default:
				fmt.Fprintf(&b, `<rect x="%d" y="%d" width="1" height="1" fill="%s"/>`, px, py, c.ForegroundColor)
			}
		}
	}

	b.WriteString(`</svg>`)
	return b.String(), nil
}

func generateBitmap(c QRCustomization) ([][]bool, error) {
	level, ok := ecLevels[c.ErrorCorrection]
	if !ok {
		level = qrcode.Medium
	}

	qr, err := qrcode.New(c.Content, level)
	if err != nil {
		return nil, fmt.Errorf("generate qr: %w", err)
	}
	return qr.Bitmap(), nil
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

func eyeRenderStyle(eyeStyle string) string {
	switch eyeStyle {
	case "circular":
		return "dots"
	case "rounded":
		return "rounded"
	default:
		return "square"
	}
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
