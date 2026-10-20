// Echo's mark: a window, and the one it repeats — the prompt in the front
// window and its echo behind it, fainter and up to the right. The generator
// draws every size from that one geometry: the tray's black template, the
// tray's echo pulse (the frames internal/gui plays), the Dock glyph and the
// app icon, light and dark.
//
//	go run build/icon/gen.go tray <out.png>
//	go run build/icon/gen.go tray-flap <dir> [frames]
//	go run build/icon/gen.go glyph <size> <out.png>
//	go run build/icon/gen.go app|app-dark <size> <out.png>
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"
	"strconv"
)

// The mark in its 44×44 box. The echo sits up and to the right, out of the
// front window's corner, the way a repeat arrives after the first sound.
var (
	echoXY, echoWH, echoR = [2]float64{13, 6.5}, [2]float64{24, 18}, 4.0
	frontXY, frontWH      = [2]float64{4.5, 13.5}, [2]float64{28, 23}
	frontR                = 5.0
	// the prompt: a chevron band and its cursor dot, inside the window
	chevron = [][2]float64{
		{12.4, 20.4}, {19.4, 25.5}, {12.4, 30.6}, // inner edge, down to the point
		{14.8, 33.6}, {25.2, 25.5}, {14.8, 17.4}, // outer edge, back up
	}
	cursor = circle(27.9, 25.5, 1.55)
)

