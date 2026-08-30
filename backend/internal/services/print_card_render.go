package services

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"linkmeqr/backend/internal/models"
)

// SizePresets maps a print card's chosen size to physical dimensions, in
// inches. The card SVG's viewBox uses cardUnitsPerInch units per inch — an
// arbitrary but fixed scale, DPI-independent by construction, same approach
// RenderSVG already uses for the QR itself (module-unit viewBox).
var SizePresets = map[models.PrintCardSizePreset]struct{ WidthIn, HeightIn float64 }{
	models.SizeBusinessCard:  {3.5, 2},
	models.SizeTableTent:     {4, 6},
	models.SizeStickerSquare: {2, 2},
	models.SizeDoorHanger:    {3.5, 8.5},
}

// doorHangerHole gives the die-cut hole's geometry for the door_hanger
// preset — a circle sized and positioned to slip over a standard door
// handle/knob, near the top with enough material above and around it to
// stay structurally sound once cut.
func doorHangerHole(w, h float64) (cx, cy, r float64) {
	return w / 2, h * 0.155, math.Min(w, h) * 0.11
}

const cardUnitsPerInch = 100.0

// cmPerInch converts a custom size's admin-entered centimeters into the
// inches every other measurement in this file (SizePresets, cardUnitsPerInch)
// is already expressed in — the internal unit stays inches; only the UI
// (see the frontend's PRINT_CARD_SIZES labels) shows centimeters.
const cmPerInch = 2.54

// qrHaloScale is how much wider the white "halo" disc behind an icon badge
// (the star, the Google/platform mark, the gift/heart glyph...) sits
// compared to the glyph it holds — the signature shape of the "bloque de
// color" style: everything floats on its own white disc so it reads
// instantly against any brand color, however saturated or dark.
const qrHaloScale = 1.55

// qrSquareScale is the QR's OWN container size relative to the QR itself —
// deliberately a tight rounded SQUARE rather than a circle. A circle sized
// to fully contain a square (qrHaloScale, sqrt(2)≈1.41x plus margin) wastes
// a lot of white space in its own corners that the QR itself never uses,
// which reads as "the QR looks small and lost floating in a big blob"
// rather than immersed in the design — confirmed by user feedback on real
// rendered cards. A rounded square hugs the QR's actual shape instead, so
// far less margin (18%) is needed for the exact same "floats on its own
// card" language, and the QR reads noticeably bigger and better-integrated
// for the same overall halo footprint.
const qrSquareScale = 1.18

// qrHaloRadius is the rounded square's corner radius as a fraction of its
// own side length — enough to read as deliberately rounded, not sharp
// corners, without drifting toward a pill/circle and losing the tight fit
// qrSquareScale was chosen for.
const qrHaloRadius = 0.16

type cardColors struct {
	Background   string // solid hex, or the gradient's first stop when GradientTo is set
	GradientTo   string // second gradient stop; empty means Background is a flat solid fill
	Accent       string
	OnBackground string // white or near-black, whichever reads legibly directly on Background
	Pattern      string // "dots" | "lines" | "grid" | "waves" | "circles" | "" (none)
	Style        string // "block" | "split" | "" (defaults to block)
}

// cardColorOverrides is the shape of PrintCard.ColorOverrides JSON, when set.
// Solid colors are always a flat override; Pattern/Style live in the same
// bag since they're other per-card visual-style choices, not worth their
// own columns.
type cardColorOverrides struct {
	Background *string `json:"background"`
	Accent     *string `json:"accent"`
	Text       *string `json:"text"`
	Pattern    *string `json:"pattern"`
	Style      *string `json:"style"`
}

var hexColorRe = regexp.MustCompile(`#[0-9a-fA-F]{3,8}`)

