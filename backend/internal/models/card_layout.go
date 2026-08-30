package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// CardLayoutVersion is the schema version stamped on every persisted
// layout. It is NOT the per-card revision number (that's
// PrintCard.LayoutVersion / print_card_layout_versions.version) — this one
// only ever changes when the shape of the JSON below changes, so a future
// reader can tell "this tree was written by an older encoder" apart from
// "this is the card's 7th saved revision".
const CardLayoutVersion = 1

// CardUnitsPerInch is the layout tree's coordinate unit: every x/y/w/h in a
// CardLayout is expressed in hundredths of a physical inch. Deliberately
// absolute rather than normalized 0..1 — a free-form editor lets an element
// sit anywhere, so resizing the card must NOT silently stretch or skew what
// the designer placed; changing a card's physical size re-seeds the tree
// from its template instead. This is the same scale the SVG viewBox has
// always used, so migrated cards keep their exact geometry.
const CardUnitsPerInch = 100.0

// ElementType enumerates what a single node in the layout tree draws. These
// six cover everything the old hardcoded layouts drew, and — unlike the
// old per-template Content fields — nothing here knows what a
// "google_review card" or a "thank you card" is.
type ElementType string

const (
	ElementText  ElementType = "text"
	ElementImage ElementType = "image"
	ElementQR    ElementType = "qr"
	ElementShape ElementType = "shape"
	ElementIcon  ElementType = "icon"
	// ElementPrompt is one "[icon] label" call-to-action — "Toca", "Escanea",
	// or any other short instruction paired with a glyph. It used to live
	// baked into a QR element's own props (see QRProps' deprecated Caption
	// fields); as its own element type it is a normal citizen of the tree —
	// selectable, draggable, resizable and deletable independently of any
	// QR, the way a business wanting "Escanea" on the left and "Toca" on the
	// right of the same code needs it to be.
	ElementPrompt ElementType = "prompt"
)

var validElementTypes = map[ElementType]bool{
	ElementText: true, ElementImage: true, ElementQR: true,
	ElementShape: true, ElementIcon: true, ElementPrompt: true,
}

// CardElement is one free-standing node of the layout tree. Geometry is
// uniform across every type (that's the whole point: the editor moves,
// resizes and rotates any element with the same code path); everything
// type-specific lives behind Props, decoded by whichever renderer handles
// that type.
type CardElement struct {
	ID   string      `json:"id"`
	Type ElementType `json:"type"`

	// X/Y is the element's top-left corner, W/H its bounding box, all in
	// CardUnitsPerInch units. Rotation is in degrees clockwise about the
	// box's own center.
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	W        float64 `json:"w"`
	H        float64 `json:"h"`
	Rotation float64 `json:"rotation"`

	// ZIndex is the paint order, lowest first. The elements slice's own
	// order is not authoritative — NormalizeZ sorts by this and rewrites it
	// to a dense 0..n-1 sequence on every save, so the layers panel and the
	// renderer can never disagree.
	ZIndex int `json:"z_index"`

	// Hidden keeps an element in the tree (and in the layers panel) without
	// drawing it. Locked keeps it drawn but not selectable/draggable in the
	// editor; the backend only stores it, since nothing server-side can
	// "move" an element anyway.
	Hidden bool `json:"hidden"`
	Locked bool `json:"locked"`

	// Name is the optional user-facing label shown in the layers panel.
	// Empty means the editor derives one from the type and props.
	Name string `json:"name,omitempty"`

	// Opacity applies to the whole element, on top of whatever per-prop
	// opacity its own renderer applies. A missing value means fully
	// opaque — read it through EffectiveOpacity, never directly, so a
	// deliberate 0 stays distinguishable from "not set".
	Opacity *float64 `json:"opacity,omitempty"`

	Props json.RawMessage `json:"props"`
}

// EffectiveOpacity treats an absent opacity as fully opaque.
func (e CardElement) EffectiveOpacity() float64 {
	if e.Opacity == nil {
		return 1
	}
	if *e.Opacity < 0 {
		return 0
	}
	if *e.Opacity > 1 {
		return 1
	}
	return *e.Opacity
}

// DecodeProps unmarshals this element's type-specific props into dst.
// Absent props decode as the zero value rather than erroring, so an element
// written by an older editor build (before some prop existed) still renders
// with that prop's default.
func (e CardElement) DecodeProps(dst any) error {
	if len(e.Props) == 0 || string(e.Props) == "null" {
		return nil
	}
	return json.Unmarshal(e.Props, dst)
}

