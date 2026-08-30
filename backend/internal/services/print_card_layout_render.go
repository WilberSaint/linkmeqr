package services

import (
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"strings"

	"linkmeqr/backend/internal/models"
)

// This file is the whole of the print card renderer now: it walks a
// models.CardLayout element tree and emits SVG, and it knows nothing about
// "a Google review card" or "a thank-you card". Every position it draws at
// comes from the element it is drawing — there is no layout math here, by
// design. The old per-template arrangement logic survives only in
// print_card_layout_seed.go, where it generates a tree's INITIAL state.

// ImageAsset is one decoded image ready to be embedded as a data URI. The
// renderer never touches the filesystem or the media repository itself —
// the caller resolves everything up front into LayoutAssets, so rendering
// stays a pure function of (tree, assets) and is trivially testable.
type ImageAsset struct {
	Bytes    []byte
	MimeType string
}

// LayoutAssets carries everything a tree references but cannot itself
// contain: the rendered SVG for each QR element (keyed by element id, since
// each QR resolves its own destination and its own tracking URL), each
// image element's decoded file, and the business's logo for elements whose
// source is "logo".
type LayoutAssets struct {
	QRSVGs map[string]string
	Images map[string]ImageAsset
	Logo   *ImageAsset
}

func (a *LayoutAssets) qr(id string) string {
	if a == nil || a.QRSVGs == nil {
		return ""
	}
	return a.QRSVGs[id]
}

// image resolves an image element's actual file: an explicit upload for
// this element, or the shared business logo when the element is declared as
// source "logo".
func (a *LayoutAssets) image(elementID string, props models.ImageProps) *ImageAsset {
	if a == nil {
		return nil
	}
	if props.Source == "logo" {
		return a.Logo
	}
	if a.Images == nil {
		return nil
	}
	if img, ok := a.Images[elementID]; ok {
		return &img
	}
	return nil
}

// RenderLayoutSVG draws a card's element tree. This is the single render
// path behind both the live preview and the printable export, so what the
// designer drags into place is exactly what comes out of the exporter.
func RenderLayoutSVG(layout *models.CardLayout, assets *LayoutAssets) string {
	w, h := layout.Canvas.W, layout.Canvas.H

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %s %s" width="100%%" height="100%%" font-family="Arial, sans-serif">`,
		num(w), num(h))

	b.WriteString(`<defs>`)
	// Shared soft drop-shadow, referenced by any element that opts into
	// one — the same visual language the hardcoded cards used for their
	// floating white discs, now available to any element the designer wants
	// it on.
	b.WriteString(`<filter id="cardShadow" x="-60%" y="-60%" width="220%" height="220%"><feDropShadow dx="0" dy="3" stdDeviation="4" flood-color="#000000" flood-opacity="0.22"/></filter>`)
	if layout.Background.GradientTo != "" {
		fmt.Fprintf(&b, `<linearGradient id="cardBg" x1="0%%" y1="0%%" x2="100%%" y2="100%%"><stop offset="0%%" stop-color="%s"/><stop offset="100%%" stop-color="%s"/></linearGradient>`,
			attr(layout.Background.Fill), attr(layout.Background.GradientTo))
	}
	patternID := backgroundPatternDef(&b, layout)
	fmt.Fprintf(&b, `<clipPath id="cardClip"><rect width="%s" height="%s" rx="%s"/></clipPath>`, num(w), num(h), num(layout.Canvas.CornerR))
	// A die-cut hole is punched out of the background only (white=keep,
	// black=remove); elements are laid out to clear it rather than being
	// masked, so an element the designer deliberately places near the hole
	// still draws whole.
	if dc := layout.Canvas.DieCut; dc != nil {
		fmt.Fprintf(&b, `<mask id="dieCut"><rect width="%s" height="%s" fill="#ffffff"/><circle cx="%s" cy="%s" r="%s" fill="#000000"/></mask>`,
			num(w), num(h), num(dc.CX), num(dc.CY), num(dc.R))
	}
	b.WriteString(elementDefs(layout))
	b.WriteString(`</defs>`)

	maskAttr := ""
	if layout.Canvas.DieCut != nil {
		maskAttr = ` mask="url(#dieCut)"`
	}
	bgFill := layout.Background.Fill
	if bgFill == "" {
		bgFill = "#ffffff"
	}
	if layout.Background.GradientTo != "" {
		bgFill = "url(#cardBg)"
	}
	fmt.Fprintf(&b, `<rect width="%s" height="%s" rx="%s" fill="%s"%s/>`, num(w), num(h), num(layout.Canvas.CornerR), attr(bgFill), maskAttr)
	if patternID != "" {
		fmt.Fprintf(&b, `<rect width="%s" height="%s" rx="%s" fill="url(#%s)"%s/>`, num(w), num(h), num(layout.Canvas.CornerR), patternID, maskAttr)
	}

	// Everything the designer placed is clipped to the card's rounded
	// rectangle, so an element dragged past the edge bleeds off cleanly
	// instead of drawing outside the printable area.
	b.WriteString(`<g clip-path="url(#cardClip)">`)
	for _, el := range orderedElements(layout) {
		renderElement(&b, el, assets)
	}
	b.WriteString(`</g>`)

	// The cut guide is drawn last, on top of everything, so whoever cuts
	// the printed piece can always see it.
	if dc := layout.Canvas.DieCut; dc != nil {
		fmt.Fprintf(&b, `<circle cx="%s" cy="%s" r="%s" fill="none" stroke="#9ca3af" stroke-width="1" stroke-dasharray="4 3"/>`,
			num(dc.CX), num(dc.CY), num(dc.R))
	}

	b.WriteString(`</svg>`)
	return b.String()
}