// hexToRGB parses a #rgb or #rrggbb color into its 0-255 channels, defaulting
// to black on anything it can't parse (an unrecognized/empty color should
// never crash rendering — it just contrasts as if it were black).
func hexToRGB(hex string) (r, g, b int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) < 6 {
		return 0, 0, 0
	}
	ri, err1 := strconv.ParseInt(hex[0:2], 16, 0)
	gi, err2 := strconv.ParseInt(hex[2:4], 16, 0)
	bi, err3 := strconv.ParseInt(hex[4:6], 16, 0)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0
	}
	return int(ri), int(gi), int(bi)
}

// contrastingOnColor picks white or near-black, whichever reads legibly
// directly on top of the given background color(s) — needed because a
// business's brand color can be anything from a pale pastel to near-black,
// and the "bloque de color" style paints text straight onto that color
// instead of onto a neutral panel.
func contrastingOnColor(bg, bgTo string) string {
	r, g, b := hexToRGB(bg)
	if bgTo != "" {
		r2, g2, b2 := hexToRGB(bgTo)
		r, g, b = (r+r2)/2, (g+g2)/2, (b+b2)/2
	}
	brightness := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
	if brightness > 150 {
		return "#111827"
	}
	return "#ffffff"
}

func resolveCardColors(card *models.PrintCard, theme *models.ProfileTheme) cardColors {
	c := cardColors{Background: "#4f46e5", Accent: "#6366f1"}
	if theme != nil {
		// The card's bold color block deliberately reads from
		// theme.SecondaryColor (the button/accent color, used the same way
		// across ProfilePreview/BlockRenderer/LoyaltyCardView), not
		// theme.BackgroundValue — almost every profile page uses a neutral
		// or white page background with color carried by its buttons
		// instead, so background_value is not a reliable signal of "this is
		// the business's brand color" the way secondary_color already is.
		if theme.SecondaryColor != "" {
			c.Background = theme.SecondaryColor
			c.Accent = theme.SecondaryColor
		}
		// A gradient background IS a deliberate, colorful choice, though —
		// worth carrying over as-is when the business set one.
		if theme.BackgroundType == "gradient" {
			if stops := hexColorRe.FindAllString(theme.BackgroundValue, 2); len(stops) == 2 {
				c.Background, c.GradientTo = stops[0], stops[1]
			}
		}
	}
	c.OnBackground = contrastingOnColor(c.Background, c.GradientTo)

	if card.ColorOverrides != nil {
		var overrides cardColorOverrides
		if err := json.Unmarshal([]byte(*card.ColorOverrides), &overrides); err == nil {
			if overrides.Background != nil && *overrides.Background != "" {
				c.Background = *overrides.Background
				c.GradientTo = "" // an explicit override wins over any theme gradient
				c.OnBackground = contrastingOnColor(c.Background, "")
			}
			if overrides.Accent != nil && *overrides.Accent != "" {
				c.Accent = *overrides.Accent
			}
			if overrides.Text != nil && *overrides.Text != "" {
				c.OnBackground = *overrides.Text // explicit override always wins over the computed default
			}
			if overrides.Pattern != nil {
				c.Pattern = *overrides.Pattern
			}
			if overrides.Style != nil {
				c.Style = *overrides.Style
			}
		}
	}
	return c
}

// cardContent is the shape of PrintCard.Content JSON — a superset covering
// every layout's fields; each layout reads only the ones it uses.
type cardContent struct {
	Headline      string `json:"headline"`
	Subheadline   string `json:"subheadline"`
	Platform      string `json:"platform"`
	LeftLabel     string `json:"left_label"`
	RightLabel    string `json:"right_label"`
	DiscountCode  string `json:"discount_code"`  // thank_you layout only
	DiscountLabel string `json:"discount_label"` // thank_you layout only, e.g. "10% de descuento"
	// TopIcon picks what the top badge shows: "" / "platform" (default) for
	// the layout's own glyph (the Google G, the chosen social platform...),
	// or "logo" to show the business's own uploaded logo there instead —
	// same image as the small corner badge, just promoted to the main spot.
	TopIcon string `json:"top_icon"`
}

func parseCardContent(raw string) cardContent {
	var c cardContent
	_ = json.Unmarshal([]byte(raw), &c)
	return c
}

