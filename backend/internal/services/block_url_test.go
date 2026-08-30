package services

import (
	"testing"

	"linkmeqr/backend/internal/models"
)

func TestNormalizeBlockURL(t *testing.T) {
	cases := []struct {
		name      string
		blockType models.BlockType
		in        string
		want      string
	}{
		// The bug this exists for: a bare host became a relative path, so
		// the public page sent visitors to /p/www.cafemani.com.
		{"bare host gets https", models.BlockWebsite, "www.cafemani.com", "https://www.cafemani.com"},
		{"host without www", models.BlockWebsite, "cafemani.com", "https://cafemani.com"},
		{"host with path", models.BlockWebsite, "instagram.com/cafemani", "https://instagram.com/cafemani"},
		{"surrounding spaces trimmed", models.BlockWebsite, "  cafemani.com  ", "https://cafemani.com"},

		// Already-valid values must survive untouched.
		{"https untouched", models.BlockWebsite, "https://cafemani.com", "https://cafemani.com"},
		{"http untouched", models.BlockWebsite, "http://cafemani.com", "http://cafemani.com"},
		{"uppercase scheme untouched", models.BlockWebsite, "HTTPS://cafemani.com", "HTTPS://cafemani.com"},
		{"protocol-relative untouched", models.BlockWebsite, "//cafemani.com", "//cafemani.com"},
		{"tel untouched", models.BlockPhone, "tel:+526441234567", "tel:+526441234567"},
		{"mailto untouched", models.BlockEmail, "mailto:hola@cafemani.com", "mailto:hola@cafemani.com"},

		// Contact blocks can legitimately hold a bare value; prefixing those
		// would corrupt the number/address rather than repair it.
		{"bare phone left alone", models.BlockPhone, "+52 644 123 4567", "+52 644 123 4567"},
		{"bare email left alone", models.BlockEmail, "hola@cafemani.com", "hola@cafemani.com"},
		{"bare whatsapp left alone", models.BlockWhatsapp, "526441234567", "526441234567"},

		// Not a URL at all — inventing one would be worse than leaving it.
		{"plain text left alone", models.BlockWebsite, "pregunta en caja", "pregunta en caja"},
		{"empty left alone", models.BlockWebsite, "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := tc.in
			got := NormalizeBlockURL(tc.blockType, &in)
			if got == nil {
				t.Fatalf("got nil, want %q", tc.want)
			}
			if *got != tc.want {
				t.Errorf("NormalizeBlockURL(%q) = %q, want %q", tc.in, *got, tc.want)
			}
		})
	}
}

func TestNormalizeBlockURLNil(t *testing.T) {
	// A block with no URL at all (a text note, a gallery) must stay nil
	// rather than become an empty string the renderer would treat as a link.
	if got := NormalizeBlockURL(models.BlockText, nil); got != nil {
		t.Errorf("nil URL became %v, want nil", *got)
	}
}
