package services

import (
	"encoding/json"
	"fmt"
	"math"

	"linkmeqr/backend/internal/models"
)

// This file is where the six built-in card designs now live: NOT as a
// rendering path, but as generators of a layout tree's INITIAL state. The
// arithmetic below is a faithful port of the old imperative renderers
// (renderSingleQRLayout, renderMultiQRLayout, renderSplitCard,
// renderCornersCard, renderFramedCard), preserved constant-for-constant so
// that backfilling an existing card produces a tree which renders
// pixel-identically to what that card printed before the refactor.
//
// Once a tree exists, this file is never consulted again for that card —
// the tree is authoritative, and the designer is free to move anything.

// seedCtx bundles the values every element builder below needs, so the
// ported math reads as closely as possible to the original functions it
// came from.
type seedCtx struct {
	colors  cardColors
	content cardContent
	layout  models.PrintCardLayout
	hasLogo bool
	// logoShape mirrors the business's chosen crop shape so a card that
	// promotes the logo to its top badge clips it the same way the profile
	// does.
	logoShape string
	// qrTargets are the destinations to stamp onto the QR elements, in the
	// order the design places them.
	qrTargets []seedQRTarget

	elements []models.CardElement
	z        int
}

type seedQRTarget struct {
	TargetType  models.QRTargetType
	TargetValue *string
	// LegacySlot is the /q/:code/:slot segment already printed on physical
	// cards for this QR position: "left"/"right" for the two-QR design, and
	// the empty string for a single-QR card, whose printed URL carried no
	// segment at all. Always non-nil here — every card seeded from the
	// pre-tree fields is one whose earlier exports may already be on paper.
	LegacySlot *string
}

func (s *seedCtx) add(e models.CardElement) {
	e.ZIndex = s.z
	s.z++
	s.elements = append(s.elements, e)
}

func props(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("{}")
	}
	return raw
}

func fptr(v float64) *float64 { return &v }

// CardSizeUnits resolves a card's physical size into the layout tree's
// coordinate units, applying the same custom-size and unknown-preset
// fallbacks the renderer has always used.
func CardSizeUnits(card *models.PrintCard) (w, h float64) {
	var widthIn, heightIn float64
	if card.SizePreset == models.SizeCustom && card.CustomWidthCm != nil && card.CustomHeightCm != nil {
		widthIn = *card.CustomWidthCm / cmPerInch
		heightIn = *card.CustomHeightCm / cmPerInch
	} else if preset, ok := SizePresets[card.SizePreset]; ok {
		widthIn, heightIn = preset.WidthIn, preset.HeightIn
	} else {
		preset = SizePresets[models.SizeBusinessCard]
		widthIn, heightIn = preset.WidthIn, preset.HeightIn
	}
	return widthIn * cardUnitsPerInch, heightIn * cardUnitsPerInch
}

// SeedCardLayout builds the element tree a card starts life with, from the
// pre-tree fields (layout_key, size, content, color overrides) plus the
// business's theme. It is used in exactly two places: backfilling cards
// created before the tree existed, and giving a newly created card its
// starting design.
func SeedCardLayout(card *models.PrintCard, theme *models.ProfileTheme, hasLogo bool, logoShape string) *models.CardLayout {
	w, h := CardSizeUnits(card)
	colors := resolveCardColors(card, theme)
	content := parseCardContent(card.Content)

	s := &seedCtx{
		colors:    colors,
		content:   content,
		layout:    card.LayoutKey,
		hasLogo:   hasLogo,
		logoShape: logoShape,
		qrTargets: seedQRTargetsFor(card, content),
	}

	canvas := models.CardCanvas{W: w, H: h, CornerR: math.Min(w, h) * 0.07}
	holeClearance := 0.0
	if card.SizePreset == models.SizeDoorHanger {
		hcx, hcy, hr := doorHangerHole(w, h)
		canvas.DieCut = &models.DieCut{Kind: "circle", CX: hcx, CY: hcy, R: hr}
		holeClearance = hcy + hr + h*0.04
	}

	background := models.CardBackground{
		Fill:       colors.Background,
		GradientTo: colors.GradientTo,
		Pattern:    colors.Pattern,
		PatternInk: colors.OnBackground,
	}

	// "split", "corners" and "framed" only ever applied to single-QR
	// designs — multi_qr already spends its own left/right split saying
	// "two things here". That fallback is preserved so a migrated multi_qr
	// card keeps the arrangement it was actually printed with.
	singleTarget := card.LayoutKey != models.PrintCardMultiQR
	switch {
	case !singleTarget:
		background = s.seedMultiQRStyled(w, h, canvas.CornerR, holeClearance, colors.Style, background)
	case colors.Style == "split" && singleTarget:
		s.seedSplit(w, h, holeClearance)
	case colors.Style == "corners" && singleTarget:
		background = models.CardBackground{Fill: cornersDarkBase}
		s.seedCorners(w, h, holeClearance)
	case colors.Style == "framed" && singleTarget:
		background = models.CardBackground{Fill: colors.Accent, GradientTo: colors.GradientTo}
		s.seedFramed(w, h, canvas.CornerR, holeClearance)
	case colors.Style == "banner" && singleTarget:
		background = models.CardBackground{Fill: bannerBase}
		s.seedBanner(w, h, holeClearance)
	case colors.Style == "spotlight" && singleTarget:
		background = models.CardBackground{Fill: spotlightBase}
		s.seedSpotlight(w, h, holeClearance)
	case colors.Style == "diagonal" && singleTarget:
		background = models.CardBackground{Fill: bannerBase}
		s.seedDiagonal(w, h, holeClearance)
	case colors.Style == "outline" && singleTarget:
		background = models.CardBackground{Fill: outlineBase}
		panelMargin := s.seedOutline(w, h, canvas.CornerR, holeClearance)
		outlineColors := colors
		outlineColors.OnBackground = outlineInk
		panelY := math.Max(panelMargin, holeClearance)
		s.seedSingleQR(panelMargin, panelY, w-2*panelMargin, h-panelY-panelMargin, outlineColors)
	case colors.Style == "pattern" && singleTarget:
		patternKind := colors.Pattern
		if patternKind == "" {
			patternKind = "dots"
		}
		background = models.CardBackground{Fill: colors.Background, GradientTo: colors.GradientTo, Pattern: patternKind, PatternInk: colors.OnBackground}
		s.seedPatternPanel(w, h, canvas.CornerR, holeClearance)
	default:
		border := math.Min(w, h) * 0.06
		panelY := math.Max(border, holeClearance)
		s.seedSingleQR(border, panelY, w-2*border, h-panelY-border, colors)
	}

	layout := &models.CardLayout{
		Version:    models.CardLayoutVersion,
		Canvas:     canvas,
		Background: background,
		Elements:   s.elements,
		SeededFrom: string(card.LayoutKey),
	}
	layout.NormalizeZ()
	return layout
}

