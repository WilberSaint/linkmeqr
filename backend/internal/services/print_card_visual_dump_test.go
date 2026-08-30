package services

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"linkmeqr/backend/internal/models"
)

// TestDumpStyleSamples renders one sample card per style to disk so the
// designs can be looked at rather than reasoned about. Print cards are a
// visual product and a passing assertion says nothing about whether a layout
// is any good; this is how you check.
//
//	CARD_DUMP_DIR=/tmp/cards go test ./internal/services -run TestDumpStyleSamples
//
// It skips unless that variable is set, so it never writes files during a
// normal test run.
func TestDumpStyleSamples(t *testing.T) {
	dir := os.Getenv("CARD_DUMP_DIR")
	if dir == "" {
		t.Skip("set CARD_DUMP_DIR to dump sample cards")
	}
	only := os.Getenv("CARD_DUMP_ONLY") // "layout/style/size" filter, substring match

	layouts := []models.PrintCardLayout{
		models.PrintCardGoogleReview, models.PrintCardSocialFollow, models.PrintCardMenuScan,
		models.PrintCardLoyaltyCard, models.PrintCardMultiQR, models.PrintCardThankYou,
	}
	styles := []string{"block", "split", "corners", "framed", "banner", "spotlight", "diagonal", "outline", "pattern"}
	sizes := []models.PrintCardSizePreset{models.SizeTableTent, models.SizeBusinessCard}

	for _, size := range sizes {
		for _, lk := range layouts {
			for _, style := range styles {
				name := fmt.Sprintf("%s_%s_%s", size, lk, style)
				if only != "" && !strings.Contains(name, only) {
					continue
				}
				content := `{"headline":"¿Nos regalas una reseña?","subheadline":"Nos ayuda muchísimo","platform":"instagram","left_label":"Síguenos","right_label":"Reséñanos","discount_code":"GRACIAS10","discount_label":"10% de descuento"}`
				card := testCard(lk, size, content,
					fmt.Sprintf(`{"style":%q,"background":"#5f6f52","accent":"#5f6f52"}`, style))
				layout := SeedCardLayout(card, nil, false, "")
				assets := &LayoutAssets{QRSVGs: map[string]string{}}
				for _, el := range layout.Elements {
					if el.Type == models.ElementQR {
						assets.QRSVGs[el.ID] = fakeQR()
					}
				}
				svg := RenderLayoutSVG(layout, assets)
				if err := os.WriteFile(fmt.Sprintf("%s/%s.svg", dir, name), []byte(svg), 0o644); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}

// fakeQR is a checkerboard standing in for a real code, so the samples show
// the QR's true size and placement without needing an encoder.
func fakeQR() string {
	out := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20"><rect width="20" height="20" fill="#fff"/>`
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			if (x*7+y*3)%5 < 2 {
				out += fmt.Sprintf(`<rect x="%d" y="%d" width="1" height="1" fill="#000"/>`, x, y)
			}
		}
	}
	return out + `</svg>`
}
