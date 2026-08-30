package services

import (
	"fmt"
	"math"
	"strings"
)

// Generic, monochrome glyphs for print-card layouts — used for anything
// that ISN'T naming a specific external platform (menu, gift/loyalty,
// location pin). Each returns a self-contained SVG fragment centered at
// (cx, cy) with the given radius.
//
// googleGIcon and platformIcon (below) are the deliberate exceptions,
// confirmed with the user: a "leave us a review" or "follow us" card needs
// to read as "this goes to Google / Instagram / ..." at a glance, the same
// way virtually every real print/table-tent card on the market uses that
// platform's own mark and brand color rather than a generic icon — it's
// naming where the QR leads, not implying that platform's endorsement of
// the business.

func starIcon(cx, cy, r float64, color string) string {
	const points = 5
	innerR := r * 0.42
	var pts []string
	for i := 0; i < points*2; i++ {
		radius := r
		if i%2 == 1 {
			radius = innerR
		}
		angle := -math.Pi/2 + float64(i)*math.Pi/points
		x := cx + radius*math.Cos(angle)
		y := cy + radius*math.Sin(angle)
		pts = append(pts, fmt.Sprintf("%.2f,%.2f", x, y))
	}
	return fmt.Sprintf(`<polygon points="%s" fill="%s"/>`, strings.Join(pts, " "), color)
}

// googleGIcon draws Google's four-color "G" mark, scaled and centered at
// (cx, cy) with the given radius. Path data is Google's standard 48x48 "G"
// logomark (the same one used across countless "Sign in with Google"
// buttons), reused here — see the package doc comment above for why this
// specific icon is the one deliberate exception to the "no real brand
// logos" rule.
func googleGIcon(cx, cy, r float64) string {
	const (
		srcCx, srcCy = 23.5, 24.0
		srcR         = 22.0
	)
	scale := r / srcR
	tx := cx - srcCx*scale
	ty := cy - srcCy*scale
	return fmt.Sprintf(`<g transform="translate(%.2f %.2f) scale(%.4f)">`+
		`<path fill="#4285F4" d="M45.12 24.5c0-1.56-.14-3.06-.4-4.5H24v8.51h11.84c-.51 2.75-2.06 5.08-4.39 6.64v5.52h7.11c4.16-3.83 6.56-9.47 6.56-16.17z"/>`+
		`<path fill="#34A853" d="M24 46c5.94 0 10.92-1.97 14.56-5.33l-7.11-5.52c-1.97 1.32-4.49 2.1-7.45 2.1-5.73 0-10.58-3.87-12.31-9.07H4.34v5.7C7.96 41.07 15.4 46 24 46z"/>`+
		`<path fill="#FBBC05" d="M11.69 28.18C11.25 26.86 11 25.45 11 24s.25-2.86.69-4.18v-5.7H4.34C2.85 17.09 2 20.45 2 24s.85 6.91 2.34 9.88l7.35-5.7z"/>`+
		`<path fill="#EA4335" d="M24 10.75c3.23 0 6.13 1.11 8.41 3.29l6.31-6.31C34.91 4.18 29.93 2 24 2 15.4 2 7.96 6.93 4.34 14.12l7.35 5.7c1.73-5.2 6.58-9.07 12.31-9.07z"/>`+
		`</g>`,
		tx, ty, scale)
}

// platformIcon dispatches to the right social platform glyph — each drawn
// in that platform's own real brand color(s), the "follow us" counterpart
// to googleGIcon (see the package doc comment for why).
func platformIcon(cx, cy, r float64, platform string) string {
	switch platform {
	case "facebook":
		return facebookIcon(cx, cy, r)
	case "tiktok":
		return tiktokIcon(cx, cy, r)
	case "youtube":
		return youtubeIcon(cx, cy, r)
	case "whatsapp":
		return whatsappIcon(cx, cy, r)
	default:
		return instagramIcon(cx, cy, r) // instagram, and any unrecognized value
	}
}

