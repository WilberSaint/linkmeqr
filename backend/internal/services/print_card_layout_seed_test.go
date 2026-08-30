package services

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"testing"

	"linkmeqr/backend/internal/models"
)

// The seeder is a port of the old imperative renderers, and most of it is a
// literal copy of their arithmetic. The risky part is the handful of places
// where a value the old code emitted directly (a text baseline, a QR halo
// rect, a star's center) had to be inverted into an element BOX that the new
// renderer then converts back. Those round-trips are what these tests pin
// down: seed a tree, render it, and check the numbers that come out the far
// end are the ones the old renderer would have written.

func testCard(layoutKey models.PrintCardLayout, size models.PrintCardSizePreset, content, overrides string) *models.PrintCard {
	c := &models.PrintCard{
		LayoutKey:    layoutKey,
		SizePreset:   size,
		QRTargetType: models.QRTargetProfile,
		Content:      content,
	}
	if overrides != "" {
		c.ColorOverrides = &overrides
	}
	return c
}

// renderSeeded seeds and renders a card with a stub QR, returning the SVG.
// The QR stub is a recognizable fragment so its placement can be asserted
// without pulling a real encoder into these tests.
func renderSeeded(t *testing.T, card *models.PrintCard) (*models.CardLayout, string) {
	t.Helper()
	layout := SeedCardLayout(card, nil, false, "")
	if err := layout.Validate(); err != nil {
		t.Fatalf("seeded layout is invalid: %v", err)
	}
	assets := &LayoutAssets{QRSVGs: map[string]string{}}
	for _, el := range layout.Elements {
		if el.Type == models.ElementQR {
			assets.QRSVGs[el.ID] = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect id="stub"/></svg>`
		}
	}
	return layout, RenderLayoutSVG(layout, assets)
}

var (
	textYRe   = regexp.MustCompile(`<text x="([-\d.]+)" y="([-\d.]+)" font-family="[^"]*" font-size="([-\d.]+)"`)
	qrSVGRe   = regexp.MustCompile(`<svg x="([-\d.]+)" y="([-\d.]+)" width="([-\d.]+)"`)
	frameRe   = regexp.MustCompile(`<rect x="([-\d.]+)" y="([-\d.]+)" width="([-\d.]+)" height="([-\d.]+)" rx="([-\d.]+)" fill="#ffffff" filter`)
	starCXRe  = regexp.MustCompile(`<polygon points="([-\d.]+),`)
	closeEnou = 0.02 // half a rendered decimal place, in card units (1/200 inch)
)

// Every element renders inside its own <g transform="translate(x,y)">, so
// the coordinates in the SVG are local to that element. The old renderer
// wrote absolute ones, so each assertion below adds the element's own origin
// back before comparing.
func origin(el models.CardElement) (float64, float64) { return el.X, el.Y }

func mustFloat(t *testing.T, s string) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return v
}

func assertClose(t *testing.T, got, want float64, what string) {
	t.Helper()
	if math.Abs(got-want) > closeEnou {
		t.Errorf("%s: got %.4f, want %.4f", what, got, want)
	}
}

// TestTextBaselineRoundTrip is the important one. addTextAt stores a BOX
// derived from a baseline; renderTextElement recomputes a baseline from that
// box. If those two are not exact inverses, every migrated card's text
// shifts vertically — the single most visible way this refactor could
// silently damage existing designs.
func TestTextBaselineRoundTrip(t *testing.T) {
	cases := []struct {
		name      string
		baseline  float64
		fontSize  float64
		lineCount int
	}{
		{"single line", 120, 24, 1},
		{"two lines", 87.5, 18.25, 2},
		{"tiny font", 40, 6, 1},
		{"three lines", 200, 30, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &seedCtx{}
			text := "A"
			for i := 1; i < tc.lineCount; i++ {
				text += "\nB"
			}
			s.addTextAt("t", "t", 100, tc.baseline, 80,
				models.TextProps{Text: text, FontSize: tc.fontSize, Color: "#000", Align: "center"}, tc.lineCount, 1)

			layout := &models.CardLayout{Canvas: models.CardCanvas{W: 200, H: 400}, Elements: s.elements}
			svg := RenderLayoutSVG(layout, &LayoutAssets{})

			m := textYRe.FindAllStringSubmatch(svg, -1)
			if len(m) != tc.lineCount {
				t.Fatalf("expected %d rendered lines, got %d", tc.lineCount, len(m))
			}
			ox, oy := origin(s.elements[0])
			// The FIRST line's baseline must land exactly where the old
			// renderer put it.
			assertClose(t, oy+mustFloat(t, m[0][2]), tc.baseline, "first baseline")
			// Subsequent lines step by fontSize*1.2, the old lineGap.
			for i := 1; i < tc.lineCount; i++ {
				want := tc.baseline + float64(i)*tc.fontSize*1.2
				assertClose(t, oy+mustFloat(t, m[i][2]), want, fmt.Sprintf("baseline line %d", i))
			}
			// Centered text anchors at the box's own center, which must be
			// the cx it was seeded with.
			assertClose(t, ox+mustFloat(t, m[0][1]), 100, "text anchor x")
			assertClose(t, mustFloat(t, m[0][3]), tc.fontSize, "font size")
		})
	}
}