// seedQRTargetsFor lists a card's QR destinations in placement order,
// carrying the slot names their earlier exports already printed, so cards
// physically in the wild keep redirecting: "left"/"right" for the two-QR
// design, the empty segment for every single-QR one.
func seedQRTargetsFor(card *models.PrintCard, content cardContent) []seedQRTarget {
	if card.LayoutKey != models.PrintCardMultiQR {
		root := ""
		return []seedQRTarget{{TargetType: card.QRTargetType, TargetValue: card.QRTargetValue, LegacySlot: &root}}
	}
	var targets struct {
		LeftTargetType   string  `json:"left_target_type"`
		LeftTargetValue  *string `json:"left_target_value"`
		RightTargetType  string  `json:"right_target_type"`
		RightTargetValue *string `json:"right_target_value"`
	}
	_ = json.Unmarshal([]byte(card.Content), &targets)
	left := models.QRTargetType(targets.LeftTargetType)
	if left == "" {
		left = models.QRTargetProfile
	}
	right := models.QRTargetType(targets.RightTargetType)
	if right == "" {
		right = models.QRTargetProfile
	}
	leftSlot, rightSlot := "left", "right"
	return []seedQRTarget{
		{TargetType: left, TargetValue: targets.LeftTargetValue, LegacySlot: &leftSlot},
		{TargetType: right, TargetValue: targets.RightTargetValue, LegacySlot: &rightSlot},
	}
}

func (s *seedCtx) target(i int) seedQRTarget {
	if i < len(s.qrTargets) {
		return s.qrTargets[i]
	}
	return seedQRTarget{TargetType: models.QRTargetProfile}
}

// ---------------------------------------------------------------------------
// Shared element builders
// ---------------------------------------------------------------------------

// minQRSize is the floor a seeded QR is never allowed to fall below.
//
// The old renderers derived the QR from "whatever vertical space is left"
// and clamped a negative result to zero — so a small card carrying both a
// long headline and a discount coupon printed with NO VISIBLE QR at all.
// That silent failure is why the clamp is a floor here instead: an
// overlapping QR is obviously wrong at a glance and can now be dragged
// clear, whereas a missing one looked like a design choice until the cards
// came back from the printer unscannable.
//
// This can only affect cards whose QR previously rendered at size zero, so
// it cannot alter any card that was actually printable before.
func seedQRSize(available, w, h float64) float64 {
	size := available / qrSquareScale
	floor := math.Min(w, h) * 0.28
	if size < floor {
		return floor
	}
	return size
}

// seedQRSizeIn is seedQRSize without the floor: the code fills the space it
// was given and no more.
//
// Use it wherever the caller has already subtracted everything above the QR
// from the space it passes in. The floor exists to stop a code collapsing to
// nothing, but it cannot be allowed to win against a real measurement — doing
// so is what made a thank-you card's QR grow up over its own discount code.
func seedQRSizeIn(available float64) float64 {
	size := available / qrSquareScale
	if size < 1 {
		return 1
	}
	return size
}

// addTopBadge emits the white disc plus whichever glyph sits on it — the
// business's own logo when the card was configured that way and one is on
// file, otherwise the design's own icon. Two elements, not one, so the
// designer can recolor, resize or delete the disc independently of what
// sits on it.
func (s *seedCtx) addTopBadge(cx, cy, iconR float64) {
	haloR := iconR * qrHaloScale
	s.add(models.CardElement{
		ID: "badge-halo", Type: models.ElementShape, Name: "Fondo del ícono",
		X: cx - haloR, Y: cy - haloR, W: haloR * 2, H: haloR * 2,
		Props: props(models.ShapeProps{Kind: "ellipse", Fill: "#ffffff", Shadow: true}),
	})

	useLogo := s.content.TopIcon == "logo" && s.hasLogo
	if useLogo {
		shape := s.logoShape
		if shape == "" {
			shape = "circle"
		}
		s.add(models.CardElement{
			ID: "badge-icon", Type: models.ElementImage, Name: "Logo del negocio",
			X: cx - iconR, Y: cy - iconR, W: iconR * 2, H: iconR * 2,
			Props: props(models.ImageProps{Source: "logo", Fit: "cover", Shape: shape}),
		})
		return
	}
	s.add(models.CardElement{
		ID: "badge-icon", Type: models.ElementIcon, Name: "Ícono",
		X: cx - iconR, Y: cy - iconR, W: iconR * 2, H: iconR * 2,
		Props: props(models.IconProps{Name: seedIconName(s.layout, s.content.Platform), Color: s.colors.Accent, Count: 1}),
	})
}

// seedIconName is the old drawLayoutIcon switch, turned into stored data.
// After seeding, the icon is just a name in the tree that the designer can
// change — the design's identity no longer decides it.
func seedIconName(layout models.PrintCardLayout, platform string) string {
	switch layout {
	case models.PrintCardGoogleReview:
		return "google"
	case models.PrintCardSocialFollow:
		if platform == "" {
			return "instagram"
		}
		return platform
	case models.PrintCardMenuScan:
		return "menu"
	case models.PrintCardLoyaltyCard:
		return "gift"
	case models.PrintCardThankYou:
		return "heart"
	default:
		return "pin"
	}
}

// addTextAt places a text element whose FIRST BASELINE lands exactly on
// baselineY, centered on cx across boxW. The old renderer emitted <text>
// with an explicit baseline; the tree stores boxes instead, so this inverts
// renderTextElement's own vertical centering to keep migrated text on the
// exact pixel it was printed at.
func (s *seedCtx) addTextAt(id, name string, cx, baselineY, boxW float64, p models.TextProps, lineCount int, opacity float64) {
	if p.LineHeight <= 0 {
		p.LineHeight = 1.2
	}
	if lineCount < 1 {
		lineCount = 1
	}
	gap := p.FontSize * p.LineHeight
	blockH := gap * float64(lineCount-1)
	boxH := blockH + p.FontSize*1.4
	// renderTextElement puts the first baseline at (H-blockH)/2 + size*0.34;
	// with H chosen as above that simplifies to size*1.04 from the top.
	y := baselineY - p.FontSize*1.04
	el := models.CardElement{
		ID: id, Type: models.ElementText, Name: name,
		X: cx - boxW/2, Y: y, W: boxW, H: boxH,
		Props: props(p),
	}
	if opacity < 1 {
		el.Opacity = fptr(opacity)
	}
	s.add(el)
}

// addQR places a QR whose white frame lands exactly where drawQRWithFrame
// used to draw it. qrTop/qrSize are that function's own parameters, so the
// conversion below is the only place the old "top of the QR itself" and the
// tree's "box of the whole component" conventions have to be reconciled.
//
// It no longer places any caption of its own — Toca and Escanea are sibling
// prompt elements now (see addPromptPair), added separately by the caller
// right after this, so they exist as their own selectable, movable layers
// rather than baked into the QR's props.
func (s *seedCtx) addQR(id, name string, cx, qrTop, qrSize float64, targetIdx int) {
	frameSide := qrSize * qrSquareScale
	centerY := qrTop + qrSize/2
	t := s.target(targetIdx)
	s.add(models.CardElement{
		ID: id, Type: models.ElementQR, Name: name,
		X: cx - frameSide/2, Y: centerY - frameSide/2, W: frameSide, H: frameSide,
		Props: props(models.QRProps{
			TargetType:  t.TargetType,
			TargetValue: t.TargetValue,
			LegacySlot:  t.LegacySlot,
			Frame:       "rounded",
			FrameFill:   "#ffffff",
			FrameRadius: qrHaloRadius,
			// qrSquareScale is the frame/QR ratio, so the padding fraction
			// that reproduces it is exactly that scale minus one.
			FramePad: qrSquareScale - 1,
			Shadow:   true,
		}),
	})
}

