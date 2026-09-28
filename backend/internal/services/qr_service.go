package services

import (
	"fmt"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// QRService builds the app's public URLs and renders plain QR codes for
// them. Deliberately unstyled (black on white, no logo): the only QR the app
// still hands out is the loyalty card's, and a plain code is both the most
// reliably scannable and the cheapest to render.
type QRService struct {
	publicBaseURL string
}

func NewQRService(publicBaseURL string) *QRService {
	return &QRService{publicBaseURL: strings.TrimSuffix(publicBaseURL, "/")}
}

// ProfileURL is a profile's permanent public URL.
func (s *QRService) ProfileURL(slug string) string {
	return fmt.Sprintf("%s/p/%s", s.publicBaseURL, slug)
}

// AbsoluteURL turns a server-relative path (e.g. an uploaded media file's
// "/media/xxx.png") into a fully-qualified URL, for consumers outside the web
// app itself such as Google Wallet.
func (s *QRService) AbsoluteURL(path string) string {
	return s.publicBaseURL + path
}

// LoyaltyURL is the permanent URL a business's loyalty stamp card lives at —
// the same link a physical NFC tag would be programmed with, so a QR
// encoding this URL and an NFC tap both land the visitor on the same page.
func (s *QRService) LoyaltyURL(token string) string {
	return fmt.Sprintf("%s/loyalty/%s", s.publicBaseURL, token)
}

// qrPNGSize is large enough to print sharply on a counter sign.
const qrPNGSize = 1024

// RenderQRPNG renders content as a black-on-white PNG. Error correction is
// "medium", which survives normal wear on a printed code.
func RenderQRPNG(content string) ([]byte, error) {
	return qrcode.Encode(content, qrcode.Medium, qrPNGSize)
}

// RenderQRSVG renders content as a vector SVG (one path, one unit per
// module) for print at any size.
func RenderQRSVG(content string) (string, error) {
	q, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return "", err
	}
	bitmap := q.Bitmap() // includes the 4-module quiet zone
	n := len(bitmap)

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="crispEdges">`, n, n)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="#ffffff"/><path fill="#000000" d="`, n, n)
	for y, row := range bitmap {
		for x, dark := range row {
			if dark {
				fmt.Fprintf(&b, "M%d %dh1v1h-1z", x, y)
			}
		}
	}
	b.WriteString(`"/></svg>`)
	return b.String(), nil
}