// instagramIcon: the familiar rounded-square camera outline + lens + flash
// dot, in Instagram's own purple-to-orange gradient.
func instagramIcon(cx, cy, r float64) string {
	var b strings.Builder
	gradID := fmt.Sprintf("igGrad%d", int(cx*100)+int(cy*100)+int(r*10))
	fmt.Fprintf(&b, `<linearGradient id="%s" x1="0%%" y1="100%%" x2="100%%" y2="0%%"><stop offset="0%%" stop-color="#833AB4"/><stop offset="50%%" stop-color="#E1306C"/><stop offset="100%%" stop-color="#F77737"/></linearGradient>`, gradID)
	half := r * 0.72
	fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.2f" fill="none" stroke="url(#%s)" stroke-width="%.2f"/>`,
		cx-half, cy-half, half*2, half*2, half*0.42, gradID, r*0.13)
	fmt.Fprintf(&b, `<circle cx="%.2f" cy="%.2f" r="%.2f" fill="none" stroke="url(#%s)" stroke-width="%.2f"/>`,
		cx, cy, r*0.32, gradID, r*0.13)
	fmt.Fprintf(&b, `<circle cx="%.2f" cy="%.2f" r="%.2f" fill="url(#%s)"/>`,
		cx+half*0.6, cy-half*0.6, r*0.07, gradID)
	return b.String()
}

// facebookIcon: the lowercase "f" mark on Facebook's own blue circle.
func facebookIcon(cx, cy, r float64) string {
	return fmt.Sprintf(
		`<circle cx="%.2f" cy="%.2f" r="%.2f" fill="#1877F2"/><text x="%.2f" y="%.2f" font-size="%.2f" font-weight="700" fill="#ffffff" text-anchor="middle" dominant-baseline="central" font-family="Georgia, serif">f</text>`,
		cx, cy, r*0.85, cx, cy+r*0.05, r*1.15,
	)
}

// whatsappIcon: a chat-bubble glyph on WhatsApp's own green circle — not
// the exact handset-in-bubble mark, but the same silhouette family and
// unmistakably "this is a chat app" alongside the real brand color.
func whatsappIcon(cx, cy, r float64) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<circle cx="%.2f" cy="%.2f" r="%.2f" fill="#25D366"/>`, cx, cy, r*0.85)
	bw, bh := r*0.85, r*0.6
	bx, by := cx-bw/2, cy-bh/2-r*0.08
	fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.2f" fill="#ffffff"/>`, bx, by, bw, bh, bh*0.35)
	fmt.Fprintf(&b, `<polygon points="%.2f,%.2f %.2f,%.2f %.2f,%.2f" fill="#ffffff"/>`,
		cx-bw*0.18, by+bh-1, cx-bw*0.18, by+bh+r*0.22, cx+bw*0.02, by+bh-1)
	return b.String()
}

// youtubeIcon: the red rounded-rect "play" mark.
func youtubeIcon(cx, cy, r float64) string {
	w, h := r*1.7, r*1.2
	return fmt.Sprintf(
		`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.2f" fill="#FF0000"/><polygon points="%.2f,%.2f %.2f,%.2f %.2f,%.2f" fill="#ffffff"/>`,
		cx-w/2, cy-h/2, w, h, h*0.28,
		cx-w*0.12, cy-h*0.28, cx-w*0.12, cy+h*0.28, cx+w*0.22, cy,
	)
}