// estTextWidth is a rough heuristic for a bold sans-serif string's rendered
// width, avoiding a real font-metrics dependency — good enough to decide
// whether a headline needs to shrink or wrap, not to lay it out precisely.
func estTextWidth(s string, fontSize float64) float64 {
	return float64(len([]rune(s))) * fontSize * 0.56
}

// fitFontSize shrinks fontSize just enough that text fits availableWidth,
// down to a legibility floor — used for single-line labels that must never
// wrap (e.g. the multi_qr column headers).
func fitFontSize(text string, fontSize, availableWidth float64) float64 {
	width := estTextWidth(text, fontSize)
	if width <= availableWidth || width == 0 {
		return fontSize
	}
	return math.Max(fontSize*availableWidth/width, fontSize*0.55)
}

// wrapHeadline splits text into two lines at whichever word boundary leaves
// them closest in length.
func wrapHeadline(s string) (string, string) {
	words := strings.Fields(s)
	if len(words) < 2 {
		return s, ""
	}
	best, bestDiff := 1, math.MaxFloat64
	for i := 1; i < len(words); i++ {
		left, right := strings.Join(words[:i], " "), strings.Join(words[i:], " ")
		if diff := math.Abs(float64(len(left) - len(right))); diff < bestDiff {
			bestDiff, best = diff, i
		}
	}
	return strings.Join(words[:best], " "), strings.Join(words[best:], " ")
}

// fitHeadlineLines picks 1-2 lines of text and a font size so a headline
// never overflows the card's width — a long "¿Nos regalas una reseña?" on a
// narrow business_card would otherwise render straight past the card's
// edge, since SVG text has no built-in wrapping or auto-shrink.
func fitHeadlineLines(text string, baseFontSize, availableWidth float64) ([]string, float64) {
	if estTextWidth(text, baseFontSize) <= availableWidth {
		return []string{text}, baseFontSize
	}
	line1, line2 := wrapHeadline(text)
	if line2 == "" {
		return []string{text}, fitFontSize(text, baseFontSize, availableWidth)
	}
	longer := line1
	if len(line2) > len(line1) {
		longer = line2
	}
	return []string{line1, line2}, fitFontSize(longer, baseFontSize, availableWidth)
}

// discountBadgeHeight predicts how much vertical space drawDiscountBadge
// will use for the same (label, h) — the two must always agree, since
// renderSplitCard measures this BEFORE it knows where to draw anything (see
// its own doc comment on why), while renderSingleQRLayout draws immediately.
// Kept as its own function (constants duplicated between the two) rather
// than deriving one from the other, since the alternative — drawing once
// into a throwaway builder just to measure it — is exactly as much code for
// a real risk of silently drawing into the wrong builder.
func discountBadgeHeight(label string, h float64) float64 {
	extra := 0.0
	if label != "" {
		extra = h*0.034 + h*0.02
	}
	codeFontSize := h * 0.05
	return extra + h*0.015 + codeFontSize*1.9
}

func escapeXML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

// positionQRSVG takes a standalone QR <svg>...</svg> fragment (as returned
// by RenderSVG) and turns it into a positioned child element by injecting
// x/y/width/height onto its own root tag — SVG natively supports a nested
// <svg> as a viewport with its own placement, so the QR's internal
// module-unit viewBox keeps working unmodified.
func positionQRSVG(qrSVG string, x, y, size float64) string {
	attrs := fmt.Sprintf(`<svg x="%.2f" y="%.2f" width="%.2f" height="%.2f" `, x, y, size, size)
	return strings.Replace(qrSVG, "<svg ", attrs, 1)
}