// TextProps is the editable surface of a text element. Deliberately generic
// typography — nothing here is "the headline" or "the discount label"; those
// were template-specific form fields in the old model.
type TextProps struct {
	Text       string  `json:"text"`
	FontFamily string  `json:"font_family"`
	FontSize   float64 `json:"font_size"`
	FontWeight int     `json:"font_weight"`
	Color      string  `json:"color"`
	Align      string  `json:"align"`       // left | center | right
	LineHeight float64 `json:"line_height"` // multiple of font size
	Letter     float64 `json:"letter_spacing"`
	Italic     bool    `json:"italic"`
	Underline  bool    `json:"underline"`
	Uppercase  bool    `json:"uppercase"`
}

// ImageProps points at an uploaded media row, or at a well-known profile
// asset resolved at render time. Source "logo" means "this business's own
// logo", which is why an image element can be meaningful before any media
// id exists for it.
type ImageProps struct {
	Source  string  `json:"source"` // media | logo
	MediaID string  `json:"media_id,omitempty"`
	Fit     string  `json:"fit"`   // cover | contain
	Shape   string  `json:"shape"` // circle | rounded | square
	Radius  float64 `json:"radius"`
}

// QRProps carries where this QR points plus how its frame is drawn. The
// target fields mirror the card-level qr_target_type/value columns, which
// stay in place only for cards that predate the layout tree.
type QRProps struct {
	TargetType  QRTargetType `json:"target_type"`
	TargetValue *string      `json:"target_value,omitempty"`

	// LegacySlot preserves the exact /q/:code/:slot segment a previously
	// EXPORTED card baked into its printed QR — "left"/"right" for the old
	// two-QR design, and the empty string for every single-QR card, whose
	// printed URL had no segment at all. Physical cards are already in the
	// wild encoding those URLs, so the redirect has to keep resolving them
	// even though the tree addresses QRs by element id now.
	//
	// It is a pointer precisely because "printed with no segment" and "never
	// printed, address me by id" are different states that both look like ""
	// once decoded. nil means the latter: a QR element the designer added
	// after the refactor, which has no legacy URL to honor.
	LegacySlot *string `json:"legacy_slot,omitempty"`

	Frame       string  `json:"frame"` // none | square | rounded
	FrameFill   string  `json:"frame_fill"`
	FrameRadius float64 `json:"frame_radius"` // fraction of the frame's side
	FramePad    float64 `json:"frame_pad"`    // fraction of the QR's own size
	Shadow      bool    `json:"shadow"`

	// CaptionMode picks what the call-to-action under the code is built from:
	// "icons" (ShowTap/ShowScan below, the default for new QRs), "text" (the
	// literal CaptionText, no icon), or "none". Empty decodes as legacy —
	// see Caption below — so a card saved before this field existed keeps
	// rendering exactly as it did.
	CaptionMode string `json:"caption_mode,omitempty"`

	// Toca and Escanea are two independent prompts, each with its own icon
	// glyph and wording, rather than one hardcoded "Toca · Escanea" pair.
	// A business whose QR only supports scanning (no NFC tag on the card) can
	// show just one; a business with an NFC tag can show either, both, or
	// reword either independently ("Acerca tu teléfono" / "Escanéame").
	ShowTap  bool   `json:"show_tap"`
	TapIcon  string `json:"tap_icon,omitempty"` // an iconGlyph name; "" uses the default
	TapText  string `json:"tap_text,omitempty"` // "" uses "Toca"
	ShowScan bool   `json:"show_scan"`
	ScanIcon string `json:"scan_icon,omitempty"` // an iconGlyph name; "" uses the default
	ScanText string `json:"scan_text,omitempty"` // "" uses "Escanea"

	// Caption is the pre-individual-segments format: "none", "dual" (Toca ·
	// Escanea together), "tap"/"scan"/"scan_me" (one prompt), or "text" (the
	// old single CaptionText line). Read only when CaptionMode is empty, so
	// a QR element saved before Toca/Escanea became independent renders
	// identically to how it was printed.
	Caption     string `json:"caption,omitempty"`
	CaptionText string `json:"caption_text,omitempty"`

	CaptionSize float64 `json:"caption_size"`
	CaptionFill string  `json:"caption_fill"`
	// CaptionBare drops the pill behind the caption, leaving just the icons
	// and labels. This is the default for seeded cards: the translucent
	// capsule reads as a muddy smear over a saturated brand colour, and the
	// bare version prints far more cleanly.
	CaptionBare bool `json:"caption_bare"`
	// CaptionMaxWidth caps how wide the caption may grow, in card units; it
	// shrinks to fit rather than overflowing. Two QRs side by side would
	// otherwise draw captions wider than their own half of the card and
	// overlap each other. 0 means "no wider than this element's own box".
	CaptionMaxWidth float64 `json:"caption_max_width,omitempty"`
}

