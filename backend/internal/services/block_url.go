package services

import (
	"strings"

	"linkmeqr/backend/internal/models"
)

// NormalizeBlockURL repairs the single most common thing a business gets
// wrong when filling in a block: pasting an address with no scheme.
//
// "www.cafemani.com" is a perfectly reasonable thing to type — it's what the
// browser bar shows — but with no scheme it is a RELATIVE url, so the public
// page sent visitors to /p/www.cafemani.com instead of the business's site.
// Nothing surfaced the mistake: the editor showed it saved, and only a
// customer hitting a dead page would ever notice.
//
// Normalizing on write (rather than patching it at render time) means the
// stored value is the one that's correct everywhere it's later used — the
// public page, the vCard, an export — and it only has to be reasoned about
// once.
func NormalizeBlockURL(blockType models.BlockType, raw *string) *string {
	if raw == nil {
		return nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return raw
	}

	// Schemes that are already meaningful as-is. tel:/mailto:/whatsapp: are
	// how the contact blocks legitimately store their values.
	lower := strings.ToLower(value)
	for _, scheme := range []string{"http://", "https://", "tel:", "mailto:", "sms:", "whatsapp:", "geo:"} {
		if strings.HasPrefix(lower, scheme) {
			return &value
		}
	}

	// These block types hold a bare value, not a URL — a phone number, an
	// email address, a Google Place ID — and prefixing them would corrupt
	// the data rather than fix it.
	switch blockType {
	case models.BlockPhone, models.BlockEmail, models.BlockWhatsapp:
		return &value
	}

	// A leading "//" is protocol-relative and already resolves correctly.
	if strings.HasPrefix(value, "//") {
		return &value
	}

	// Anything else that looks like a host gets https://. The dot check is
	// what keeps a stray note ("pregunta en caja") from being turned into a
	// link — better to leave an obviously-not-a-URL alone than to invent one.
	if strings.Contains(value, ".") && !strings.HasPrefix(value, "/") {
		fixed := "https://" + value
		return &fixed
	}

	return &value
}