// promptRowCY is the vertical center a QR's Toca/Escanea row sits at, given
// the same (qrTop, qrSize) addQR was called with. It is the exact formula
// the old QR-embedded caption used to place itself at (frame bottom edge +
// fontSize*1.9), kept as its own function so every call site agrees with
// every other on where "under this QR" means.
func promptRowCY(qrTop, qrSize, fontSize float64) float64 {
	frameSide := qrSize * qrSquareScale
	frameBottom := qrTop + qrSize/2 + frameSide/2
	return frameBottom + fontSize*1.9
}

// addPromptPair places Toca and/or Escanea as independent elements centred
// as a pair on rowCY, sharing idPrefix so multiple QRs on the same card
// (multi_qr) each get their own pair of ids. Passing only one of
// showTap/showScan places a single centred prompt instead of a pair — the
// case a card design uses when its own layout (a column label above the
// code, say) already says everything Toca would.
func (s *seedCtx) addPromptPair(idPrefix string, cx, rowCY, fontSize float64, fill string, showTap, showScan bool, maxWidth float64) {
	if !showTap && !showScan {
		return
	}
	const (
		tapIcon, tapText   = "contactless", "Toca"
		scanIcon, scanText = "scan", "Escanea"
	)
	iconD := fontSize * 1.05
	innerGap := fontSize * 0.3
	segW := func(text string) float64 { return iconD + innerGap + estTextWidth(text, fontSize) }
	tapW, scanW := segW(tapText), segW(scanText)
	gap := fontSize * 1.0

	total := 0.0
	switch {
	case showTap && showScan:
		total = tapW + gap + scanW
	case showTap:
		total = tapW
	default:
		total = scanW
	}
	// Shrink both segments together so a pair that would collide (two QRs
	// side by side, say) fits its column instead of overflowing it.
	if maxWidth > 0 && total > maxWidth {
		scale := math.Max(maxWidth/total, 0.6)
		fontSize *= scale
		iconD, innerGap = fontSize*1.05, fontSize*0.3
		tapW, scanW = segW(tapText), segW(scanText)
		gap = fontSize * 1.0
		total *= scale
	}

	promptH := fontSize * 1.4
	y := rowCY - promptH/2
	startX := cx - total/2

	if showTap && showScan {
		s.add(promptElement(idPrefix+"-tap", "Toca", startX, y, tapW, promptH, tapIcon, tapText, fill))
		s.add(promptElement(idPrefix+"-scan", "Escanea", startX+tapW+gap, y, scanW, promptH, scanIcon, scanText, fill))
		return
	}
	if showTap {
		s.add(promptElement(idPrefix+"-tap", "Toca", cx-tapW/2, y, tapW, promptH, tapIcon, tapText, fill))
		return
	}
	s.add(promptElement(idPrefix+"-scan", "Escanea", cx-scanW/2, y, scanW, promptH, scanIcon, scanText, fill))
}

func promptElement(id, name string, x, y, w, h float64, icon, text, fill string) models.CardElement {
	return models.CardElement{
		ID: id, Type: models.ElementPrompt, Name: name,
		X: x, Y: y, W: w, H: h,
		Props: props(models.PromptProps{Icon: icon, Text: text, Color: fill, Bare: true}),
	}
}

// addDiscountBadge ports drawDiscountBadge into two elements — the dashed
// coupon border and the code sitting in it — plus an optional label above,
// returning nothing because nothing downstream of it needs its height once
// positions are frozen.
func (s *seedCtx) addDiscountBadge(cx, topY, availableW, h float64, onBg string) {
	code, label := s.content.DiscountCode, s.content.DiscountLabel
	y := topY
	if label != "" {
		labelFontSize := fitFontSize(label, h*0.034, availableW)
		y += labelFontSize
		s.addTextAt("coupon-label", "Etiqueta del descuento", cx, y, availableW,
			models.TextProps{Text: label, FontSize: labelFontSize, FontWeight: 700, Color: onBg, Align: "center"}, 1, 0.85)
		y += h * 0.02
	}
	codeFontSize := fitFontSize(code, h*0.05, availableW*0.75)
	badgeH := codeFontSize * 1.9
	badgeW := math.Min(availableW, estTextWidth(code, codeFontSize)+codeFontSize*2.4)
	badgeY := y + h*0.015

	el := models.CardElement{
		ID: "coupon-frame", Type: models.ElementShape, Name: "Marco del cupón",
		X: cx - badgeW/2, Y: badgeY, W: badgeW, H: badgeH,
		Opacity: fptr(0.7),
		Props: props(models.ShapeProps{
			Kind: "rect", Fill: "none", Stroke: onBg, StrokeWidth: codeFontSize * 0.06,
			DashArray: fmt.Sprintf("%.2f %.2f", codeFontSize*0.18, codeFontSize*0.12),
			Radius:    badgeH * 0.25,
		}),
	}
	s.add(el)

	s.addTextAt("coupon-code", "Código de descuento", cx, badgeY+badgeH*0.68, badgeW,
		models.TextProps{Text: code, FontSize: codeFontSize, FontWeight: 800, Color: onBg, Align: "center", Letter: 1}, 1, 1)
}

// ---------------------------------------------------------------------------
// "bloque de color" — the default design
// ---------------------------------------------------------------------------

// singleStack is the measured text column above the QR: everything from the
// badge down to the coupon, at one particular scale.
type singleStack struct {
	iconR, haloR                     float64
	lines                            []string
	fontSize, headlineBoxH           float64
	starR                            float64
	subFontSize, subBoxH             float64
	couponH                          float64
	gapIconText, gapAfter            float64
	showStars, showSub, showDiscount bool
	total                            float64
}

