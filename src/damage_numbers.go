//go:build !server

package opennox

import (
	"image"
	"image/color"
	"strconv"

	"github.com/opennox/libs/noxfont"

	"github.com/opennox/opennox/v1/client"
	"github.com/opennox/opennox/v1/legacy"
)

const (
	configDamageNumbers = "game.extensions.damage_numbers"

	damageNumberTime = 30 // in input ticks, same as in sub_49A6A0
	damageNumberRise = 2  // per input tick, in world pixels
)

var damageNumbersEnabled = true

func init() {
	configToggle(configDamageNumbers, "damagenumbers", "show bigger damage numbers, scaled with the GUI", true, &damageNumbersEnabled)
}

// damageNumbers draws damage and heal numbers of units on the GUI layer, instead of the small ones drawn by sub_49A6A0.
type damageNumbers struct {
	units []*client.Drawable // units drawn in the current frame
}

var noxDamageNumbers damageNumbers

// beginFrame must be called before the world is drawn.
func (d *damageNumbers) beginFrame() {
	d.units = d.units[:0]
	legacy.SetDamageNumbersHidden(damageNumbersEnabled)
}

// addUnit remembers a unit drawn in this frame, so its numbers can be drawn later.
func (d *damageNumbers) addUnit(dr *client.Drawable) {
	if damageNumbersEnabled {
		d.units = append(d.units, dr)
	}
}

// draw draws the numbers. It must be called in the GUI screen space.
func (d *damageNumbers) draw(c *Client) {
	if !damageNumbersEnabled || len(d.units) == 0 {
		return
	}
	changes := legacy.HealthChanges()
	if len(changes) == 0 {
		return
	}
	r := c.r
	vp := c.Viewport()
	world, scr := videoGetWorldSize(), videoGetWindowSize()
	fnt := r.Fonts.FontByName(noxfont.LargeName)
	seq := c.GetInputSeq()
	player := c.ClientPlayerUnit()
	for _, ch := range changes {
		age := int(seq - ch.Seq)
		if age < 0 || age > damageNumberTime {
			continue
		}
		for _, dr := range d.units {
			if dr.NetCode32 != ch.NetCode {
				continue
			}
			var cl color.Color = nox_color_yellow_2589772
			if ch.Value > 0 {
				cl = nox_color_green
			} else if dr == player {
				cl = nox_color_red
			}
			text := strconv.Itoa(max(ch.Value, -ch.Value))
			// same position as in sub_49A6A0, converted to the GUI screen space
			sx := vp.Screen.Min.X + dr.PosVec.X - vp.World.Min.X
			sy := vp.Screen.Min.Y + dr.PosVec.Y - vp.World.Min.Y - age*damageNumberRise - int(dr.ZVal) - int(dr.ZSizeMax)
			x := sx*scr.X/world.X - r.GetStringSizeWrapped(fnt, text, 0).X/2
			y := sy * scr.Y / world.Y
			r.Data().SetTextColor(nox_color_black_2650656)
			for _, o := range [...]image.Point{{X: -1, Y: -1}, {X: -1, Y: 1}, {X: 1, Y: -1}, {X: 1, Y: 1}} {
				r.DrawStringWrapped(fnt, text, image.Rect(x+o.X, y+o.Y, x+o.X, y+o.Y))
			}
			r.Data().SetTextColor(cl)
			r.DrawStringWrapped(fnt, text, image.Rect(x, y, x, y))
			break
		}
	}
}
