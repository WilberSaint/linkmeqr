package handlers

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
	"linkmeqr/backend/internal/services"
)

// Serving /p/:slug through the API rather than straight off the static
// frontend exists for one reason: link previews.
//
// The profile page is a SPA route, so the HTML every crawler receives is the
// same generic shell for every business — WhatsApp, Facebook and Google all
// read the document, not the JavaScript, so a business sharing its own link
// got "LinkMeQR — Todo tu negocio en un QR" and no image, identical to every
// other customer's page. WhatsApp is the channel these links actually travel
// through, which made it the most visible gap on the public page.
//
// This fetches that same shell and injects the business's own title,
// description and image into <head> before returning it. Real visitors get
// the untouched SPA (same markup, same scripts, it boots exactly as before);
// crawlers get tags that describe the specific business. Nothing is served
// conditionally on user-agent, so there is no cloaking to explain to Google.

// shellCache holds the SPA shell after the first successful fetch. The
// document only changes on redeploy, which restarts this process anyway, so
// caching it forever is both correct and what keeps a momentary frontend
// blip from taking the public page down with it.
type shellCache struct {
	mu       sync.RWMutex
	html     string
	fetched  bool
	lastTry  time.Time
	retryGap time.Duration
}

func (c *shellCache) get() (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.html, c.fetched
}

func (c *shellCache) set(doc string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.html, c.fetched = doc, true
}

// shouldRetry rate-limits re-fetching while the shell is still unavailable,
// so a down frontend produces one attempt every few seconds rather than one
// per request.
func (c *shellCache) shouldRetry() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.lastTry) < c.retryGap {
		return false
	}
	c.lastTry = time.Now()
	return true
}

// Matches the shell's own <title>…</title> so it can be dropped in favour of
// the business's. Deliberately non-greedy and case-insensitive; the shell is
// a build artifact with exactly one title, not arbitrary user input.
var titleTagRe = regexp.MustCompile(`(?is)<title>.*?</title>`)

type ShellHandler struct {
	profiles *services.ProfileService
	licenses *repository.LicenseRepository
	media    *repository.MediaRepository
	shellURL string
	baseURL  string
	client   *http.Client
	cache    *shellCache
}

func NewShellHandler(
	profiles *services.ProfileService,
	licenses *repository.LicenseRepository,
	media *repository.MediaRepository,
	shellURL, baseURL string,
) *ShellHandler {
	return &ShellHandler{
		profiles: profiles,
		licenses: licenses,
		media:    media,
		shellURL: shellURL,
		baseURL:  strings.TrimSuffix(baseURL, "/"),
		client:   &http.Client{Timeout: 5 * time.Second},
		cache:    &shellCache{retryGap: 5 * time.Second},
	}
}

func (h *ShellHandler) fetchShell(ctx context.Context) (string, error) {
	if doc, ok := h.cache.get(); ok {
		return doc, nil
	}
	if !h.cache.shouldRetry() {
		return "", errors.New("shell unavailable, retry throttled")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.shellURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("shell returned %d", resp.StatusCode)
	}
	// 2 MB is far more than an index.html shell ever is; the cap just keeps
	// a misconfigured shellURL from reading something unbounded.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}
	doc := string(body)
	if !strings.Contains(doc, "</head>") {
		return "", errors.New("shell has no </head> to inject into")
	}
	h.cache.set(doc)
	return doc, nil
}

// absoluteMediaURL turns a stored /media/... path into the absolute URL a
// crawler needs — og:image is fetched by Facebook's and WhatsApp's servers,
// which have no notion of the site's own origin.
func (h *ShellHandler) absoluteMediaURL(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return h.baseURL + "/" + strings.TrimPrefix(path, "/")
}