// ShapeProps covers every non-text decoration: the old styles' corner
// triangles, the split wave, the framed ring, the discount coupon's dashed
// border. Points/Path let the seed templates express the exact geometry the
// hardcoded renderers used to draw inline.
type ShapeProps struct {
	Kind        string  `json:"kind"` // rect | ellipse | line | polygon | path
	Fill        string  `json:"fill"`
	Stroke      string  `json:"stroke"`
	StrokeWidth float64 `json:"stroke_width"`
	DashArray   string  `json:"dash_array,omitempty"`
	Radius      float64 `json:"radius"`

	// Points (polygon) and Path (path) are in the element's OWN local box
	// coordinates, 0..W and 0..H, so moving or resizing the element
	// transforms them for free.
	Points string `json:"points,omitempty"`
	Path   string `json:"path,omitempty"`

	Shadow bool `json:"shadow"`

	// Pattern draws one of the card background's own textures (dots | lines
	// | grid | waves | circles) over Fill instead of a flat colour — a
	// shape is not limited to plain fills any more than the card itself is.
	// Empty means solid Fill, same as always. PatternInk defaults to Fill
	// when empty, so a fresh shape's texture is visible without an extra
	// step.
	Pattern    string `json:"pattern,omitempty"`
	PatternInk string `json:"pattern_ink,omitempty"`
}

// IconProps names one of the renderer's built-in glyphs. Keeping icons as
// their own type (rather than pre-baked SVG in a shape's Path) means the
// editor can recolor and swap them, and a business's platform icon stays a
// single editable choice instead of frozen artwork.
type IconProps struct {
	Name  string  `json:"name"` // google | instagram | facebook | ... | star | menu | gift | heart | pin
	Color string  `json:"color"`
	Count int     `json:"count"` // >1 lays the glyph out in a row (the rating stars)
	Gap   float64 `json:"gap"`   // spacing between glyphs, as a fraction of one glyph's diameter
}

// PromptProps is one "[icon] label" call-to-action — an icon glyph (any name
// iconGlyph understands, typically one of the tap/scan family) paired with a
// short instruction. It is deliberately generic rather than QR-specific:
// nothing here says "this belongs to a QR", so a business is free to add a
// "Reclama tu premio" prompt next to a gift icon just as easily as a
// "Escanea" prompt next to a code.
type PromptProps struct {
	Icon  string `json:"icon,omitempty"` // an iconGlyph name; empty draws text only
	Text  string `json:"text"`
	Color string `json:"color"`
	// Bare drops the pill behind the prompt, leaving just the icon and text —
	// the default, and the only option that has ever shipped, since the
	// translucent capsule reads as a muddy smear over a saturated colour.
	Bare bool `json:"bare"`
	// FontSize overrides the size the element's own box height would
	// otherwise derive. 0 means "size to the box", the same convention
	// TextProps and IconProps already use.
	FontSize float64 `json:"font_size,omitempty"`
}

// CardBackground is the canvas fill itself. It is not an element: nothing
// in the editor should be able to drag the card's own background off the
// card, and every layout has exactly one.
type CardBackground struct {
	Fill       string `json:"fill"`
	GradientTo string `json:"gradient_to,omitempty"`
	Pattern    string `json:"pattern,omitempty"` // dots | lines | grid | waves | circles
	PatternInk string `json:"pattern_ink,omitempty"`
}

// DieCut describes a physical hole punched through the printed piece — only
// the door_hanger preset has one. Kept on the canvas rather than as an
// element because it is a property of the paper, not of the design.
type DieCut struct {
	Kind string  `json:"kind"` // circle
	CX   float64 `json:"cx"`
	CY   float64 `json:"cy"`
	R    float64 `json:"r"`
}