// measureSingleStack sizes the whole text column without drawing anything, so
// the caller can decide how to split the card between it and the QR before
// committing to either. scale shrinks every element uniformly — the lever
// pulled only when the QR cannot otherwise reach a scannable size.
func (s *seedCtx) measureSingleStack(w, h, availableW, scale float64) singleStack {
	st := singleStack{
		iconR:       math.Min(w, h) * 0.075 * scale,
		gapIconText: h * 0.04 * scale,
		gapAfter:    h * 0.032 * scale,
		starR:       h * 0.021 * scale,
	}
	st.haloR = st.iconR * qrHaloScale

	headline := s.content.Headline
	if headline == "" {
		headline = defaultHeadline(s.layout)
	}
	st.lines, st.fontSize = fitHeadlineLines(headline, h*0.058*scale, availableW)
	st.headlineBoxH = st.fontSize*1.2*float64(len(st.lines)-1) + st.fontSize*1.4

	st.showStars = s.layout == models.PrintCardGoogleReview
	st.showSub = s.content.Subheadline != ""
	if st.showSub {
		st.subFontSize = fitFontSize(s.content.Subheadline, h*0.038*scale, availableW)
		st.subBoxH = st.subFontSize * 1.4
	}
	st.showDiscount = s.layout == models.PrintCardThankYou && s.content.DiscountCode != ""
	if st.showDiscount {
		st.couponH = discountBadgeHeight(s.content.DiscountLabel, h) * scale
	}

	st.total = st.haloR*2 + st.gapIconText + st.headlineBoxH
	if st.showStars {
		st.total += st.gapAfter + st.starR*2
	}
	if st.showSub {
		st.total += st.gapAfter + st.subBoxH
	}
	if st.showDiscount {
		st.total += st.gapAfter + st.couponH
	}
	return st
}

// seedSingleQR lays out the classic stack: badge, headline, optional stars /
// subtitle / coupon, and the QR.
//
// The vertical budget is the point, and it is the opposite of what the
// pre-refactor renderer did. That one measured the text and handed the QR
// "whatever is left", which on a tall card produced a postage stamp under a
// big empty gap — the worst thing about the old cards. Here the two negotiate:
// the QR gets a generous preferred size, gives ground when the text genuinely
// needs it, and stops giving at a floor below which a printed code stops being
// reliably scannable. Only if even that floor does not fit does the text
// shrink instead.
func (s *seedCtx) seedSingleQR(offX, offY, w, h float64, colors cardColors) {
	cx := w / 2
	margin := h * 0.06
	availableW := w * 0.86
	gapBelowText := h * 0.035

	hintFontSize := h * 0.040
	hintH := hintFontSize * 2.4

	preferredFrame := math.Min(w*0.66, h*0.42) * qrSquareScale
	// The floor is a scannability limit, not an aesthetic one: below roughly
	// this share of the card a printed code starts failing to decode.
	//
	// It is driven by WIDTH, with height only as a backstop. Tying it to
	// height was backwards: on a tall card like a table tent — the format with
	// the most room to spare — a height-derived floor produced the smallest
	// codes of all.
	minFrame := math.Min(w*0.48, h*0.34) * qrSquareScale

	budget := h - margin*2 - hintH - gapBelowText
	st := s.measureSingleStack(w, h, availableW, 1)

	frameSide := budget - st.total
	switch {
	case frameSide >= preferredFrame:
		frameSide = preferredFrame
	case frameSide < minFrame:
		// Not even the floor fits: the words yield, not the code.
		frameSide = minFrame
		if allowed := budget - minFrame; allowed > 0 && st.total > allowed {
			st = s.measureSingleStack(w, h, availableW, math.Max(0.55, allowed/st.total))
		}
	}

	qrSize := frameSide / qrSquareScale
	frameTop := h - margin - hintH - frameSide
	qrTop := frameTop + (frameSide-qrSize)/2

	// Centre whatever room is left over between the top margin and the QR, so
	// spare space is shared rather than dumped into one gap.
	zoneTop := margin
	zoneBottom := frameTop - gapBelowText
	y := zoneTop + math.Max(0, (zoneBottom-zoneTop-st.total)/2)

	s.addTopBadge(offX+cx, offY+y+st.haloR, st.iconR)
	y += st.haloR*2 + st.gapIconText

	// addTextAt positions from a baseline; converting a box top into one is
	// the inverse of what renderTextElement does when it centres the lines.
	s.addTextAt("headline", "Título", offX+cx, offY+y+st.fontSize*1.04, availableW,
		models.TextProps{Text: joinLines(st.lines), FontSize: st.fontSize, FontWeight: 800, Color: colors.OnBackground, Align: "center", LineHeight: 1.2},
		len(st.lines), 1)
	y += st.headlineBoxH

	if st.showStars {
		y += st.gapAfter
		rowW := st.starR * 12.4
		s.add(models.CardElement{
			ID: "stars", Type: models.ElementIcon, Name: "Estrellas",
			X: offX + cx - rowW/2, Y: offY + y, W: rowW, H: st.starR * 2,
			Props: props(models.IconProps{Name: "star", Color: starGold, Count: 5, Gap: 1.3}),
		})
		y += st.starR * 2
	}
	if st.showSub {
		y += st.gapAfter
		s.addTextAt("subheadline", "Subtítulo", offX+cx, offY+y+st.subFontSize*1.04, availableW,
			models.TextProps{Text: s.content.Subheadline, FontSize: st.subFontSize, FontWeight: 400, Color: colors.OnBackground, Align: "center"}, 1, 0.85)
		y += st.subBoxH
	}
	if st.showDiscount {
		y += st.gapAfter
		s.addDiscountBadge(offX+cx, offY+y, availableW, h, colors.OnBackground)
	}

	s.addQR("qr", "Código QR", offX+cx, offY+qrTop, qrSize, 0)
	s.addPromptPair("qr", offX+cx, offY+promptRowCY(qrTop, qrSize, hintFontSize), hintFontSize, colors.OnBackground, true, true, 0)
}

// seedMultiQR lays out two independent QR columns.
//
// Each column is a self-contained stack — label, code, prompt — centred as a
// unit in the card's height. The pre-refactor version pinned the labels to the
// top and the codes to the bottom, which left a large empty band across the
// middle of every combined card; that gap was the first thing anyone noticed
// about them.
//
// In the tree, "two QRs" is not a kind of card at all: it is two QR elements.
func (s *seedCtx) seedMultiQR(offX, offY, w, h float64, ink string) {
	half := w / 2
	labelAvailableW := half * 0.88

	leftLabel := s.content.LeftLabel
	if leftLabel == "" {
		leftLabel = "Síguenos"
	}
	rightLabel := s.content.RightLabel
	if rightLabel == "" {
		rightLabel = "Déjanos tu reseña"
	}

	labelSize := math.Min(
		fitFontSize(leftLabel, h*0.05, labelAvailableW),
		fitFontSize(rightLabel, h*0.05, labelAvailableW),
	)
	labelBoxH := labelSize * 1.4

	hintFontSize := h * 0.032
	hintH := hintFontSize * 2.4
	gapLabelQR := h * 0.035

	qrSize := math.Min(half*0.80, h*0.44)
	frameSide := qrSize * qrSquareScale

	stackH := labelBoxH + gapLabelQR + frameSide + hintH
	top := math.Max(h*0.04, (h-stackH)/2)

	labelBaseline := top + labelSize*1.04
	frameTop := top + labelBoxH + gapLabelQR
	qrTop := frameTop + (frameSide-qrSize)/2

	// A full-height rule reads as "two separate offers"; the old hairline
	// stopping short of both edges read as a stray mark.
	const dividerW = 1.5
	s.add(models.CardElement{
		ID: "divider", Type: models.ElementShape, Name: "Divisor",
		X: offX + half - dividerW/2, Y: offY + h*0.06, W: dividerW, H: h * 0.88,
		Opacity: fptr(0.3),
		Props:   props(models.ShapeProps{Kind: "rect", Fill: ink}),
	})

	columns := []struct {
		idSuffix, name, label string
		cx                    float64
		targetIdx             int
	}{
		{"left", "izquierdo", leftLabel, half / 2, 0},
		{"right", "derecho", rightLabel, half + half/2, 1},
	}
	for _, col := range columns {
		s.addTextAt("label-"+col.idSuffix, "Etiqueta "+col.name, offX+col.cx, offY+labelBaseline, labelAvailableW,
			models.TextProps{Text: col.label, FontSize: labelSize, FontWeight: 800, Color: ink, Align: "center"}, 1, 1)
		s.addQR("qr-"+col.idSuffix, "QR "+col.name, offX+col.cx, offY+qrTop, qrSize, col.targetIdx)
		// Just "Escanea", no "Toca": the column's own label above the code
		// already says what it's for, and a caption wider than half the card
		// is exactly how the two columns used to collide.
		s.addPromptPair("qr-"+col.idSuffix, offX+col.cx, offY+promptRowCY(qrTop, qrSize, hintFontSize),
			hintFontSize, ink, false, true, half*0.92)
	}
}