// drawDualScanCaption renders "[icon] Toca | [icon] Escanea" as ONE pill —
// a single rounded, tinted-and-bordered capsule holding both methods,
// split by a thin vertical divider — rather than two separate floating
// icon+label pairs. Two earlier versions were tried and both fell short in
// review: bordered-but-separate pills read as an unbalanced layout
// accident, and dropping the pill treatment entirely (to win back width for
// a bigger font) read as "no design at all" — two disconnected scraps of
// text rather than a deliberate component. One pill gets both: a single
// unmistakably-designed shape, still reading as one cohesive "how to
// connect" unit instead of two things that happen to sit near each other.
// captionSegment is one "[icon] label" group inside a scan caption. Modelling
// the caption as a list of segments is what lets the same pill draw "Toca ·
// Escanea", just "Escanea", or a custom line, instead of the three separate
// hardcoded variants an earlier version would have needed.
type captionSegment struct {
	icon  string // an iconGlyph name, or "" for a label with no glyph
	label string
}

// captionWidth is what drawScanCaption would occupy at this font size — used
// to shrink an over-wide caption before drawing rather than letting it spill
// past its own element.
func captionWidth(segments []captionSegment, fontSize float64) float64 {
	if len(segments) == 0 {
		return 0
	}
	iconD := fontSize * 1.05
	innerGap := fontSize * 0.3
	dividerGap := fontSize * 0.5
	padX := fontSize * 0.55
	total := 0.0
	for _, seg := range segments {
		total += estTextWidth(seg.label, fontSize)
		if seg.icon != "" {
			total += iconD + innerGap
		}
	}
	total += dividerGap * 2 * float64(len(segments)-1)
	return total + padX*2
}

// fitCaptionFontSize shrinks a caption just enough to fit maxWidth, down to a
// legibility floor — below that it is better to overflow slightly than to
// print something nobody can read.
func fitCaptionFontSize(segments []captionSegment, fontSize, maxWidth float64) float64 {
	if maxWidth <= 0 {
		return fontSize
	}
	w := captionWidth(segments, fontSize)
	if w <= maxWidth || w == 0 {
		return fontSize
	}
	return math.Max(fontSize*maxWidth/w, fontSize*0.6)
}

// captionSegments turns a QR element's caption configuration into the
// segments to draw. An empty result means the caption is switched off.
//
// Toca and Escanea are independent prompts (models.QRProps.ShowTap/ShowScan),
// each with its own icon glyph and wording — a business can show either,
// both, or reword one without touching the other. CaptionMode == "" reads
// the pre-refactor single-Caption-string encoding instead, so a QR element
// saved before the two prompts became independent renders exactly as it did
// when it was printed.
func captionSegments(p models.QRProps) []captionSegment {
	if p.CaptionMode == "" && p.Caption != "" {
		return legacyCaptionSegments(p.Caption, p.CaptionText)
	}
	if p.CaptionMode == "text" {
		if p.CaptionText == "" {
			return nil
		}
		return []captionSegment{{label: p.CaptionText}}
	}
	if p.CaptionMode != "icons" {
		return nil
	}

	var segments []captionSegment
	if p.ShowTap {
		segments = append(segments, captionSegment{
			icon:  firstNonEmpty(p.TapIcon, "contactless"),
			label: firstNonEmpty(p.TapText, "Toca"),
		})
	}
	if p.ShowScan {
		segments = append(segments, captionSegment{
			icon:  firstNonEmpty(p.ScanIcon, "scan"),
			label: firstNonEmpty(p.ScanText, "Escanea"),
		})
	}
	return segments
}