// ProfileShell handles GET /p/{slug}.
func (h *ShellHandler) ProfileShell(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	doc, err := h.fetchShell(r.Context())
	if err != nil {
		// Without the shell there is no app to serve, and guessing at its
		// markup would ship a broken page. A 503 is honest and transient.
		http.Error(w, "El sitio no está disponible en este momento.", http.StatusServiceUnavailable)
		return
	}

	tags := h.metaTagsFor(r.Context(), slug)
	if tags != "" {
		// The shell ships its own generic <title>. Leaving it in place would
		// win — crawlers and browsers both take the first one — so the
		// business's title only counts once the original is gone.
		doc = titleTagRe.ReplaceAllString(doc, "")
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Crawlers re-fetch on every share; a short cache keeps a burst of
	// shares off the database without pinning a stale title for long.
	w.Header().Set("Cache-Control", "public, max-age=60")
	_, _ = io.WriteString(w, strings.Replace(doc, "</head>", tags+"</head>", 1))
}

// metaTagsFor builds the <meta> block for one profile, falling back to the
// generic product tags when the profile is missing, unpublished or expired —
// exactly the cases where the page itself shows the "inactive" screen, and
// where leaking the business's details into a preview would be wrong.
func (h *ShellHandler) metaTagsFor(ctx context.Context, slug string) string {
	profile, err := h.profiles.GetBySlug(ctx, slug)
	if err != nil {
		return ""
	}

	license, err := h.licenses.GetByUserID(ctx, profile.UserID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return ""
	}
	if !profile.IsPublished || services.EffectiveStatus(license) != models.LicenseActive {
		return ""
	}

	title := profile.BusinessName
	description := ""
	if profile.Description != nil {
		description = *profile.Description
	}
	if description == "" {
		description = fmt.Sprintf("Conoce %s: enlaces, contacto y más en un solo lugar.", profile.BusinessName)
	}

	// The cover is the wide banner a preview card wants; the logo is the
	// fallback, and better than nothing even when it's square.
	image := ""
	if profile.CoverMediaID != nil {
		if m, err := h.media.GetByID(ctx, *profile.CoverMediaID); err == nil {
			image = h.absoluteMediaURL(m.FilePath)
		}
	}
	if image == "" && profile.LogoMediaID != nil {
		if m, err := h.media.GetByID(ctx, *profile.LogoMediaID); err == nil {
			image = h.absoluteMediaURL(m.FilePath)
		}
	}

	pageURL := fmt.Sprintf("%s/p/%s", h.baseURL, profile.Slug)

	var b strings.Builder
	b.WriteString("\n    <title>" + html.EscapeString(title) + "</title>\n")
	b.WriteString(`    <meta name="description" content="` + html.EscapeString(description) + "\" />\n")
	b.WriteString(`    <meta property="og:type" content="website" />` + "\n")
	b.WriteString(`    <meta property="og:site_name" content="LinkMeQR" />` + "\n")
	b.WriteString(`    <meta property="og:title" content="` + html.EscapeString(title) + "\" />\n")
	b.WriteString(`    <meta property="og:description" content="` + html.EscapeString(description) + "\" />\n")
	b.WriteString(`    <meta property="og:url" content="` + html.EscapeString(pageURL) + "\" />\n")
	if image != "" {
		b.WriteString(`    <meta property="og:image" content="` + html.EscapeString(image) + "\" />\n")
		// summary_large_image is what turns a thumbnail into a full-width
		// card on Twitter/X; WhatsApp keys off og:image alone.
		b.WriteString(`    <meta name="twitter:card" content="summary_large_image" />` + "\n")
	} else {
		b.WriteString(`    <meta name="twitter:card" content="summary" />` + "\n")
	}
	b.WriteString(`    <meta name="twitter:title" content="` + html.EscapeString(title) + "\" />\n")
	b.WriteString(`    <meta name="twitter:description" content="` + html.EscapeString(description) + "\" />\n")
	b.WriteString(`    <link rel="canonical" href="` + html.EscapeString(pageURL) + "\" />\n")

	return b.String()
}