// seedMultiQRStyled gives the two-QR layout the same visual treatments every
// other design gets, and returns the background to paint behind it.
//
// Without this the style picker was lying: choosing any style for a combined
// card produced the identical plain colour block, because multi_qr fell
// through to the default branch in every case.
func (s *seedCtx) seedMultiQRStyled(w, h, cornerR, holeClearance float64, style string, base models.CardBackground) models.CardBackground {
	inset := func(margin, topExtra float64) (x, y, iw, ih float64) {
		y = math.Max(margin, holeClearance) + topExtra
		return margin, y, w - 2*margin, h - y - margin
	}

	switch style {
	case "spotlight":
		x, y, iw, ih := inset(math.Min(w, h)*0.06, 0)
		s.seedMultiQR(x, y, iw, ih, "#ffffff")
		return models.CardBackground{Fill: spotlightBase}

	case "corners":
		s.addCornerAccents(w, h)
		x, y, iw, ih := inset(math.Min(w, h)*0.06, 0)
		s.seedMultiQR(x, y, iw, ih, "#ffffff")
		return models.CardBackground{Fill: cornersDarkBase}

	case "framed":
		margin := s.addFrameInner(w, h, cornerR)
		x, y, iw, ih := inset(margin, 0)
		s.seedMultiQR(x, y, iw, ih, framedInk)
		return models.CardBackground{Fill: base.Fill, GradientTo: base.GradientTo}

	case "banner":
		// A slim accent strip rather than the single-QR version's deep band:
		// with no badge or headline to carry, a tall band is just an empty
		// block of colour, and the two columns need the height more.
		bandH := math.Max(h*0.055, holeClearance)
		s.add(models.CardElement{
			ID: "banner-band", Type: models.ElementShape, Name: "Franja de color",
			X: 0, Y: 0, W: w, H: bandH,
			Props: props(models.ShapeProps{Kind: "rect", Fill: s.colors.Background}),
		})
		margin := math.Min(w, h) * 0.05
		s.seedMultiQR(margin, bandH+margin, w-2*margin, h-bandH-2*margin, splitInk)
		return models.CardBackground{Fill: bannerBase}

	case "diagonal":
		// Same slim-strip reasoning as "banner" above, just cut at an angle
		// instead of straight across.
		bandH := math.Max(h*0.055, holeClearance)
		slant := h * 0.04
		s.add(diagonalWedge(w, bandH, slant, s.colors.Background))
		margin := math.Min(w, h) * 0.05
		s.seedMultiQR(margin, bandH+slant+margin, w-2*margin, h-bandH-slant-2*margin, splitInk)
		return models.CardBackground{Fill: bannerBase}

	case "outline":
		panelMargin := s.seedOutline(w, h, cornerR, holeClearance)
		x, y, iw, ih := inset(panelMargin, 0)
		s.seedMultiQR(x, y, iw, ih, outlineInk)
		return models.CardBackground{Fill: outlineBase}

	case "pattern":
		patternKind := s.colors.Pattern
		if patternKind == "" {
			patternKind = "dots"
		}
		margin := math.Min(w, h) * 0.06
		panelY := math.Max(margin, holeClearance)
		s.add(models.CardElement{
			ID: "pattern-panel", Type: models.ElementShape, Name: "Panel de contenido",
			X: margin, Y: panelY, W: w - margin*2, H: h - panelY - margin,
			Props: props(models.ShapeProps{Kind: "rect", Fill: "#ffffff", Radius: math.Max(cornerR-margin*0.4, 0), Shadow: true}),
		})
		s.seedMultiQR(margin, panelY, w-margin*2, h-panelY-margin, "#111827")
		return models.CardBackground{Fill: base.Fill, GradientTo: base.GradientTo, Pattern: patternKind, PatternInk: s.colors.OnBackground}

	default:
		x, y, iw, ih := inset(math.Min(w, h)*0.06, 0)
		s.seedMultiQR(x, y, iw, ih, s.colors.OnBackground)
		return base
	}
}

// ---------------------------------------------------------------------------
// The three alternative styles
// ---------------------------------------------------------------------------

const (
	bannerBase      = "#ffffff"
	spotlightBase   = "#111318"
	cornersDarkBase = "#14141a"
	framedCream     = "#fffaf2"
	framedInk       = "#1f2430"
	splitInk        = "#111827"
	starGold        = "#FBBC05"
	// outlineBase/outlineInk are near-white and near-black rather than the
	// pure #ffffff/#111827 pair used elsewhere: "outline" is the one style
	// built entirely around restraint, and a card that's plain white behind
	// a thin line has less presence than one with the barest warmth to it.
	outlineBase = "#fdfdfb"
	outlineInk  = "#1f2430"
)