// legacyCaptionSegments is the caption-style encoding print cards used before
// Toca and Escanea became independently configurable: one style string
// picking a fixed pairing, plus one optional text override.
func legacyCaptionSegments(style, custom string) []captionSegment {
	if style == "text" {
		if custom == "" {
			return nil
		}
		return []captionSegment{{label: custom}}
	}
	switch style {
	case "dual":
		return []captionSegment{
			{icon: "contactless", label: firstNonEmpty(custom, "Toca")},
			{icon: "scan", label: "Escanea"},
		}
	case "tap":
		return []captionSegment{{icon: "contactless", label: firstNonEmpty(custom, "Toca")}}
	case "scan":
		return []captionSegment{{icon: "scan", label: firstNonEmpty(custom, "Escanea")}}
	case "scan_me":
		return []captionSegment{{icon: "scan", label: firstNonEmpty(custom, "Escanéame")}}
	default:
		return nil
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// drawScanCaption renders the segments as ONE pill — a single rounded,
// tinted-and-bordered capsule, with a thin divider between segments — rather
// than separate floating icon+label pairs. Two earlier versions were tried and
// both fell short in review: bordered-but-separate pills read as an unbalanced
// layout accident, and dropping the pill entirely read as "no design at all",
// two disconnected scraps of text rather than a deliberate component.
//
// bare draws the labels with no pill behind them, for designs where the
// capsule competes with the card's own framing.
func drawScanCaption(b *strings.Builder, cx, cy, fontSize float64, color string, segments []captionSegment, bare bool) {
	if len(segments) == 0 {
		return
	}
	iconD := fontSize * 1.05
	innerGap := fontSize * 0.3
	dividerGap := fontSize * 0.5
	padX := fontSize * 0.55
	pillH := fontSize * 2.1
	textY := cy + fontSize*0.32
	iconCy := cy - fontSize*0.06
	strokeW := math.Max(fontSize*0.045, 1)

	widths := make([]float64, len(segments))
	contentW := 0.0
	for i, seg := range segments {
		w := estTextWidth(seg.label, fontSize)
		if seg.icon != "" {
			w += iconD + innerGap
		}
		widths[i] = w
		contentW += w
	}
	// Each gap between two segments costs two dividerGaps (one either side of
	// the divider line itself).
	contentW += dividerGap * 2 * float64(len(segments)-1)

	pillW := contentW + padX*2
	pillX := cx - pillW/2
	pillY := cy - pillH/2

	if !bare {
		fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.2f" fill="%s" opacity="0.16" stroke="%s" stroke-width="%.2f" stroke-opacity="0.55"/>`,
			pillX, pillY, pillW, pillH, pillH/2, color, color, strokeW)
	}

	x := pillX + padX
	for i, seg := range segments {
		if i > 0 {
			dividerX := x + dividerGap
			if !bare {
				fmt.Fprintf(b, `<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="%s" stroke-width="%.2f" opacity="0.4"/>`,
					dividerX, cy-pillH*0.3, dividerX, cy+pillH*0.3, color, strokeW)
			}
			x = dividerX + dividerGap
		}
		labelX := x
		if seg.icon != "" {
			iconCx := x + iconD/2
			fmt.Fprint(b, iconGlyph(seg.icon, iconCx, iconCy, iconD/2, color))
			labelX = iconCx + iconD/2 + innerGap
		}
		fmt.Fprintf(b, `<text x="%.2f" y="%.2f" font-size="%.2f" font-weight="700" fill="%s">%s</text>`,
			labelX, textY, fontSize, color, escapeXML(seg.label))
		x += widths[i]
	}
}

// lerpHex linearly blends two hex colors — used to derive a small family of
// related tints/shades from one brand accent color (see renderCornersCard)
// without needing full HSL conversion.
func lerpHex(a, b string, t float64) string {
	ar, ag, ab := hexToRGB(a)
	br, bg, bb := hexToRGB(b)
	lerp := func(x, y int) int { return x + int((float64(y)-float64(x))*t) }
	return fmt.Sprintf("#%02x%02x%02x", lerp(ar, br), lerp(ag, bg), lerp(ab, bb))
}

func defaultHeadline(layout models.PrintCardLayout) string {
	switch layout {
	case models.PrintCardGoogleReview:
		return "¿Nos regalas una reseña?"
	case models.PrintCardSocialFollow:
		return "Síguenos"
	case models.PrintCardMenuScan:
		return "Escanea el menú"
	case models.PrintCardLoyaltyCard:
		return "Junta sellos y gana"
	case models.PrintCardThankYou:
		return "¡Gracias por tu compra!"
	default:
		return "Escanéame"
	}
}
