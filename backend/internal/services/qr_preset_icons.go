package services

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
)

// drawPresetIcon fills innerRect with a procedurally-drawn icon silhouette —
// no external image assets involved, matching the rest of this package's
// hand-rolled rasterization style. Unlike a logo (see drawCenterBadge), a
// preset icon is just a flat single-color glyph with no silhouette of its
// own to blend into, so it gets a small circular bg halo first — round
// rather than a square badge, so it still reads as sitting IN the code
// rather than boxed on top of it.
func drawPresetIcon(img *image.RGBA, icon string, rect image.Rectangle, fg, bg color.RGBA) {
	haloRect := inflateRect(rect, 1.35)
	cx := float64(haloRect.Min.X+haloRect.Max.X) / 2
	cy := float64(haloRect.Min.Y+haloRect.Max.Y) / 2
	drawFilledCircle(img, cx, cy, float64(haloRect.Dx())/2, bg)

	switch icon {
	case "heart":
		fillIconMask(img, rect, fg, heartTest)
	case "star":
		pts := starPolygon(5, 0.42, 1.0)
		fillIconMask(img, rect, fg, func(u, v float64) bool { return pointInPolygon(u, v, pts) })
	case "matcha", "leaf":
		fillIconMask(img, rect, fg, leafTest)
	case "coffee":
		fillIconMask(img, rect, fg, coffeeTest)
	case "gift":
		fillIconMask(img, rect, fg, giftBoxTest)
		fillIconMask(img, rect, bg, giftRibbonTest)
	}
}

// fillIconMask calls test with pixel coordinates normalized to roughly
// [-1, 1] over rect and fills every pixel where it returns true.
func fillIconMask(img *image.RGBA, rect image.Rectangle, c color.Color, test func(u, v float64) bool) {
	cx := float64(rect.Min.X+rect.Max.X) / 2
	cy := float64(rect.Min.Y+rect.Max.Y) / 2
	r := float64(rect.Dx()) / 2
	if r <= 0 {
		return
	}
	for py := rect.Min.Y; py < rect.Max.Y; py++ {
		for px := rect.Min.X; px < rect.Max.X; px++ {
			u := (float64(px) - cx) / r
			v := (float64(py) - cy) / r
			if test(u, v) {
				img.Set(px, py, c)
			}
		}
	}
}

// heartTest uses the classic implicit heart curve (u²+v²-1)³ - u²v³ <= 0,
// with v flipped so the point sits at the bottom of the icon (image
// coordinates grow downward, the canonical formula expects "up" positive).
func heartTest(u, v float64) bool {
	uu, vv := u*1.15, -v*1.15
	val := math.Pow(uu*uu+vv*vv-1, 3) - uu*uu*vv*vv*vv
	return val <= 0
}

type point2D struct{ X, Y float64 }

func starPolygon(points int, innerRatio, outerRatio float64) []point2D {
	pts := make([]point2D, 0, points*2)
	for i := 0; i < points*2; i++ {
		angle := -math.Pi/2 + float64(i)*math.Pi/float64(points)
		r := outerRatio
		if i%2 == 1 {
			r = innerRatio
		}
		pts = append(pts, point2D{r * math.Cos(angle), r * math.Sin(angle)})
	}
	return pts
}

func pointInPolygon(x, y float64, pts []point2D) bool {
	inside := false
	j := len(pts) - 1
	for i := range pts {
		xi, yi := pts[i].X, pts[i].Y
		xj, yj := pts[j].X, pts[j].Y
		if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
			inside = !inside
		}
		j = i
	}
	return inside
}

// leafTest is a vesica (lens) formed by two overlapping circles on a
// diagonal, read as a simple leaf silhouette.
func leafTest(u, v float64) bool {
	d, r := 0.42, 0.78
	du1, dv1 := u-d, v-d
	du2, dv2 := u+d, v+d
	return du1*du1+dv1*dv1 <= r*r && du2*du2+dv2*dv2 <= r*r
}

// coffeeTest draws a mug: a rounded-bottom cup body plus a ring-segment handle.
func coffeeTest(u, v float64) bool {
	inBody := u >= -0.55 && u <= 0.55 && v >= -0.35 && v <= 0.75
	if inBody {
		corner := 0.18
		cut := false
		if u < -0.55+corner && v > 0.75-corner {
			dx, dy := u-(-0.55+corner), v-(0.75-corner)
			cut = dx*dx+dy*dy > corner*corner
		} else if u > 0.55-corner && v > 0.75-corner {
			dx, dy := u-(0.55-corner), v-(0.75-corner)
			cut = dx*dx+dy*dy > corner*corner
		}
		if !cut {
			return true
		}
	}

	hcx, hcy := 0.75, 0.15
	outerR, innerR := 0.34, 0.17
	dx, dy := u-hcx, v-hcy
	dist2 := dx*dx + dy*dy
	return dist2 <= outerR*outerR && dist2 >= innerR*innerR
}