// seedBanner is the layout most real "review us" table tents actually use: a
// solid colour band across the top carrying the badge and the call to action,
// and a clean white lower half where the QR can be as large as it needs to
// be. Keeping the code on plain white — rather than floating it on the brand
// colour — is what lets it be printed big and still scan reliably.
func (s *seedCtx) seedBanner(w, h, holeClearance float64) {
	cx := w / 2
	bandH := math.Max(h*0.42, holeClearance+h*0.08)
	// The offer belongs with the message, in the colour band — not in the
	// white zone, where it would take the room that makes this style's big
	// scannable code worth having.
	showDiscount := s.layout == models.PrintCardThankYou && s.content.DiscountCode != ""
	couponH := 0.0
	if showDiscount {
		couponH = discountBadgeHeight(s.content.DiscountLabel, h)
		bandH += couponH + h*0.03
	}

	s.add(models.CardElement{
		ID: "banner-band", Type: models.ElementShape, Name: "Franja de color",
		X: 0, Y: 0, W: w, H: bandH,
		Props: props(models.ShapeProps{Kind: "rect", Fill: s.colors.Background}),
	})

	iconR := math.Min(w, h) * 0.075
	iconCY := bandH * 0.30
	if holeClearance > 0 {
		iconCY = math.Max(iconCY, holeClearance+iconR*qrHaloScale)
	}
	s.addTopBadge(cx, iconCY, iconR)

	headline := s.content.Headline
	if headline == "" {
		headline = defaultHeadline(s.layout)
	}
	availableW := w * 0.84
	lines, fontSize := fitHeadlineLines(headline, h*0.055, availableW)
	headlineY := iconCY + iconR*qrHaloScale + h*0.075
	s.addTextAt("headline", "Título", cx, headlineY, availableW,
		models.TextProps{Text: joinLines(lines), FontSize: fontSize, FontWeight: 800, Color: s.colors.OnBackground, Align: "center", LineHeight: 1.2},
		len(lines), 1)

	textBottom := headlineY + float64(len(lines)-1)*fontSize*1.2
	if s.layout == models.PrintCardGoogleReview {
		r := h * 0.019
		rowW := r * 12.4
		starY := textBottom + h*0.045
		s.add(models.CardElement{
			ID: "stars", Type: models.ElementIcon, Name: "Estrellas",
			X: cx - rowW/2, Y: starY - r, W: rowW, H: r * 2,
			Props: props(models.IconProps{Name: "star", Color: starGold, Count: 5, Gap: 1.3}),
		})
		textBottom = starY + r
	} else if s.content.Subheadline != "" {
		subFontSize := fitFontSize(s.content.Subheadline, h*0.034, availableW)
		s.addTextAt("subheadline", "Subtítulo", cx, textBottom+subFontSize*1.4, availableW,
			models.TextProps{Text: s.content.Subheadline, FontSize: subFontSize, FontWeight: 400, Color: s.colors.OnBackground, Align: "center"}, 1, 0.85)
		textBottom += subFontSize * 1.4
	}
	if showDiscount {
		s.addDiscountBadge(cx, textBottom+h*0.03, availableW, h, s.colors.OnBackground)
	}

	// The white zone below the band always uses dark ink: it is white
	// whatever the brand colour is, so the contrast colour computed against
	// that colour would be wrong here.
	hintFontSize := h * 0.036
	hintH := hintFontSize * 2.6
	zoneMargin := h * 0.05
	zoneTop := bandH + h*0.04

	available := math.Min(w*0.66, h-zoneTop-zoneMargin-hintH)
	qrSize := seedQRSizeIn(available)
	qrY := (h - hintH - zoneMargin) - qrSize/2*(1+qrSquareScale)
	s.addQR("qr", "Código QR", cx, qrY, qrSize, 0)
	s.addPromptPair("qr", cx, promptRowCY(qrY, qrSize, hintFontSize), hintFontSize, splitInk, true, true, 0)
}

// diagonalWedge is the shape "banner"'s straight band would be if it were
// cut at an angle instead: full colour at the left edge, down to bandH at
// the right — the wedge tallest where a business card is read first — with
// slant added on top of bandH so the deepest point still clears whatever
// bandH itself was sized to clear (badge, headline, a hole punch).
func diagonalWedge(w, bandH, slant float64, fill string) models.CardElement {
	return models.CardElement{
		ID: "diagonal-band", Type: models.ElementShape, Name: "Cuña de color",
		X: 0, Y: 0, W: w, H: bandH + slant,
		Props: props(models.ShapeProps{
			Kind: "polygon", Fill: fill,
			Points: fmt.Sprintf("0,0 %.2f,0 %.2f,%.2f 0,%.2f", w, w, bandH, bandH+slant),
		}),
	}
}

// seedDiagonal is "banner" with its straight band swapped for a diagonal
// wedge — otherwise identical, badge/headline/QR sit at the exact positions
// they would under seedBanner, since the wedge's shallow (right) edge is cut
// at the same bandH banner itself uses, so content never sits below where
// the colour actually reaches for anyone standing to the card's right.
func (s *seedCtx) seedDiagonal(w, h, holeClearance float64) {
	cx := w / 2
	bandH := math.Max(h*0.42, holeClearance+h*0.08)
	showDiscount := s.layout == models.PrintCardThankYou && s.content.DiscountCode != ""
	couponH := 0.0
	if showDiscount {
		couponH = discountBadgeHeight(s.content.DiscountLabel, h)
		bandH += couponH + h*0.03
	}
	slant := h * 0.07
	s.add(diagonalWedge(w, bandH, slant, s.colors.Background))

	iconR := math.Min(w, h) * 0.075
	iconCY := bandH * 0.30
	if holeClearance > 0 {
		iconCY = math.Max(iconCY, holeClearance+iconR*qrHaloScale)
	}
	s.addTopBadge(cx, iconCY, iconR)

	headline := s.content.Headline
	if headline == "" {
		headline = defaultHeadline(s.layout)
	}
	availableW := w * 0.84
	lines, fontSize := fitHeadlineLines(headline, h*0.055, availableW)
	headlineY := iconCY + iconR*qrHaloScale + h*0.075
	s.addTextAt("headline", "Título", cx, headlineY, availableW,
		models.TextProps{Text: joinLines(lines), FontSize: fontSize, FontWeight: 800, Color: s.colors.OnBackground, Align: "center", LineHeight: 1.2},
		len(lines), 1)

	textBottom := headlineY + float64(len(lines)-1)*fontSize*1.2
	if s.layout == models.PrintCardGoogleReview {
		r := h * 0.019
		rowW := r * 12.4
		starY := textBottom + h*0.045
		s.add(models.CardElement{
			ID: "stars", Type: models.ElementIcon, Name: "Estrellas",
			X: cx - rowW/2, Y: starY - r, W: rowW, H: r * 2,
			Props: props(models.IconProps{Name: "star", Color: starGold, Count: 5, Gap: 1.3}),
		})
		textBottom = starY + r
	} else if s.content.Subheadline != "" {
		subFontSize := fitFontSize(s.content.Subheadline, h*0.034, availableW)
		s.addTextAt("subheadline", "Subtítulo", cx, textBottom+subFontSize*1.4, availableW,
			models.TextProps{Text: s.content.Subheadline, FontSize: subFontSize, FontWeight: 400, Color: s.colors.OnBackground, Align: "center"}, 1, 0.85)
		textBottom += subFontSize * 1.4
	}
	if showDiscount {
		s.addDiscountBadge(cx, textBottom+h*0.03, availableW, h, s.colors.OnBackground)
	}

	hintFontSize := h * 0.036
	hintH := hintFontSize * 2.6
	zoneMargin := h * 0.05
	zoneTop := bandH + slant + h*0.04

	available := math.Min(w*0.66, h-zoneTop-zoneMargin-hintH)
	qrSize := seedQRSizeIn(available)
	qrY := (h - hintH - zoneMargin) - qrSize/2*(1+qrSquareScale)
	s.addQR("qr", "Código QR", cx, qrY, qrSize, 0)
	s.addPromptPair("qr", cx, promptRowCY(qrY, qrSize, hintFontSize), hintFontSize, splitInk, true, true, 0)
}