// TestQRFrameRoundTrip pins the other derived conversion: addQR turns the old
// drawQRWithFrame(cx, top, size) call into a box, and renderQRElement turns
// that box back into a halo rect, a QR viewport and a caption position. All
// three must match what drawQRWithFrame drew.
func TestQRFrameRoundTrip(t *testing.T) {
	const cx, qrTop, qrSize, hint = 150.0, 220.0, 90.0, 8.0

	s := &seedCtx{qrTargets: []seedQRTarget{{TargetType: models.QRTargetProfile}}}
	s.addQR("qr", "qr", cx, qrTop, qrSize, 0)
	// Toca/Escanea are sibling elements now, not props on the QR — seeded
	// separately, right where the QR-embedded caption used to draw itself.
	s.addPromptPair("qr", cx, promptRowCY(qrTop, qrSize, hint), hint, "#111827", true, true, 0)

	layout := &models.CardLayout{Canvas: models.CardCanvas{W: 400, H: 600}, Elements: s.elements}
	svg := RenderLayoutSVG(layout, &LayoutAssets{
		QRSVGs: map[string]string{"qr": `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"></svg>`},
	})

	// What drawQRWithFrame used to compute, restated here directly.
	haloSide := qrSize * qrSquareScale
	haloR := haloSide / 2
	haloCY := qrTop + qrSize/2

	ox, oy := origin(s.elements[0])

	fm := frameRe.FindStringSubmatch(svg)
	if fm == nil {
		t.Fatal("no white QR frame rendered")
	}
	assertClose(t, ox+mustFloat(t, fm[1]), cx-haloR, "frame x")
	assertClose(t, oy+mustFloat(t, fm[2]), haloCY-haloR, "frame y")
	assertClose(t, mustFloat(t, fm[3]), haloSide, "frame width")
	assertClose(t, mustFloat(t, fm[4]), haloSide, "frame height")
	assertClose(t, mustFloat(t, fm[5]), haloSide*qrHaloRadius, "frame corner radius")

	qm := qrSVGRe.FindStringSubmatch(svg)
	if qm == nil {
		t.Fatal("no positioned QR rendered")
	}
	assertClose(t, ox+mustFloat(t, qm[1]), cx-qrSize/2, "qr x")
	assertClose(t, oy+mustFloat(t, qm[2]), qrTop, "qr y")
	assertClose(t, mustFloat(t, qm[3]), qrSize, "qr size")

	// The old QR-embedded caption put its row at haloCY + haloR + hint*1.9 —
	// exactly what promptRowCY computes, which is the whole point of having
	// pulled it out into its own named function: every seeder and this test
	// agree with each other by construction, not by coincidence.
	wantRowCY := haloCY + haloR + hint*1.9
	assertClose(t, promptRowCY(qrTop, qrSize, hint), wantRowCY, "prompt row center")

	// "Toca" now renders inside its OWN element's <g>, translated by that
	// element's own X/Y — not the QR's.
	tapEl := findElement(t, s.elements, "qr-tap")
	_, toy := origin(*tapEl)

	captionRe := regexp.MustCompile(`y="([-\d.]+)" font-size="[-\d.]+" font-weight="700" fill="[^"]*">Toca<`)
	cm := captionRe.FindStringSubmatch(svg)
	if cm == nil {
		t.Fatal("no Toca prompt rendered")
	}
	// renderPromptElement centers its box on H/2 and adds fontSize*0.32 for
	// the text baseline; the box sits at rowCY-H/2, so the absolute baseline
	// works out to rowCY + fontSize*0.32 regardless of the box's own X.
	assertClose(t, toy+mustFloat(t, cm[1]), wantRowCY+hint*0.32, "Toca baseline")
}

