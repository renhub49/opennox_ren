//go:build !server

package opennox

import (
	"fmt"
	"image"
	"image/color"
	"strconv"

	"github.com/opennox/libs/noxfont"

	noxflags "github.com/opennox/opennox/v1/common/flags"
	"github.com/opennox/opennox/v1/common/memmap"
	"github.com/opennox/opennox/v1/legacy"
	"github.com/opennox/opennox/v1/server"
)

const (
	configEffectTimers = "game.extensions.effect_timers"

	// effect icons are drawn by nox_xxx_inventoryDrawAllMB_463430, relative to the inventory window
	effectIconsOff  = 1062540 // mask of the effects shown
	effectIconsX    = 10
	effectIconsY    = 234
	effectIconStep  = 35
	effectIconSize  = 30
	effectWarnSecs  = 5 // the timer flashes when less seconds are left
	effectTimerFont = noxfont.SmallName
)

var effectTimersEnabled = true

func init() {
	configToggle(configEffectTimers, "effecttimers", "show the time left under the active effect icons", true, &effectTimersEnabled)
}

// formatEffectTime formats the time left as seconds, or minutes and seconds.
func formatEffectTime(secs int) string {
	if secs >= 60 {
		return fmt.Sprintf("%d:%02d", secs/60, secs%60)
	}
	return strconv.Itoa(secs)
}

// drawEffectTimers draws the time left under the effect icons drawn by the game.
// The time is only known when the game is hosted locally.
func (c *Client) drawEffectTimers() {
	if !effectTimersEnabled || gameGetPlayState() != 3 || !nox_client_renderGUI_80828 {
		return
	}
	mask := memmap.Uint32(0x5D4594, effectIconsOff)
	if mask == 0 {
		return
	}
	win := legacy.Get_dword_5d4594_1062456()
	if win == nil || !windowVisible(win) {
		return
	}
	var u *server.Object
	if noxflags.HasGame(noxflags.GameHost) {
		u = c.srv.Players.HostUnit()
	}
	fps := int(c.srv.TickRate())
	r := c.r
	fnt := r.Fonts.FontByName(effectTimerFont)
	pos := win.GlobalPos()
	x, y := pos.X+effectIconsX, pos.Y+effectIconsY
	for id := server.EnchantID(0); id < 30; id++ { // same as the game, 30 and 31 have no icons
		if mask&(1<<id) == 0 {
			continue
		}
		if u != nil && fps > 0 {
			if dur := u.EnchantDur(id); dur > 0 {
				secs := (dur + fps - 1) / fps
				var cl color.Color = nox_color_white_2523948
				if secs <= effectWarnSecs && (dur/(fps/4+1))%2 == 0 {
					cl = nox_color_yellow_2589772
				}
				s := formatEffectTime(secs)
				tx := x + (effectIconSize-r.GetStringSizeWrapped(fnt, s, 0).X)/2
				ty := y + effectIconSize + 1
				r.Data().SetTextColor(nox_color_black_2650656)
				r.DrawStringWrapped(fnt, s, image.Rect(tx+1, ty+1, tx+1, ty+1))
				r.Data().SetTextColor(cl)
				r.DrawStringWrapped(fnt, s, image.Rect(tx, ty, tx, ty))
			}
		}
		x += effectIconStep
	}
}