// seedSpotlight is the dark, high-contrast treatment: near-black card,
// oversized rating stars, and the QR on its own bright panel. It reads as the
// most "premium" of the styles and is the one that survives being printed
// small, because the code sits on the largest unbroken light area of any
// layout here.
func (s *seedCtx) seedSpotlight(w, h, holeClearance float64) {
	cx := w / 2
	margin := math.Max(h*0.075, holeClearance)

	topY := margin
	if s.layout == models.PrintCardGoogleReview {
		// Stars lead the card rather than sitting under the headline: on a
		// review card they say what it is faster than any wording.
		r := math.Min(w, h) * 0.042
		rowW := r * 12.4
		s.add(models.CardElement{
			ID: "stars", Type: models.ElementIcon, Name: "Estrellas",
			X: cx - rowW/2, Y: topY, W: rowW, H: r * 2,
			Props: props(models.IconProps{Name: "star", Color: starGold, Count: 5, Gap: 1.3}),
		})
		topY += r * 2
	} else {
		iconR := math.Min(w, h) * 0.07
		s.addTopBadge(cx, topY+iconR*qrHaloScale, iconR)
		topY += iconR * qrHaloScale * 2
	}

	headline := s.content.Headline
	if headline == "" {
		headline = defaultHeadline(s.layout)
	}
	availableW := w * 0.86
	lines, fontSize := fitHeadlineLines(headline, h*0.062, availableW)
	headlineY := topY + h*0.08
	s.addTextAt("headline", "Título", cx, headlineY, availableW,
		models.TextProps{Text: joinLines(lines), FontSize: fontSize, FontWeight: 800, Color: "#ffffff", Align: "center", LineHeight: 1.2},
		len(lines), 1)

	textBottom := headlineY + float64(len(lines)-1)*fontSize*1.2
	if s.content.Subheadline != "" {
		subFontSize := fitFontSize(s.content.Subheadline, h*0.036, availableW)
		s.addTextAt("subheadline", "Subtítulo", cx, textBottom+subFontSize*1.5, availableW,
			models.TextProps{Text: s.content.Subheadline, FontSize: subFontSize, FontWeight: 400, Color: "#ffffff", Align: "center"}, 1, 0.75)
		textBottom += subFontSize * 1.5
	}

	hintFontSize := h * 0.038
	hintH := hintFontSize * 2.6
	zoneMargin := h * 0.06
	zoneTop := textBottom + h*0.05

	if s.layout == models.PrintCardThankYou && s.content.DiscountCode != "" {
		s.addDiscountBadge(cx, zoneTop, availableW, h, "#ffffff")
		zoneTop += discountBadgeHeight(s.content.DiscountLabel, h) + h*0.03
	}
	available := math.Min(w*0.72, h-zoneTop-zoneMargin-hintH)
	qrSize := seedQRSizeIn(available)
	qrY := (h - hintH - zoneMargin) - qrSize/2*(1+qrSquareScale)
	// White text: plain white has all the contrast it needs on a near-black
	// card, so it stays bare rather than fighting a tinted capsule.
	s.addQR("qr", "Código QR", cx, qrY, qrSize, 0)
	s.addPromptPair("qr", cx, promptRowCY(qrY, qrSize, hintFontSize), hintFontSize, "#ffffff", true, true, 0)
}

// seedSplit is the port of renderSplitCard: a colored top zone over a white
// bottom zone divided by a soft wave. The wave becomes a path element, so a
// designer can drag the divide up or down — which the old fixed-then-clamped
// splitY computation never allowed.
func (s *seedCtx) seedSplit(w, h, holeClearance float64) {
	margin := math.Max(h*0.05, holeClearance)
	cx := w / 2
	iconR := math.Min(w, h) * 0.095
	iconCY := margin + iconR*1.55
	haloR := iconR * qrHaloScale

	headline := s.content.Headline
	if headline == "" {
		headline = defaultHeadline(s.layout)
	}
	availableW := w * 0.86
	lines, fontSize := fitHeadlineLines(headline, h*0.05, availableW)
	lineGap := fontSize * 1.2
	headlineY := iconCY + haloR + h*0.075
	textBottom := headlineY + float64(len(lines)-1)*lineGap

	showStars := s.layout == models.PrintCardGoogleReview
	showSub := !showStars && s.content.Subheadline != ""
	var subFontSize, starY, subY float64
	switch {
	case showStars:
		starY = textBottom + h*0.04
		textBottom = starY
	case showSub:
		subFontSize = fitFontSize(s.content.Subheadline, h*0.032, availableW)
		subY = textBottom + subFontSize*1.3
		textBottom = subY
	}
	discountTopY := textBottom + h*0.03
	showDiscount := s.layout == models.PrintCardThankYou && s.content.DiscountCode != ""
	if showDiscount {
		textBottom = discountTopY + discountBadgeHeight(s.content.DiscountLabel, h)
	}

	splitY := math.Max(h*0.5, math.Min(h*0.68, textBottom+h*0.06))
	bulge := h * 0.045

	// The wave's own box starts at the crest and runs to the bottom edge, so
	// its path is expressed in local coordinates and moves with it.
	waveTop := splitY - bulge
	waveH := h - waveTop
	s.add(models.CardElement{
		ID: "split-wave", Type: models.ElementShape, Name: "Zona blanca",
		X: 0, Y: waveTop, W: w, H: waveH,
		Props: props(models.ShapeProps{
			Kind: "path",
			Fill: "#ffffff",
			Path: fmt.Sprintf("M 0 %.2f Q %.2f 0 %.2f %.2f L %.2f %.2f L 0 %.2f Z",
				bulge, w/2, w, bulge, w, waveH, waveH),
		}),
	})

	s.addTopBadge(cx, iconCY, iconR)
	s.addTextAt("headline", "Título", cx, headlineY, availableW,
		models.TextProps{Text: joinLines(lines), FontSize: fontSize, FontWeight: 800, Color: s.colors.OnBackground, Align: "center", LineHeight: 1.2},
		len(lines), 1)

	switch {
	case showStars:
		r := h * 0.018
		rowW := r * 12.4
		s.add(models.CardElement{
			ID: "stars", Type: models.ElementIcon, Name: "Estrellas",
			X: cx - rowW/2, Y: starY - r, W: rowW, H: r * 2,
			Props: props(models.IconProps{Name: "star", Color: starGold, Count: 5, Gap: 1.3}),
		})
	case showSub:
		s.addTextAt("subheadline", "Subtítulo", cx, subY, availableW,
			models.TextProps{Text: s.content.Subheadline, FontSize: subFontSize, FontWeight: 400, Color: s.colors.OnBackground, Align: "center"}, 1, 0.8)
	}
	if showDiscount {
		s.addDiscountBadge(cx, discountTopY, availableW, h, s.colors.OnBackground)
	}

	// The bottom zone is white whatever the card's own color is, so its QR
	// and caption always used dark ink rather than the contrast color
	// computed against the brand background.
	hintFontSize := h * 0.036
	hintH := hintFontSize * 2.6
	zoneMargin := h * 0.045
	zoneTop := splitY + bulge*0.4
	available := math.Min(w*0.62, h-zoneTop-zoneMargin-hintH)
	qrSize := seedQRSize(available, w, h)
	qrY := (h - hintH - zoneMargin) - qrSize/2*(1+qrSquareScale)
	s.addQR("qr", "Código QR", cx, qrY, qrSize, 0)
	s.addPromptPair("qr", cx, promptRowCY(qrY, qrSize, hintFontSize), hintFontSize, splitInk, true, true, 0)
}