func findElement(t *testing.T, elements []models.CardElement, id string) *models.CardElement {
	t.Helper()
	for i := range elements {
		if elements[i].ID == id {
			return &elements[i]
		}
	}
	t.Fatalf("no element with id %q", id)
	return nil
}

// TestStarRowSpacing checks the one place a multi-glyph element replaced a
// hand-rolled loop: drawStarRow spaced five stars 2.6r apart starting at
// cx - 2*spacing. The icon element reproduces that only if the gap fraction
// and the row width agree with each other.
func TestStarRowSpacing(t *testing.T) {
	const cx, starY, r = 200.0, 150.0, 5.0

	rowW := r * 12.4
	el := models.CardElement{
		ID: "stars", Type: models.ElementIcon,
		X: cx - rowW/2, Y: starY - r, W: rowW, H: r * 2,
		Props: props(models.IconProps{Name: "star", Color: starGold, Count: 5, Gap: 1.3}),
	}
	layout := &models.CardLayout{Canvas: models.CardCanvas{W: 400, H: 300}, Elements: []models.CardElement{el}}
	svg := RenderLayoutSVG(layout, &LayoutAssets{})

	// starIcon draws a polygon whose first point is directly above center,
	// i.e. at x == cx of that glyph.
	got := starCXRe.FindAllStringSubmatch(svg, -1)
	if len(got) != 5 {
		t.Fatalf("expected 5 stars, got %d", len(got))
	}
	spacing := r * 2.6
	startX := cx - spacing*2
	for i, m := range got {
		// The rendered X is relative to the element's own translated group.
		want := startX + float64(i)*spacing - el.X
		assertClose(t, mustFloat(t, m[1]), want, fmt.Sprintf("star %d center", i))
	}
}

// TestSeedsEveryCombination is a broad smoke test: every built-in design, at
// every size, in every style, must produce a valid tree that renders. It is
// what would have caught a nil-deref or an empty tree in a combination
// nobody happened to open by hand.
func TestSeedsEveryCombination(t *testing.T) {
	sizes := []models.PrintCardSizePreset{
		models.SizeBusinessCard, models.SizeTableTent,
		models.SizeStickerSquare, models.SizeDoorHanger, models.SizeCustom,
	}
	layouts := []models.PrintCardLayout{
		models.PrintCardGoogleReview, models.PrintCardSocialFollow,
		models.PrintCardMenuScan, models.PrintCardLoyaltyCard,
		models.PrintCardMultiQR, models.PrintCardThankYou,
	}
	styles := []string{"block", "split", "corners", "framed", "banner", "spotlight", "diagonal", "outline", "pattern"}
	content := `{"headline":"¿Nos regalas una reseña muy larga que obliga a envolver?","subheadline":"Gracias","platform":"instagram","discount_code":"ABC123","discount_label":"10% de descuento","left_label":"Síguenos","right_label":"Reseña"}`

	for _, size := range sizes {
		for _, lk := range layouts {
			for _, style := range styles {
				name := fmt.Sprintf("%s/%s/%s", size, lk, style)
				t.Run(name, func(t *testing.T) {
					card := testCard(lk, size, content, fmt.Sprintf(`{"style":%q,"pattern":"dots"}`, style))
					if size == models.SizeCustom {
						w, h := 9.0, 5.0
						card.CustomWidthCm, card.CustomHeightCm = &w, &h
					}
					layout, svg := renderSeeded(t, card)
					if len(layout.Elements) == 0 {
						t.Fatal("seeded an empty tree")
					}
					if len(svg) < 200 {
						t.Fatalf("suspiciously short SVG (%d bytes)", len(svg))
					}
					// Every element must carry a usable box — a zero-size
					// element is invisible in the editor and impossible to
					// grab, which is the free-editor equivalent of losing it.
					for _, el := range layout.Elements {
						if el.W <= 0 || el.H <= 0 {
							t.Errorf("element %s has a degenerate box %.2fx%.2f", el.ID, el.W, el.H)
						}
					}
					// The tree must round-trip through JSON unchanged —
					// it is stored as JSON and reloaded on every edit.
					encoded, err := json.Marshal(layout)
					if err != nil {
						t.Fatalf("encode: %v", err)
					}
					var back models.CardLayout
					if err := json.Unmarshal(encoded, &back); err != nil {
						t.Fatalf("decode: %v", err)
					}
					if len(back.Elements) != len(layout.Elements) {
						t.Errorf("round-trip lost elements: %d -> %d", len(layout.Elements), len(back.Elements))
					}
				})
			}
		}
	}
}