// tiktokIcon: a simplified musical-note glyph layered in TikTok's own
// cyan/magenta/black "glitch" offset style, instead of one flat color.
func tiktokIcon(cx, cy, r float64) string {
	note := func(dx, dy float64, color string) string {
		stemX, headY := cx+dx+r*0.1, cy+dy+r*0.28
		return fmt.Sprintf(
			`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.2f" fill="%s"/><circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s"/>`,
			stemX-r*0.09, cy+dy-r*0.55, r*0.18, r*0.85, r*0.09, color,
			stemX-r*0.24, headY, r*0.22, color,
		)
	}
	var b strings.Builder
	fmt.Fprint(&b, note(r*0.12, r*0.07, "#25F4EE"))
	fmt.Fprint(&b, note(-r*0.12, -r*0.07, "#FE2C55"))
	fmt.Fprint(&b, note(0, 0, "#000000"))
	return b.String()
}

// menuGlyph is a simple 3-bar "list" icon — universally read as "menu".
func menuGlyph(cx, cy, r float64, color string) string {
	w := r * 1.3
	gap := r * 0.55
	var b strings.Builder
	for i := -1; i <= 1; i++ {
		y := cy + float64(i)*gap
		fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.2f" fill="%s"/>`,
			cx-w/2, y-r*0.09, w, r*0.18, r*0.09, color)
	}
	return b.String()
}

// giftGlyph is a simple gift-box shape — used for reward/loyalty layouts.
func giftGlyph(cx, cy, r float64, color string) string {
	boxW, boxH := r*1.5, r*1.1
	x, y := cx-boxW/2, cy-boxH/2+r*0.15
	var b strings.Builder
	fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.2f" fill="%s"/>`, x, y, boxW, boxH, r*0.08, color)
	fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s" opacity="0.55"/>`, cx-r*0.09, y, r*0.18, boxH, color)
	fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s" opacity="0.55"/>`, x, cy-r*0.09, boxW, r*0.18, color)
	// small bow, two loops
	fmt.Fprintf(&b, `<circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s"/>`, cx-r*0.18, y-r*0.05, r*0.16, color)
	fmt.Fprintf(&b, `<circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s"/>`, cx+r*0.18, y-r*0.05, r*0.16, color)
	return b.String()
}

// heartGlyph is a simple filled heart — used for the "thank you" layout.
func heartGlyph(cx, cy, r float64, color string) string {
	return fmt.Sprintf(
		`<path d="M %.2f %.2f C %.2f %.2f %.2f %.2f %.2f %.2f C %.2f %.2f %.2f %.2f %.2f %.2f C %.2f %.2f %.2f %.2f %.2f %.2f C %.2f %.2f %.2f %.2f %.2f %.2f Z" fill="%s"/>`,
		cx, cy+r*0.82,
		cx-r*1.05, cy-r*0.05, cx-r*0.95, cy-r*0.95, cx-r*0.35, cy-r*0.95,
		cx-r*0.12, cy-r*0.95, cx, cy-r*0.6, cx, cy-r*0.5,
		cx, cy-r*0.6, cx+r*0.12, cy-r*0.95, cx+r*0.35, cy-r*0.95,
		cx+r*0.95, cy-r*0.95, cx+r*1.05, cy-r*0.05, cx, cy+r*0.82,
		color,
	)
}

// mapGlyph is a simple pin shape — used for map/location-flavored layouts.
func pinGlyph(cx, cy, r float64, color string) string {
	return fmt.Sprintf(
		`<path d="M %.2f %.2f C %.2f %.2f %.2f %.2f %.2f %.2f C %.2f %.2f %.2f %.2f %.2f %.2f Z" fill="%s"/><circle cx="%.2f" cy="%.2f" r="%.2f" fill="white"/>`,
		cx, cy+r,
		cx-r*0.85, cy+r*0.15, cx-r*0.85, cy-r*0.65, cx, cy-r,
		cx+r*0.85, cy-r*0.65, cx+r*0.85, cy+r*0.15, cx, cy+r,
		color, cx, cy-r*0.25, r*0.32,
	)
}

