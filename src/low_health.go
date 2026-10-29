//go:build !server

package opennox

import (
	"image"
	"math"

	"github.com/opennox/opennox/v1/legacy"
)

const (
	configLowHealthVignette = "game.extensions.low_health_vignette"

	// lowHealthStart is the health ratio when the vignette appears.
	lowHealthStart = 0.30
)

var lowHealthVignetteEnabled = true

func init() {
	configToggle(configLowHealthVignette, "vignette", "show a red vignette around the screen when health is low", true, &lowHealthVignetteEnabled)
}

// vignette is a red overlay at the screen edges.
type vignette struct {
	size image.Point
	dist []uint8 // distance from the screen center for each pixel, 255 at the corners
}

var lowHealthVignette vignette

func (v *vignette) reset(sz image.Point) {
	v.size = sz
	v.dist = make([]uint8, sz.X*sz.Y)
	cx, cy := float64(sz.X)/2, float64(sz.Y)/2
	for y := 0; y < sz.Y; y++ {
		dy := (float64(y) + 0.5 - cy) / cy
		for x := 0; x < sz.X; x++ {
			dx := (float64(x) + 0.5 - cx) / cx
			d := math.Sqrt((dx*dx + dy*dy) / 2)
			v.dist[y*sz.X+x] = uint8(min(d, 1) * 255)
		}
	}
}

// drawLowHealthVignette draws a red vignette on the world image when the player health is low.
// It reaches further inside, gets darker and stronger when the health drops.
func (c *Client) drawLowHealthVignette() {
	if !lowHealthVignetteEnabled || gameGetPlayState() != 3 {
		return
	}
	_, cur, maxv := legacy.HealthManaBar(0)
	if maxv <= 0 {
		return
	}
	ratio := float64(cur) / float64(maxv)
	if ratio >= lowHealthStart {
		return
	}
	img := noxPixBuffer.img
	if img == nil {
		return
	}
	v := &lowHealthVignette
	sz := img.Rect.Size()
	if v.size != sz {
		v.reset(sz)
	}
	t := (lowHealthStart - max(ratio, 0)) / lowHealthStart // 0 when it appears, 1 at zero health
	inner := 0.6 - 0.35*t                                  // distance where the vignette starts
	strength := 0.3 + 0.55*t                               // opacity at the corners
	var alpha [256]int32
	for i := range alpha {
		d := float64(i) / 255
		if d <= inner {
			continue
		}
		k := (d - inner) / (1 - inner)
		alpha[i] = int32(strength * k * k * 256)
	}
	// dark red, in 5 bit components
	const tr, tg, tb = 18, 0, 0
	for y := 0; y < sz.Y; y++ {
		row := img.Row(y)[:sz.X]
		dist := v.dist[y*sz.X : (y+1)*sz.X]
		for x, d := range dist {
			a := alpha[d]
			if a == 0 {
				continue
			}
			p := row[x]
			r := int32(p>>10) & 31
			g := int32(p>>5) & 31
			b := int32(p) & 31
			r += (tr - r) * a >> 8
			g += (tg - g) * a >> 8
			b += (tb - b) * a >> 8
			row[x] = p&0x8000 | uint16(r)<<10 | uint16(g)<<5 | uint16(b)
		}
	}
}