// TestLegacyScanSlotsPreserved is the compatibility guarantee that matters
// most in the real world: cards are already printed and in customers' hands
// encoding /q/:code and /q/:code/left. Those URLs must keep resolving after
// the tree replaces the old columns.
func TestLegacyScanSlotsPreserved(t *testing.T) {
	t.Run("single QR keeps the empty segment", func(t *testing.T) {
		card := testCard(models.PrintCardGoogleReview, models.SizeBusinessCard, `{}`, "")
		layout := SeedCardLayout(card, nil, false, "")

		el := layout.FindQRBySlot("")
		if el == nil {
			t.Fatal("/q/:code no longer resolves to any QR")
		}
		if got := el.ScanSlot(); got != "" {
			t.Errorf("re-export would change the printed URL: slot %q, want empty", got)
		}
	})

	t.Run("two QRs keep left and right", func(t *testing.T) {
		card := testCard(models.PrintCardMultiQR, models.SizeTableTent,
			`{"left_target_type":"profile","right_target_type":"loyalty"}`, "")
		layout := SeedCardLayout(card, nil, false, "")

		for _, slot := range []string{"left", "right"} {
			el := layout.FindQRBySlot(slot)
			if el == nil {
				t.Fatalf("/q/:code/%s no longer resolves", slot)
			}
			if got := el.ScanSlot(); got != slot {
				t.Errorf("re-export would change the printed URL for %s: got %q", slot, got)
			}
		}

		left := layout.FindQRBySlot("left")
		right := layout.FindQRBySlot("right")
		if left.ID == right.ID {
			t.Fatal("both slots resolved to the same element")
		}
		var lp, rp models.QRProps
		_ = left.DecodeProps(&lp)
		_ = right.DecodeProps(&rp)
		if lp.TargetType != models.QRTargetProfile || rp.TargetType != models.QRTargetLoyalty {
			t.Errorf("destinations swapped or lost: left=%s right=%s", lp.TargetType, rp.TargetType)
		}
	})

	t.Run("an unknown slot does not resolve", func(t *testing.T) {
		card := testCard(models.PrintCardMultiQR, models.SizeTableTent, `{}`, "")
		layout := SeedCardLayout(card, nil, false, "")
		if el := layout.FindQRBySlot("middle"); el != nil {
			t.Errorf("unknown slot resolved to %s; the caller must fall back instead", el.ID)
		}
	})
}

// TestDieCutClearance checks that a door hanger's content still clears the
// physical hole. The old renderer enforced this by pushing its panel down;
// the seeder has to reproduce that or the first thing punched out of a
// printed hanger is its own headline.
func TestDieCutClearance(t *testing.T) {
	card := testCard(models.PrintCardGoogleReview, models.SizeDoorHanger, `{"headline":"Reséñanos"}`, "")
	layout := SeedCardLayout(card, nil, false, "")

	if layout.Canvas.DieCut == nil {
		t.Fatal("door hanger has no die cut")
	}
	dc := layout.Canvas.DieCut
	holeBottom := dc.CY + dc.R

	for _, el := range layout.Elements {
		// Background decorations are allowed behind the hole; content is not.
		if el.Type == models.ElementShape {
			continue
		}
		if el.Y < holeBottom {
			t.Errorf("element %s starts at y=%.2f, inside the die-cut hole (bottom %.2f)", el.ID, el.Y, holeBottom)
		}
	}
}