// CardCanvas is the printable area every element is positioned within.
type CardCanvas struct {
	W       float64 `json:"w"`
	H       float64 `json:"h"`
	CornerR float64 `json:"corner_r"`
	DieCut  *DieCut `json:"die_cut,omitempty"`
}

// CardLayout is the complete, self-describing design of one print card —
// the single source of truth for both the editor and the exporter. It
// replaces the old (layout_key + content + color_overrides) triple, where
// the actual arrangement lived in Go code and only the strings lived in the
// database.
type CardLayout struct {
	Version    int            `json:"version"`
	Canvas     CardCanvas     `json:"canvas"`
	Background CardBackground `json:"background"`
	Elements   []CardElement  `json:"elements"`

	// SeededFrom records which built-in template this tree was first
	// generated from. Purely informational (the tree is authoritative from
	// then on) but it lets the editor offer "reset to template" and lets us
	// tell migrated cards apart from hand-built ones.
	SeededFrom string `json:"seeded_from,omitempty"`
}

var (
	ErrLayoutEmpty        = errors.New("layout has no canvas")
	ErrLayoutTooManyElems = errors.New("layout has too many elements")
)

// maxLayoutElements is a sanity bound on a single card's tree — far above
// any real design, low enough that a malformed or hostile payload can't
// make the renderer allocate unboundedly.
const maxLayoutElements = 500

// Validate rejects a layout that could not be rendered, or that would let a
// client store unbounded data. It deliberately does NOT reject unknown prop
// fields or off-canvas coordinates: a designer dragging something half off
// the edge is a legitimate work-in-progress state, not a bad request.
func (l *CardLayout) Validate() error {
	if l.Canvas.W <= 0 || l.Canvas.H <= 0 {
		return ErrLayoutEmpty
	}
	if len(l.Elements) > maxLayoutElements {
		return ErrLayoutTooManyElems
	}
	seen := make(map[string]bool, len(l.Elements))
	for i, e := range l.Elements {
		if e.ID == "" {
			return fmt.Errorf("element %d has no id", i)
		}
		if seen[e.ID] {
			return fmt.Errorf("duplicate element id %q", e.ID)
		}
		seen[e.ID] = true
		if !validElementTypes[e.Type] {
			return fmt.Errorf("element %q has unknown type %q", e.ID, e.Type)
		}
	}
	return nil
}

// NormalizeZ sorts the tree into paint order and rewrites ZIndex to a dense
// 0..n-1 run. Called on every save so the stored order is canonical: the
// layers panel renders the slice reversed (top layer first) and the
// exporter walks it forward, with no possibility of the two disagreeing
// because some client sent duplicate or sparse z values.
func (l *CardLayout) NormalizeZ() {
	sort.SliceStable(l.Elements, func(i, j int) bool {
		return l.Elements[i].ZIndex < l.Elements[j].ZIndex
	})
	for i := range l.Elements {
		l.Elements[i].ZIndex = i
	}
}

// ScanSlot is the /q/:code/:slot segment this QR element should encode when
// exported with tracking: its already-printed segment when it has one, so
// re-exporting an old card reproduces exactly the URL its earlier prints
// carry, and otherwise its own element id. Meaningless for non-QR elements.
func (e CardElement) ScanSlot() string {
	var p QRProps
	if err := e.DecodeProps(&p); err == nil && p.LegacySlot != nil {
		return *p.LegacySlot
	}
	return e.ID
}

// FindQRBySlot resolves the /q/:code/:slot path segment a printed card's QR
// encodes. New exports use the element's own id; cards printed before the
// layout refactor encode "left"/"right", matched here against the
// LegacySlot the migration preserved. Returns nil when the slot names
// nothing in this tree, which the caller treats as "fall back to the card's
// own columns".
func (l *CardLayout) FindQRBySlot(slot string) *CardElement {
	var firstQR *CardElement
	for i := range l.Elements {
		e := &l.Elements[i]
		if e.Type != ElementQR {
			continue
		}
		if firstQR == nil {
			firstQR = e
		}
		if e.ID == slot {
			return e
		}
		var p QRProps
		if err := e.DecodeProps(&p); err == nil && p.LegacySlot != nil && *p.LegacySlot == slot {
			return e
		}
	}
	// A card printed before the refactor with a single QR encoded /q/:code
	// and nothing else. Falling back to the tree's first QR keeps those
	// redirecting even if its element was later renamed or re-seeded.
	if slot == "" {
		return firstQR
	}
	return nil
}