// addCornerAccents draws the four triangular accents of the "corners" style,
// derived from the business's own accent colour. Each triangle is its own
// polygon element rather than baked-in artwork, so they can be recoloured or
// removed individually — and, being separate from the content, the two-QR
// layout can wear the same chrome.
func (s *seedCtx) addCornerAccents(w, h float64) {
	// Equal-sided arms: scaling each by its own axis made the four triangles
	// meet across a landscape card and turned it into a hexagon rather than a
	// rectangle with corner accents.
	arm := math.Min(w*0.28, h*0.22)
	armW, armH := arm, arm

	corners := []struct {
		id, name string
		x, y     float64
		points   string
		color    string
	}{
		{"corner-tl", "Esquina superior izquierda", 0, 0,
			fmt.Sprintf("0,0 %.2f,0 0,%.2f", armW, armH), s.colors.Accent},
		{"corner-tr", "Esquina superior derecha", w - armW, 0,
			fmt.Sprintf("%.2f,0 0,0 %.2f,%.2f", armW, armW, armH), lerpHex(s.colors.Accent, "#ffffff", 0.32)},
		{"corner-bl", "Esquina inferior izquierda", 0, h - armH,
			fmt.Sprintf("0,%.2f %.2f,%.2f 0,0", armH, armW, armH), lerpHex(s.colors.Accent, "#000000", 0.28)},
		{"corner-br", "Esquina inferior derecha", w - armW, h - armH,
			fmt.Sprintf("%.2f,%.2f 0,%.2f %.2f,0", armW, armH, armH, armW), lerpHex(s.colors.Accent, "#ffffff", 0.14)},
	}
	for _, c := range corners {
		s.add(models.CardElement{
			ID: c.id, Type: models.ElementShape, Name: c.name,
			X: c.x, Y: c.y, W: armW, H: armH,
			Props: props(models.ShapeProps{Kind: "polygon", Fill: c.color, Points: c.points}),
		})
	}

}

func (s *seedCtx) seedCorners(w, h, holeClearance float64) {
	s.addCornerAccents(w, h)

	border := math.Min(w, h) * 0.06
	panelY := math.Max(border, holeClearance)
	panelW, panelH := w-2*border, h-panelY-border
	// Content always drew in white here, since the base is always dark —
	// never the contrast color computed against the brand background.
	cornersColors := s.colors
	cornersColors.OnBackground = "#ffffff"
	s.seedSingleQR(border, panelY, panelW, panelH, cornersColors)
}

// seedFramed is the port of renderFramedCard: a bold color border ring
// around a cream interior, with content in dark ink.
// addFrameInner draws the cream interior panel of the "framed" style,
// returning the margin content has to keep clear of the border.
func (s *seedCtx) addFrameInner(w, h, cornerR float64) float64 {
	borderW := math.Min(w, h) * 0.055
	innerR := math.Max(cornerR-borderW*0.6, 0)
	s.add(models.CardElement{
		ID: "frame-inner", Type: models.ElementShape, Name: "Interior",
		X: borderW, Y: borderW, W: w - borderW*2, H: h - borderW*2,
		Props: props(models.ShapeProps{Kind: "rect", Fill: framedCream, Radius: innerR}),
	})
	return math.Min(w, h)*0.055 + math.Min(w, h)*0.045
}

func (s *seedCtx) seedFramed(w, h, cornerR, holeClearance float64) {
	margin := s.addFrameInner(w, h, cornerR)
	panelY := math.Max(margin, holeClearance)
	panelW, panelH := w-2*margin, h-panelY-margin
	framedColors := s.colors
	framedColors.OnBackground = framedInk
	s.seedSingleQR(margin, panelY, panelW, panelH, framedColors)
}

// seedOutline draws "outline"'s entire chrome: a single hairline rounded
// rectangle, nothing else. Where "framed" reads as a bold colour block with
// a light interior, this is the opposite instinct — the brand colour shows
// up only as a thin line, and everything else is negative space. Returns the
// margin content has to keep clear of the line.
func (s *seedCtx) seedOutline(w, h, cornerR, holeClearance float64) float64 {
	margin := math.Min(w, h) * 0.06
	borderW := math.Min(w, h) * 0.012
	s.add(models.CardElement{
		ID: "outline-border", Type: models.ElementShape, Name: "Borde",
		X: margin, Y: margin, W: w - margin*2, H: h - margin*2,
		Props: props(models.ShapeProps{
			Kind: "rect", Fill: "none", Stroke: s.colors.Accent, StrokeWidth: borderW,
			Radius: math.Max(cornerR-margin*0.5, 0),
		}),
	})
	return margin + borderW*3
}

// seedPatternPanel is "corners" turned inside out: instead of a dark card
// with a plain white content panel, the card itself carries the business's
// pattern texture (the same dots/lines/grid/waves/circles the designer's own
// "Fondo" tab offers) at full colour, and a plain white panel sits on top to
// keep the badge/headline/QR legible over it. The pattern *value* on the
// tree is set by the caller — this only draws the panel and the content
// inside it.
func (s *seedCtx) seedPatternPanel(w, h, cornerR, holeClearance float64) {
	margin := math.Min(w, h) * 0.06
	panelY := math.Max(margin, holeClearance)
	panelW, panelH := w-2*margin, h-panelY-margin
	s.add(models.CardElement{
		ID: "pattern-panel", Type: models.ElementShape, Name: "Panel de contenido",
		X: margin, Y: panelY, W: panelW, H: panelH,
		Props: props(models.ShapeProps{Kind: "rect", Fill: "#ffffff", Radius: math.Max(cornerR-margin*0.4, 0), Shadow: true}),
	})
	patternColors := s.colors
	patternColors.OnBackground = "#111827"
	s.seedSingleQR(margin, panelY, panelW, panelH, patternColors)
}

// joinLines turns the wrapper's two fitted lines into the single
// newline-separated string a text element stores. The wrapping decision is
// made once, at seed time; from then on the designer owns the line breaks.
func joinLines(lines []string) string {
	out := ""
	for i, l := range lines {
		if i > 0 {
			out += "\n"
		}
		out += l
	}
	return out
}