// contactlessIcon draws a small hand-with-radiating-arcs glyph next to
// "Toca o escanea" — the same visual shorthand phones/terminals use for
// "tap here", reinforcing that this also works via an NFC tap, not only a
// camera scan.
// contactlessIcon is the standard "tap to pay" glyph: three arcs radiating
// from a single fixed origin point, all sharing that same center and the
// same angular sweep — true concentric rings, the way the real contactless
// symbol on every payment card and terminal is built. An earlier version
// computed each arc's start/end points as a fraction of ITS OWN radius
// instead of sharing one origin, so the three rings didn't actually share a
// center — at print size that read as a messy, slightly-off blob rather
// than clean radio waves, the recurring complaint across several rounds of
// feedback. Fixed opacity (no fade) so it stays visually as solid as
// scanTargetIcon right next to it — see that fix's own note.
func contactlessIcon(cx, cy, r float64, color string) string {
	var b strings.Builder
	ox, oy := cx, cy+r*0.5
	const startAngle = -135.0 * math.Pi / 180
	const endAngle = -45.0 * math.Pi / 180
	for _, radius := range []float64{r * 0.42, r * 0.72, r * 1.02} {
		x1, y1 := ox+radius*math.Cos(startAngle), oy+radius*math.Sin(startAngle)
		x2, y2 := ox+radius*math.Cos(endAngle), oy+radius*math.Sin(endAngle)
		fmt.Fprintf(&b, `<path d="M %.2f %.2f A %.2f %.2f 0 0 1 %.2f %.2f" stroke="%s" stroke-width="%.2f" stroke-linecap="round" fill="none"/>`,
			x1, y1, radius, radius, x2, y2, color, r*0.13)
	}
	fmt.Fprintf(&b, `<circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s"/>`, ox, oy, r*0.15, color)
	return b.String()
}

// scanTargetIcon is a small camera-viewfinder glyph (corner brackets around
// an empty square) — pairs with contactlessIcon in the dual "Toca / Escanea"
// caption so the two methods each get their own unmistakable pictogram
// instead of sharing one ambiguous icon.
func scanTargetIcon(cx, cy, r float64, color string) string {
	var b strings.Builder
	x, y, size := cx-r, cy-r, r*2
	arm := size * 0.32
	sw := r * 0.22
	corners := [][2]float64{{x, y}, {x + size, y}, {x, y + size}, {x + size, y + size}}
	dirs := [][2]float64{{1, 1}, {-1, 1}, {1, -1}, {-1, -1}}
	for i, c := range corners {
		dx, dy := dirs[i][0], dirs[i][1]
		fmt.Fprintf(&b, `<path d="M %.2f %.2f L %.2f %.2f M %.2f %.2f L %.2f %.2f" stroke="%s" stroke-width="%.2f" stroke-linecap="round" fill="none"/>`,
			c[0], c[1]+dy*arm, c[0], c[1],
			c[0], c[1], c[0]+dx*arm, c[1],
			color, sw)
	}
	return b.String()
}

// --- alternate "Toca" glyphs ------------------------------------------------
//
// contactlessIcon (the concentric-arcs symbol above) reads instantly to
// anyone who has tapped a payment terminal, but it says nothing about WHERE
// to tap — a business whose card has no visible NFC tag benefits from a
// glyph that shows a finger or a phone actually making contact. These two
// give the caption editor real alternatives instead of one fixed picture.

// tapArcs draws two concentric quarter-arcs radiating from (ox,oy) toward the
// upper-left — the "waves" half shared by every tap glyph on this page, kept
// as one function so the wave geometry (and the bug history behind sharing
// one true origin — see contactlessIcon's own note) only has to be right once.
func tapArcs(ox, oy, r float64, startDeg, endDeg float64, color string) string {
	var b strings.Builder
	start, end := startDeg*math.Pi/180, endDeg*math.Pi/180
	for _, radius := range []float64{r * 0.4, r * 0.7} {
		x1, y1 := ox+radius*math.Cos(start), oy+radius*math.Sin(start)
		x2, y2 := ox+radius*math.Cos(end), oy+radius*math.Sin(end)
		fmt.Fprintf(&b, `<path d="M %.2f %.2f A %.2f %.2f 0 0 1 %.2f %.2f" stroke="%s" stroke-width="%.2f" stroke-linecap="round" fill="none"/>`,
			x1, y1, radius, radius, x2, y2, color, r*0.11)
	}
	return b.String()
}