// giftBoxTest is the solid silhouette of a gift box with a small bow on top.
func giftBoxTest(u, v float64) bool {
	inBox := u >= -0.6 && u <= 0.6 && v >= -0.15 && v <= 0.75
	if inBox {
		corner := 0.14
		if u < -0.6+corner && v > 0.75-corner {
			dx, dy := u-(-0.6+corner), v-(0.75-corner)
			if dx*dx+dy*dy > corner*corner {
				inBox = false
			}
		} else if u > 0.6-corner && v > 0.75-corner {
			dx, dy := u-(0.6-corner), v-(0.75-corner)
			if dx*dx+dy*dy > corner*corner {
				inBox = false
			}
		}
	}
	lb := (u+0.18)*(u+0.18)+(v+0.32)*(v+0.32) <= 0.05
	rb := (u-0.18)*(u-0.18)+(v+0.32)*(v+0.32) <= 0.05
	return inBox || lb || rb
}

// giftRibbonTest punches a cross-shaped gap (rendered in the background
// color) through the box so it reads as a wrapped present.
func giftRibbonTest(u, v float64) bool {
	inBox := u >= -0.6 && u <= 0.6 && v >= -0.15 && v <= 0.75
	if !inBox {
		return false
	}
	vertical := u >= -0.1 && u <= 0.1
	horizontal := v >= -0.02 && v <= 0.18
	return vertical || horizontal
}

// svgPresetIcon writes an SVG approximation of the same preset icons, drawn
// with plain shapes rather than a pixel mask.
func svgPresetIcon(b *strings.Builder, icon string, cx, cy, r float64, fg, bg string) {
	fmt.Fprintf(b, `<circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s"/>`, cx, cy, r*1.35, bg)

	switch icon {
	case "heart":
		fmt.Fprintf(b, `<circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s"/>`, cx-0.28*r, cy-0.15*r, 0.35*r, fg)
		fmt.Fprintf(b, `<circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s"/>`, cx+0.28*r, cy-0.15*r, 0.35*r, fg)
		fmt.Fprintf(b, `<polygon points="%.2f,%.2f %.2f,%.2f %.2f,%.2f" fill="%s"/>`,
			cx-0.55*r, cy-0.05*r, cx+0.55*r, cy-0.05*r, cx, cy+0.6*r, fg)
	case "star":
		pts := starPolygon(5, 0.42, 1.0)
		var sb strings.Builder
		for _, p := range pts {
			fmt.Fprintf(&sb, "%.2f,%.2f ", cx+p.X*r, cy+p.Y*r)
		}
		fmt.Fprintf(b, `<polygon points="%s" fill="%s"/>`, strings.TrimSpace(sb.String()), fg)
	case "matcha", "leaf":
		fmt.Fprintf(b, `<ellipse cx="%.2f" cy="%.2f" rx="%.2f" ry="%.2f" transform="rotate(45 %.2f %.2f)" fill="%s"/>`,
			cx, cy, 0.75*r, 0.4*r, cx, cy, fg)
	case "coffee":
		fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.2f" fill="%s"/>`,
			cx-0.55*r, cy-0.35*r, 1.1*r, 1.1*r, 0.18*r, fg)
		fmt.Fprintf(b, `<circle cx="%.2f" cy="%.2f" r="%.2f" fill="none" stroke="%s" stroke-width="%.2f"/>`,
			cx+0.75*r, cy+0.15*r, 0.25*r, fg, 0.17*r)
	case "gift":
		fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.2f" fill="%s"/>`,
			cx-0.6*r, cy-0.15*r, 1.2*r, 0.9*r, 0.14*r, fg)
		fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s"/>`,
			cx-0.1*r, cy-0.15*r, 0.2*r, 0.9*r, bg)
		fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s"/>`,
			cx-0.6*r, cy-0.02*r, 1.2*r, 0.2*r, bg)
		fmt.Fprintf(b, `<circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s"/>`, cx-0.18*r, cy-0.32*r, 0.22*r, fg)
		fmt.Fprintf(b, `<circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s"/>`, cx+0.18*r, cy-0.32*r, 0.22*r, fg)
	}
}