// orderedElements returns the visible elements in paint order. It sorts
// defensively by ZIndex rather than trusting the slice order, because a
// tree can reach the renderer straight off a request body that
// NormalizeZ has not been run over yet (the stateless preview path).
func orderedElements(layout *models.CardLayout) []models.CardElement {
	out := make([]models.CardElement, 0, len(layout.Elements))
	for _, e := range layout.Elements {
		if !e.Hidden {
			out = append(out, e)
		}
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].ZIndex < out[j-1].ZIndex; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func backgroundPatternDef(b *strings.Builder, layout *models.CardLayout) string {
	ink := layout.Background.PatternInk
	if ink == "" {
		ink = contrastingOnColor(layout.Background.Fill, layout.Background.GradientTo)
	}
	base := math.Min(layout.Canvas.W, layout.Canvas.H)
	const id = "cardPattern"
	if !writePatternDef(b, id, layout.Background.Pattern, base, ink) {
		return ""
	}
	return id
}

// writePatternDef writes one texture's <pattern> def, sized off base (the
// shortest side of whatever box it will tile across), and reports whether
// anything was written. Shared by the card's own background — the only
// place a texture could go before shape elements could carry one too — and
// by shape elements, so both draw the exact same five textures at the exact
// same relative scale.
func writePatternDef(b *strings.Builder, id, kind string, base float64, ink string) bool {
	switch kind {
	case "dots":
		b.WriteString(dotPatternDef(id, base*0.045, ink, 0.14))
	case "lines":
		b.WriteString(linePatternDef(id, base*0.05, ink, 0.12))
	case "grid":
		b.WriteString(gridPatternDef(id, base*0.05, ink, 0.16))
	case "waves":
		b.WriteString(wavesPatternDef(id, base*0.04, ink, 0.16))
	case "circles":
		b.WriteString(circlesPatternDef(id, base*0.05, ink, 0.16))
	default:
		return false
	}
	return true
}

// elementDefs emits the per-element <clipPath>s image elements need and the
// per-element <pattern>s a textured shape needs. They have to live in
// <defs> before first use, and each is namespaced by element id so two
// elements on the same card never collide.
func elementDefs(layout *models.CardLayout) string {
	var b strings.Builder
	for _, e := range layout.Elements {
		if e.Hidden {
			continue
		}
		switch e.Type {
		case models.ElementImage:
			var p models.ImageProps
			_ = e.DecodeProps(&p)
			fmt.Fprintf(&b, `<clipPath id="clip-%s">%s</clipPath>`, attr(e.ID), imageClipShape(p, e.W, e.H))
		case models.ElementShape:
			var p models.ShapeProps
			_ = e.DecodeProps(&p)
			if p.Pattern == "" {
				continue
			}
			ink := p.PatternInk
			if ink == "" {
				ink = p.Fill
			}
			if ink == "" {
				ink = "#111827"
			}
			writePatternDef(&b, shapePatternID(e.ID), p.Pattern, math.Min(e.W, e.H), ink)
		}
	}
	return b.String()
}

// shapePatternID namespaces a shape's texture def by its element id, so two
// textured shapes on the same card never collide. Escaped the same way any
// other element-id-derived attribute value is.
func shapePatternID(elementID string) string {
	return "shape-pattern-" + attr(elementID)
}

// imageClipShape mirrors the three logo shapes the profile editor's crop
// modal offers, expressed in the element's own local box coordinates.
func imageClipShape(p models.ImageProps, w, h float64) string {
	switch p.Shape {
	case "square":
		return fmt.Sprintf(`<rect width="%s" height="%s"/>`, num(w), num(h))
	case "rounded":
		r := p.Radius
		if r <= 0 {
			r = math.Min(w, h) * 0.14
		}
		return fmt.Sprintf(`<rect width="%s" height="%s" rx="%s"/>`, num(w), num(h), num(r))
	case "circle":
		return fmt.Sprintf(`<ellipse cx="%s" cy="%s" rx="%s" ry="%s"/>`, num(w/2), num(h/2), num(w/2), num(h/2))
	default:
		// No shape chosen means no clipping at all — a plain rectangular
		// image, which is what a designer dropping a photo onto the canvas
		// expects. Only the logo badge opts into a shape.
		return fmt.Sprintf(`<rect width="%s" height="%s"/>`, num(w), num(h))
	}
}

// renderElement wraps one element in its own local coordinate system: the
// group is translated to the element's top-left and rotated about its own
// center, so every type-specific renderer below can draw at 0,0..W,H and
// never think about placement or rotation again. This is what makes drag,
// resize and rotate uniform across every element type.
func renderElement(b *strings.Builder, e models.CardElement, assets *LayoutAssets) {
	transform := fmt.Sprintf("translate(%s,%s)", num(e.X), num(e.Y))
	if e.Rotation != 0 {
		transform += fmt.Sprintf(" rotate(%s,%s,%s)", num(e.Rotation), num(e.W/2), num(e.H/2))
	}
	opacity := ""
	if o := e.EffectiveOpacity(); o < 1 {
		opacity = fmt.Sprintf(` opacity="%s"`, num(o))
	}
	fmt.Fprintf(b, `<g transform="%s"%s>`, transform, opacity)

	switch e.Type {
	case models.ElementText:
		renderTextElement(b, e)
	case models.ElementImage:
		renderImageElement(b, e, assets)
	case models.ElementQR:
		renderQRElement(b, e, assets)
	case models.ElementShape:
		renderShapeElement(b, e)
	case models.ElementIcon:
		renderIconElement(b, e)
	case models.ElementPrompt:
		renderPromptElement(b, e)
	}

	b.WriteString(`</g>`)
}

func renderTextElement(b *strings.Builder, e models.CardElement) {
	var p models.TextProps
	_ = e.DecodeProps(&p)
	if p.Text == "" {
		return
	}
	text := p.Text
	if p.Uppercase {
		text = strings.ToUpper(text)
	}
	fontSize := p.FontSize
	if fontSize <= 0 {
		fontSize = e.H * 0.6
	}
	lineHeight := p.LineHeight
	if lineHeight <= 0 {
		lineHeight = 1.2
	}
	weight := p.FontWeight
	if weight <= 0 {
		weight = 400
	}
	color := p.Color
	if color == "" {
		color = "#111827"
	}
	family := p.FontFamily
	if family == "" {
		family = "Arial, sans-serif"
	}

	// Alignment resolves to an anchor plus an x within the element's own
	// box, so the text stays put relative to its box under any alignment —
	// which is what a designer expects when they resize a text element.
	anchor, x := "middle", e.W/2
	switch p.Align {
	case "left":
		anchor, x = "start", 0
	case "right":
		anchor, x = "end", e.W
	}

	// Explicit newlines are the only wrapping. The old renderer auto-wrapped
	// and auto-shrank headlines because it had to guess at a size it never
	// controlled; here the designer sets the box and the breaks, so guessing
	// would fight them.
	lines := strings.Split(text, "\n")
	gap := fontSize * lineHeight
	// Vertically center the block of lines within the box, with the first
	// baseline offset by roughly the cap height.
	blockH := gap * float64(len(lines)-1)
	firstBaseline := (e.H-blockH)/2 + fontSize*0.34

	extra := ""
	if p.Letter != 0 {
		extra += fmt.Sprintf(` letter-spacing="%s"`, num(p.Letter))
	}
	if p.Italic {
		extra += ` font-style="italic"`
	}
	if p.Underline {
		extra += ` text-decoration="underline"`
	}

	for i, line := range lines {
		fmt.Fprintf(b, `<text x="%s" y="%s" font-family="%s" font-size="%s" font-weight="%d" fill="%s" text-anchor="%s"%s>%s</text>`,
			num(x), num(firstBaseline+float64(i)*gap), attr(family), num(fontSize), weight, attr(color), anchor, extra, escapeXML(line))
	}
}

func renderImageElement(b *strings.Builder, e models.CardElement, assets *LayoutAssets) {
	var p models.ImageProps
	_ = e.DecodeProps(&p)
	img := assets.image(e.ID, p)
	if img == nil || len(img.Bytes) == 0 {
		return
	}
	preserve := "xMidYMid slice"
	if p.Fit == "contain" {
		preserve = "xMidYMid meet"
	}
	encoded := base64.StdEncoding.EncodeToString(img.Bytes)
	fmt.Fprintf(b, `<image width="%s" height="%s" href="data:%s;base64,%s" preserveAspectRatio="%s" clip-path="url(#clip-%s)"/>`,
		num(e.W), num(e.H), attr(img.MimeType), encoded, preserve, attr(e.ID))
}

// renderQRElement draws the QR plus the two decorations that belong to it
// as a component rather than as separate draggable parts: its frame (the
// white rounded square it floats on) and its optional scan caption. Both
// are QR properties in the editor, so a designer moves "the QR" as one
// thing instead of having to keep three elements aligned by hand.
func renderQRElement(b *strings.Builder, e models.CardElement, assets *LayoutAssets) {
	var p models.QRProps
	_ = e.DecodeProps(&p)
	qrSVG := assets.qr(e.ID)

	// The QR itself is always square and centered in its box — a stretched
	// QR is an unscannable QR, so this is the one place the tree's own w/h
	// is deliberately not honored literally.
	side := math.Min(e.W, e.H)
	pad := p.FramePad
	if pad <= 0 {
		pad = 0.18
	}
	qrSize := side / (1 + pad)
	cx, cy := e.W/2, e.H/2

	if p.Frame != "" && p.Frame != "none" {
		fill := p.FrameFill
		if fill == "" {
			fill = "#ffffff"
		}
		radius := p.FrameRadius
		if p.Frame == "square" {
			radius = 0
		} else if radius <= 0 {
			radius = 0.16
		}
		shadow := ""
		if p.Shadow {
			shadow = ` filter="url(#cardShadow)"`
		}
		fmt.Fprintf(b, `<rect x="%s" y="%s" width="%s" height="%s" rx="%s" fill="%s"%s/>`,
			num(cx-side/2), num(cy-side/2), num(side), num(side), num(side*radius), attr(fill), shadow)
	}

	if qrSVG != "" {
		fmt.Fprint(b, positionQRSVG(qrSVG, cx-qrSize/2, cy-qrSize/2, qrSize))
	} else {
		// Placeholder so an element whose destination could not be resolved
		// still shows up in the editor as a real, selectable object rather
		// than vanishing and leaving the designer with an invisible layer.
		fmt.Fprintf(b, `<rect x="%s" y="%s" width="%s" height="%s" fill="#e5e7eb" stroke="#9ca3af" stroke-width="1" stroke-dasharray="4 3"/>`,
			num(cx-qrSize/2), num(cy-qrSize/2), num(qrSize), num(qrSize))
	}

	if segments := captionSegments(p); len(segments) > 0 {
		fontSize := p.CaptionSize
		if fontSize <= 0 {
			fontSize = side * 0.12
		}
		fill := p.CaptionFill
		if fill == "" {
			fill = "#111827"
		}
		// A caption wider than the space it was given overlaps whatever sits
		// beside it — the failure two QRs side by side hit every time.
		maxWidth := p.CaptionMaxWidth
		if maxWidth <= 0 {
			maxWidth = e.W
		}
		fontSize = fitCaptionFontSize(segments, fontSize, maxWidth)
		captionCY := cy + side/2 + fontSize*1.9
		drawScanCaption(b, cx, captionCY, fontSize, fill, segments, p.CaptionBare)
	}
}

func renderShapeElement(b *strings.Builder, e models.CardElement) {
	var p models.ShapeProps
	_ = e.DecodeProps(&p)

	fill := p.Fill
	if fill == "" {
		fill = "none"
	}
	style := fmt.Sprintf(` fill="%s"`, attr(fill))
	if p.Stroke != "" && p.StrokeWidth > 0 {
		style += fmt.Sprintf(` stroke="%s" stroke-width="%s"`, attr(p.Stroke), num(p.StrokeWidth))
		if p.DashArray != "" {
			style += fmt.Sprintf(` stroke-dasharray="%s"`, attr(p.DashArray))
		}
	}
	if p.Shadow {
		style += ` filter="url(#cardShadow)"`
	}
	drawShapeGeometry(b, e, p, style)

	// The texture layers on top of the solid fill, exactly the way the
	// card's own background does it — a shape keeps its colour and gains a
	// subtle pattern over it, rather than the pattern replacing the colour
	// outright. Redrawing the SAME geometry (same kind, same radius, same
	// path) is what keeps the texture confined to the shape's own outline —
	// no separate clip needed. No stroke on this pass: the base shape
	// already drew the border once.
	if p.Pattern != "" {
		drawShapeGeometry(b, e, p, fmt.Sprintf(` fill="url(#%s)"`, shapePatternID(e.ID)))
	}
}

// drawShapeGeometry emits the actual path/rect/etc for one shape, given a
// pre-built style attribute string — factored out so the texture overlay
// pass draws exactly the same geometry as the base fill pass, just with a
// different fill.
func drawShapeGeometry(b *strings.Builder, e models.CardElement, p models.ShapeProps, style string) {
	switch p.Kind {
	case "ellipse":
		fmt.Fprintf(b, `<ellipse cx="%s" cy="%s" rx="%s" ry="%s"%s/>`, num(e.W/2), num(e.H/2), num(e.W/2), num(e.H/2), style)
	case "line":
		fmt.Fprintf(b, `<line x1="0" y1="%s" x2="%s" y2="%s"%s/>`, num(e.H/2), num(e.W), num(e.H/2), style)
	case "polygon":
		fmt.Fprintf(b, `<polygon points="%s"%s/>`, attr(p.Points), style)
	case "path":
		fmt.Fprintf(b, `<path d="%s"%s/>`, attr(p.Path), style)
	default:
		fmt.Fprintf(b, `<rect width="%s" height="%s" rx="%s"%s/>`, num(e.W), num(e.H), num(p.Radius), style)
	}
}

// renderIconElement draws one of the built-in glyphs, or a row of them when
// Count > 1 — the rating stars are a single element the designer moves as a
// unit, not five separate layers to keep aligned.
func renderIconElement(b *strings.Builder, e models.CardElement) {
	var p models.IconProps
	_ = e.DecodeProps(&p)
	count := p.Count
	if count < 1 {
		count = 1
	}
	color := p.Color
	if color == "" {
		color = "#111827"
	}

	if count == 1 {
		r := math.Min(e.W, e.H) / 2
		fmt.Fprint(b, iconGlyph(p.Name, e.W/2, e.H/2, r, color))
		return
	}

	gap := p.Gap
	if gap <= 0 {
		gap = 1.3
	}
	// Fit the whole row inside the element's width: with count glyphs of
	// diameter d spaced gap*d apart, the row spans d*(1 + gap*(count-1)).
	d := e.W / (1 + gap*float64(count-1))
	r := math.Min(d/2, e.H/2)
	spacing := d * gap
	startX := e.W/2 - spacing*float64(count-1)/2
	for i := 0; i < count; i++ {
		fmt.Fprint(b, iconGlyph(p.Name, startX+float64(i)*spacing, e.H/2, r, color))
	}
}

// renderPromptElement draws one "[icon] label" call-to-action — "Toca",
// "Escanea", or any other short instruction — scaled to fill its own box the
// same way a text or icon element does, so resizing it in the editor resizes
// what it draws. It was folded into a QR element's own rendering before
// prompts became independent; this is a straight port of that geometry
// (drawScanCaption in print_card_render.go) onto ONE segment inside a box
// instead of N segments centred on a point, which is the only thing that
// actually changed.
func renderPromptElement(b *strings.Builder, e models.CardElement) {
	var p models.PromptProps
	_ = e.DecodeProps(&p)
	if p.Text == "" && p.Icon == "" {
		return
	}

	fontSize := p.FontSize
	if fontSize <= 0 {
		fontSize = e.H / 1.4
	}
	color := p.Color
	if color == "" {
		color = "#111827"
	}

	measure := func(size float64) float64 {
		w := estTextWidth(p.Text, size)
		if p.Icon != "" {
			w += size*1.05 + size*0.3
		}
		return w
	}
	// Shrink to fit the box's own width, the same floor fitCaptionFontSize
	// uses — below it, an overflowing label is more legible than a
	// vanishingly small one.
	if contentW := measure(fontSize); contentW > e.W && contentW > 0 {
		fontSize = math.Max(fontSize*e.W/contentW, fontSize*0.6)
	}

	iconD := fontSize * 1.05
	innerGap := fontSize * 0.3
	contentW := measure(fontSize)
	cy := e.H / 2

	if !p.Bare {
		padX := fontSize * 0.55
		pillH := fontSize * 2.1
		pillW := contentW + padX*2
		strokeW := math.Max(fontSize*0.045, 1)
		fmt.Fprintf(b, `<rect x="%s" y="%s" width="%s" height="%s" rx="%s" fill="%s" opacity="0.16" stroke="%s" stroke-width="%s" stroke-opacity="0.55"/>`,
			num((e.W-pillW)/2), num(cy-pillH/2), num(pillW), num(pillH), num(pillH/2), attr(color), attr(color), num(strokeW))
	}

	x := (e.W - contentW) / 2
	if p.Icon != "" {
		iconCx := x + iconD/2
		fmt.Fprint(b, iconGlyph(p.Icon, iconCx, cy, iconD/2, color))
		x = iconCx + iconD/2 + innerGap
	}
	if p.Text != "" {
		fmt.Fprintf(b, `<text x="%s" y="%s" font-size="%s" font-weight="700" fill="%s">%s</text>`,
			num(x), num(cy+fontSize*0.32), num(fontSize), attr(color), escapeXML(p.Text))
	}
}

// IconGlyphSVG is iconGlyph exported for the editor's icon endpoint, so the
// glyph a designer positions on screen is literally the same artwork the
// exporter prints.
func IconGlyphSVG(name string, cx, cy, r float64, color string) string {
	return iconGlyph(name, cx, cy, r, color)
}

// iconGlyph is the one lookup mapping a stored icon name to the drawing
// helpers in print_card_icons.go. Unlike the old drawLayoutIcon, it is
// keyed by the icon the designer chose — not by which template the card is,
// which no longer exists as a rendering concept.
func iconGlyph(name string, cx, cy, r float64, color string) string {
	switch name {
	case "google":
		return googleGIcon(cx, cy, r)
	case "instagram":
		return instagramIcon(cx, cy, r)
	case "facebook":
		return facebookIcon(cx, cy, r)
	case "whatsapp":
		return whatsappIcon(cx, cy, r)
	case "youtube":
		return youtubeIcon(cx, cy, r)
	case "tiktok":
		return tiktokIcon(cx, cy, r)
	case "star":
		return starIcon(cx, cy, r, color)
	case "menu":
		return menuGlyph(cx, cy, r, color)
	case "gift":
		return giftGlyph(cx, cy, r, color)
	case "heart":
		return heartGlyph(cx, cy, r, color)
	case "contactless":
		return contactlessIcon(cx, cy, r, color)
	case "tap_hand":
		return tapHandIcon(cx, cy, r, color)
	case "tap_card":
		return tapCardIcon(cx, cy, r, color)
	case "scan":
		return scanTargetIcon(cx, cy, r, color)
	case "scan_qr":
		return scanQRIcon(cx, cy, r, color)
	case "scan_camera":
		return scanCameraIcon(cx, cy, r, color)
	case "pin":
		return pinGlyph(cx, cy, r, color)
	default:
		return platformIcon(cx, cy, r, name)
	}
}

// num formats a coordinate compactly. Two decimals is well below one
// thousandth of an inch at this scale, and trimming the trailing zeros
// keeps a tree of a few dozen elements from bloating the exported SVG.
func num(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	s := strconv.FormatFloat(v, 'f', 2, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

// attr escapes a value destined for an XML attribute. Element ids, colors
// and paths all originate in a request body, so none of them can be
// interpolated into markup unescaped.
func attr(s string) string {
	return escapeXML(s)
}