// tapHandIcon draws a finger reaching down to touch a surface, with tap-waves
// coming off the fingertip — spells out "tap here with your phone or finger"
// more literally than the abstract concentric-rings symbol does.
func tapHandIcon(cx, cy, r float64, color string) string {
	var b strings.Builder
	fingerW := r * 0.34
	tipY := cy - r*0.62
	baseY := cy + r*0.5
	fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.2f" fill="%s"/>`,
		cx-fingerW/2, tipY, fingerW, baseY-tipY, fingerW/2, color)
	fmt.Fprintf(&b, `<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="%s" stroke-width="%.2f" stroke-linecap="round"/>`,
		cx-r*0.55, cy+r*0.75, cx+r*0.55, cy+r*0.75, color, r*0.13)
	fmt.Fprint(&b, tapArcs(cx, tipY, r, -160, -20, color))
	return b.String()
}

// tapCardIcon draws a rounded card/phone shape with tap-waves off one corner
// — the shorthand used on real payment terminals for "tap your card or
// phone here", which reads unambiguously even without any wording.
func tapCardIcon(cx, cy, r float64, color string) string {
	var b strings.Builder
	w, h := r*1.1, r*0.62
	fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.2f" fill="none" stroke="%s" stroke-width="%.2f"/>`,
		cx-w/2, cy-h/2+r*0.08, w, h, r*0.12, color, r*0.1)
	fmt.Fprint(&b, tapArcs(cx+r*0.18, cy-r*0.55, r, -155, -25, color))
	return b.String()
}

// --- alternate "Escanea" glyphs ---------------------------------------------
//
// scanTargetIcon (the viewfinder brackets above) is the generic "point a
// camera here" symbol; these two are more specific about WHAT is being
// scanned, for a business that wants that spelled out.

// scanQRIcon draws a miniature QR code — the three finder-pattern corners a
// real QR always has (and never a fourth, in the corner nearest the data),
// plus a small cluster of data modules — instantly readable as "this is a QR"
// rather than a generic scan-target bracket.
func scanQRIcon(cx, cy, r float64, color string) string {
	var b strings.Builder
	eye := func(x, y, size float64) {
		fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.2f" fill="none" stroke="%s" stroke-width="%.2f"/>`,
			x, y, size, size, size*0.18, color, size*0.22)
		inner := size * 0.42
		fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s"/>`,
			x+(size-inner)/2, y+(size-inner)/2, inner, inner, color)
	}
	eyeSize := r * 0.62
	eye(cx-r*0.85, cy-r*0.85, eyeSize)
	eye(cx+r*0.85-eyeSize, cy-r*0.85, eyeSize)
	eye(cx-r*0.85, cy+r*0.85-eyeSize, eyeSize)

	// A few data-module dots in the one corner real QR codes leave bare — the
	// detail that makes this read as an actual code rather than a diagram.
	dot := r * 0.22
	for _, off := range [][2]float64{{0.05, 0.05}, {0.38, 0.05}, {0.05, 0.38}, {0.38, 0.38}} {
		fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s"/>`,
			cx+r*off[0], cy+r*off[1], dot, dot, color)
	}
	return b.String()
}