func main() {
	var img image.Image
	switch {
	case len(os.Args) == 3 && os.Args[1] == "tray":
		img = tray()
	case len(os.Args) == 3 && os.Args[1] == "tray-flap":
		if err := trayFlap(os.Args[2], 44); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case len(os.Args) == 4 && os.Args[1] == "tray-flap":
		n, _ := strconv.Atoi(os.Args[3])
		if err := trayFlap(os.Args[2], n); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case len(os.Args) == 4 && os.Args[1] == "glyph":
		n, _ := strconv.Atoi(os.Args[2])
		img = glyph(n)
	case len(os.Args) == 4 && (os.Args[1] == "app" || os.Args[1] == "app-dark"):
		n, _ := strconv.Atoi(os.Args[2])
		img = app(n, os.Args[1] == "app-dark")
	default:
		fmt.Fprintln(os.Stderr, "usage: gen.go tray <out.png> | gen.go app|app-dark <size> <out.png>")
		os.Exit(2)
	}
	f, err := os.Create(os.Args[len(os.Args)-1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// trayK is the mark's size in its 22pt square: drawn to the edges it was a
// full 22pt wide, bigger than the menu bar's own glyphs; at .8 it is about
// 18pt by 15pt, centred.
const trayK = 0.8

// tray draws the mark as a macOS template image (black + alpha).
func tray() image.Image {
	const S = 44
	off := S / 2 * (1 - trayK) // 44 is even; the offset stays whole
	img := image.NewNRGBA(image.Rect(0, 0, S, S))
	cov := oneColour(S, 8, trayK, off, off)
	for i, a := range cov {
		img.SetNRGBA(i%S, i/S, color.NRGBA{0, 0, 0, uint8(a*255 + 0.5)})
	}
	return img
}

// oneColour is the mark in a single ink (the tray, the Dock glyph): the
// echo's band behind, hidden by the front window, the window's band and
// the prompt unioned on top. Pieces join by max, not even-odd — the bands
// overlap and even-odd would punch holes where they cross.
func oneColour(n, ss int, k, ox, oy float64) []float64 {
	echoBand := coverage(ringRect(echoXY, echoWH, echoR, 2.6), n, ss, k, ox, oy)
	frontOuter := coverage(ringRect(frontXY, frontWH, frontR, 0), n, ss, k, ox, oy)
	pieces := [][]float64{
		coverage(ringRect(frontXY, frontWH, frontR, 3.2), n, ss, k, ox, oy),
		coverage([][][2]float64{chevron, cursor}, n, ss, k, ox, oy),
	}
	out := make([]float64, n*n)
	for i := range out {
		a := echoBand[i] * (1 - frontOuter[i])
		for _, pc := range pieces {
			if pc[i] > a {
				a = pc[i]
			}
		}
		out[i] = a
	}
	return out
}

// The tray's motion is the echo: every .9s the back window repeats itself,
// easing a step further out and brighter, then back to rest. The front
// window and its prompt hold still.
const (
	flapSecs  = 0.9
	flapEvery = 0.03 // a frame every 30ms: internal/gui's flap plays them so
)

// the echo's step out and back over the cycle
var echoKeys = []key{{0, 0}, {.5, 1}, {1, 0}}

func trayFlap(dir string, S int) error {
	n := int(math.Round(flapSecs / flapEvery))
	const ss = 8
	k := float64(S) / 44
	off := float64(S) / 2 * (1 - trayK)
	frontBand := coverage(ringRect(frontXY, frontWH, frontR, 3.2), S, ss, k, off, off)
	frontOuter := coverage(ringRect(frontXY, frontWH, frontR, 0), S, ss, k, off, off)
	prompt := coverage([][][2]float64{chevron, cursor}, S, ss, k, off, off)
	for f := 1; f < n; f++ {
		t := float64(f) / float64(n)
		d := keyed(echoKeys, t) // 0..1..0 over the cycle
		// the echo eases out along its diagonal and brightens at the top
		// of the step
		moved := make([][2]float64, len(echoRing))
		for i, p := range echoRing {
			moved[i] = [2]float64{p[0] + d*3.2, p[1] - d*3.2}
		}
		echoCov := coverage([][][2]float64{moved}, S, ss, k, off, off)
		img := image.NewNRGBA(image.Rect(0, 0, S, S))
		for i := range frontBand {
			a := echoCov[i] * (1 - frontOuter[i]) * (0.45 + 0.3*d)
			for _, pc := range [][]float64{frontBand, prompt} {
				if pc[i] > a {
					a = pc[i]
				}
			}
			img.SetNRGBA(i%S, i/S, color.NRGBA{0, 0, 0, uint8(a*255 + 0.5)})
		}
		out, err := os.Create(fmt.Sprintf("%s/flap-%02d.png", dir, f))
		if err != nil {
			return err
		}
		err = png.Encode(out, img)
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// echoRing is the echo's band flattened to one ring list, so trayFlap can
// move its points.
var echoRing = func() [][2]float64 {
	outer := rounded(echoXY, echoWH, echoR)
	inner := rounded([2]float64{echoXY[0] + 2.6, echoXY[1] + 2.6}, [2]float64{echoWH[0] - 5.2, echoWH[1] - 5.2}, echoR-2.6)
	return append(outer, inner...)
}()

// A keyframe: at is the fraction of the animation, d the distance out.
type key struct{ at, d float64 }

// keyed is the distance at t of keyframes played as CSS plays them, each
// step eased by cubic-bezier(.3, 0, .3, 1).
func keyed(ks []key, t float64) float64 {
	for i := 1; i < len(ks); i++ {
		if t <= ks[i].at {
			a, b := ks[i-1], ks[i]
			return a.d + (b.d-a.d)*ease((t-a.at)/(b.at-a.at))
		}
	}
	return ks[len(ks)-1].d
}

// ease is cubic-bezier(.3, 0, .3, 1): x found for the curve's parameter by
// bisection, and its y.
func ease(x float64) float64 {
	bez := func(u, p1, p2 float64) float64 { return 3*u*(1-u)*(1-u)*p1 + 3*u*u*(1-u)*p2 + u*u*u }
	lo, hi := 0.0, 1.0
	for range 40 {
		m := (lo + hi) / 2
		if bez(m, .3, .3) < x {
			lo = m
		} else {
			hi = m
		}
	}
	return bez((lo+hi)/2, 0, 1)
}

// glyph is the mark alone, black on clear, as big on an n-pixel canvas as
// app draws it on its tile: the layer of build/darwin/AppIcon.icon, which
// macOS 26 lays on a tile of its own, light or dark.
func glyph(n int) image.Image {
	S := float64(n)
	k := S * 0.8 / 44
	o := S/2 - 22*k
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	cov := oneColour(n, 4, k, o, o)
	for i, a := range cov {
		img.SetNRGBA(i%n, i/n, color.NRGBA{0, 0, 0, uint8(a*255 + 0.5)})
	}
	return img
}

// app draws the blue squircle with the echoed window on it: the repeat
// faintly behind, the front window white, the prompt in the deep blue.
func app(n int, dark bool) image.Image {
	S := float64(n)
	// macOS icon grid: the squircle occupies ~80% of the canvas
	inset := S * 0.1
	side := S - 2*inset
	radius := side * 0.225
	k := side * 0.8 / 44
	o := inset + side/2 - 22*k
	ss := 4
	if n <= 64 {
		ss = 8
	}
	echoCov := coverage(ringRect(echoXY, echoWH, echoR, 0), n, ss, k, o, o)
	frontCov := coverage(ringRect(frontXY, frontWH, frontR, 0), n, ss, k, o, o)
	promptCov := coverage([][][2]float64{chevron, cursor}, n, ss, k, o, o)
	// the tile, top colour shading to bottom; the prompt's ink on it
	var bg1, bg2, ink [3]float64
	bg1, bg2, ink = [3]float64{91, 156, 250}, [3]float64{47, 111, 232}, [3]float64{43, 91, 215} // #5B9CFA → #2F6FE8, #2B5BD7
	echoA, front := .32, .97
	if dark {
		// the Dock's dark appearance (macOS 26): a dark tile, as macOS
		// would otherwise darken the tile under a blue one and lose it
		bg1, bg2, ink = [3]float64{52, 55, 68}, [3]float64{30, 32, 44}, [3]float64{127, 165, 255} // #343744 → #1E202C, #7FA5FF
		echoA, front = .10, .93
	}
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			var tile float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					fx := float64(x) + (float64(sx)+0.5)/float64(ss)
					fy := float64(y) + (float64(sy)+0.5)/float64(ss)
					if roundRect(fx, fy, inset, inset, side, side, radius) {
						tile++
					}
				}
			}
			if tile == 0 {
				continue
			}
			t := float64(y) / S
			// the pieces in draw order: echo, window, prompt
			c := [3]float64{bg1[0] + (bg2[0]-bg1[0])*t, bg1[1] + (bg2[1]-bg1[1])*t, bg1[2] + (bg2[2]-bg1[2])*t}
			for _, pc := range []struct {
				cov []float64
				ink [3]float64
				a   float64
			}{
				{echoCov, [3]float64{255, 255, 255}, echoA},
				{frontCov, [3]float64{255, 255, 255}, front},
				{promptCov, ink, 1},
			} {
				a := pc.cov[y*n+x] * pc.a
				for i := range c {
					c[i] = c[i]*(1-a) + pc.ink[i]*a
				}
			}
			img.SetNRGBA(x, y, color.NRGBA{uint8(c[0] + 0.5), uint8(c[1] + 0.5), uint8(c[2] + 0.5), uint8(tile / float64(ss*ss) * 255)})
		}
	}
	return img
}

// ringRect is a window: its outer ring, or outer and inner when sw > 0,
// even-odd filling the band between them.
func ringRect(xy, wh [2]float64, r, sw float64) [][][2]float64 {
	outer := rounded(xy, wh, r)
	if sw <= 0 {
		return [][][2]float64{outer}
	}
	innerXY := [2]float64{xy[0] + sw, xy[1] + sw}
	innerWH := [2]float64{wh[0] - 2*sw, wh[1] - 2*sw}
	return [][][2]float64{outer, rounded(innerXY, innerWH, r-sw)}
}

// coverage rasterises a shape, scaled by k and moved to (ox, oy), onto an n×n
// grid of pixels with ss×ss samples each, and gives each pixel's coverage in
// 0..1. It fills even-odd, a scanline at a time.
func coverage(rings [][][2]float64, n, ss int, k, ox, oy float64) []float64 {
	cov := make([]float64, n*n)
	per := 1 / float64(ss*ss)
	var xs []float64
	for y := 0; y < n*ss; y++ {
		v := ((float64(y)+0.5)/float64(ss) - oy) / k
		xs = xs[:0]
		for _, r := range rings {
			for i, j := 0, len(r)-1; i < len(r); j, i = i, i+1 {
				a, b := r[j], r[i]
				if (a[1] > v) != (b[1] > v) {
					u := a[0] + (v-a[1])*(b[0]-a[0])/(b[1]-a[1])
					xs = append(xs, (u*k+ox)*float64(ss))
				}
			}
		}
		sort.Float64s(xs)
		row := cov[(y/ss)*n:][:n]
		for i := 0; i+1 < len(xs); i += 2 {
			// sample centres x+0.5 that fall between the two crossings
			lo := int(math.Max(0, math.Ceil(xs[i]-0.5)))
			hi := int(math.Min(float64(n*ss), math.Ceil(xs[i+1]-0.5)))
			for x := lo; x < hi; x++ {
				row[x/ss] += per
			}
		}
	}
	return cov
}

// rounded is the ring of a rounded rect, each corner's arc cut into
// six chords.
func rounded(xy, wh [2]float64, r float64) [][2]float64 {
	x, y, w, h := xy[0], xy[1], wh[0], wh[1]
	const seg = 6
	var pts [][2]float64
	corner := func(cx, cy, a0 float64) {
		for s := 0; s <= seg; s++ {
			a := a0 + (math.Pi/2)*float64(s)/seg
			pts = append(pts, [2]float64{cx + r*math.Cos(a), cy + r*math.Sin(a)})
		}
	}
	corner(x+w-r, y+r, -math.Pi/2)
	corner(x+w-r, y+h-r, 0)
	corner(x+r, y+h-r, math.Pi/2)
	corner(x+r, y+r, math.Pi)
	return pts
}

// circle is the ring of a disc, 24 sides.
func circle(cx, cy, r float64) [][2]float64 {
	var pts [][2]float64
	for s := 0; s < 24; s++ {
		a := 2 * math.Pi * float64(s) / 24
		pts = append(pts, [2]float64{cx + r*math.Cos(a), cy + r*math.Sin(a)})
	}
	return pts
}

// roundRect is a rounded square: true inside it.
func roundRect(px, py, x, y, w, h, r float64) bool {
	if px < x || py < y || px > x+w || py > y+h {
		return false
	}
	cx := math.Max(x+r, math.Min(px, x+w-r))
	cy := math.Max(y+r, math.Min(py, y+h-r))
	return math.Hypot(px-cx, py-cy) <= r
}