// scanCameraIcon draws a simple camera body with a lens — "point your camera
// here" stated as literally as scanQRIcon states "this is a QR", for whoever
// wants the glyph to describe the ACTION rather than the CODE.
func scanCameraIcon(cx, cy, r float64, color string) string {
	var b strings.Builder
	w, h := r*1.5, r*0.95
	fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.2f" fill="none" stroke="%s" stroke-width="%.2f"/>`,
		cx-w/2, cy-h/2+r*0.12, w, h, r*0.16, color, r*0.12)
	fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.2f" fill="%s"/>`,
		cx-r*0.22, cy-h/2-r*0.1, r*0.44, r*0.22, r*0.06, color)
	fmt.Fprintf(&b, `<circle cx="%.2f" cy="%.2f" r="%.2f" fill="none" stroke="%s" stroke-width="%.2f"/>`,
		cx, cy+r*0.12, r*0.32, color, r*0.12)
	fmt.Fprintf(&b, `<circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s"/>`, cx, cy+r*0.12, r*0.08, color)
	return b.String()
}

// dotPatternDef/linePatternDef write a reusable, tileable SVG <pattern>
// definition (dot grid / diagonal lines) that RenderCardSVG can fill the
// outer frame with — the "patrones, líneas" texture the brand border wears
// instead of a flat, plain color.
func dotPatternDef(id string, spacing float64, color string, opacity float64) string {
	return fmt.Sprintf(
		`<pattern id="%s" width="%.2f" height="%.2f" patternUnits="userSpaceOnUse"><circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s" opacity="%.2f"/></pattern>`,
		id, spacing, spacing, spacing/2, spacing/2, spacing*0.12, color, opacity,
	)
}

func linePatternDef(id string, spacing float64, color string, opacity float64) string {
	return fmt.Sprintf(
		`<pattern id="%s" width="%.2f" height="%.2f" patternUnits="userSpaceOnUse" patternTransform="rotate(45)"><line x1="0" y1="0" x2="0" y2="%.2f" stroke="%s" stroke-width="%.2f" opacity="%.2f"/></pattern>`,
		id, spacing, spacing, spacing, color, spacing*0.14, opacity,
	)
}

// gridPatternDef draws a graph-paper crosshatch — each tile is just the
// top and left edge of a cell, which is all a repeating grid needs.
func gridPatternDef(id string, spacing float64, color string, opacity float64) string {
	return fmt.Sprintf(
		`<pattern id="%s" width="%.2f" height="%.2f" patternUnits="userSpaceOnUse"><path d="M %.2f 0 L 0 0 0 %.2f" stroke="%s" stroke-width="%.2f" opacity="%.2f" fill="none"/></pattern>`,
		id, spacing, spacing, spacing, spacing, color, spacing*0.06, opacity,
	)
}

// wavesPatternDef draws a repeating horizontal wavy line — two arcs
// (opposite sweep flags) bulging up then down, tiling seamlessly because
// the path starts and ends at the same height.
func wavesPatternDef(id string, spacing float64, color string, opacity float64) string {
	r := spacing / 2
	path := fmt.Sprintf("M 0 %.2f A %.2f %.2f 0 0 1 %.2f %.2f A %.2f %.2f 0 0 0 %.2f %.2f", r, r, r, spacing, r, r, r, spacing*2, r)
	return fmt.Sprintf(
		`<pattern id="%s" width="%.2f" height="%.2f" patternUnits="userSpaceOnUse"><path d="%s" stroke="%s" stroke-width="%.2f" opacity="%.2f" fill="none"/></pattern>`,
		id, spacing*2, spacing, path, color, spacing*0.1, opacity,
	)
}

// circlesPatternDef tiles a grid of small ring outlines — bubbles rather
// than filled dots, a lighter/airier texture than dotPatternDef.
func circlesPatternDef(id string, spacing float64, color string, opacity float64) string {
	return fmt.Sprintf(
		`<pattern id="%s" width="%.2f" height="%.2f" patternUnits="userSpaceOnUse"><circle cx="%.2f" cy="%.2f" r="%.2f" stroke="%s" stroke-width="%.2f" opacity="%.2f" fill="none"/></pattern>`,
		id, spacing, spacing, spacing/2, spacing/2, spacing*0.35, color, spacing*0.06, opacity,
	)
}
