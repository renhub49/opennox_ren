//go:build !server

package opennox

import (
	"image"
	"image/color"
	"strconv"
	"unsafe"

	noxcolor "github.com/opennox/libs/color"
	"github.com/opennox/libs/noxfont"
	"github.com/opennox/libs/player"
	"github.com/opennox/libs/spell"

	"github.com/opennox/opennox/v1/common/memmap"
	"github.com/opennox/opennox/v1/legacy"
)

const (
	configSpellCosts = "game.extensions.spell_costs"

	spellSlotSize = 30 // see nox_xxx_quickBarWarriorDraw_45FDE0
	trapBarOff    = 1047940
)

var (
	spellCostsEnabled = true

	spellCostColor = noxcolor.RGB5551Color(200, 200, 200)
)

func init() {
	configToggle(configSpellCosts, "spellcosts", "show the mana cost of spells on the spell bar", true, &spellCostsEnabled)
}

// drawSpellBarCosts draws the mana cost in the corner of each spell bar slot, red when there is not enough mana.
func (c *Client) drawSpellBarCosts() {
	if !spellCostsEnabled || gameGetPlayState() != 3 || !nox_client_renderGUI_80828 {
		return
	}
	if info := clientPlayerInfo(); info == nil || info.PlayerClass() == player.Warrior {
		return // the warrior bar has abilities instead of spells
	}
	_, mana, _ := legacy.HealthManaBar(1)
	c.drawSpellBarPageCosts(legacy.Get_nox_xxx_aClosewoodengat_587000_133480(), 5, mana, false)
	c.drawSpellBarPageCosts(memmap.PtrOff(0x5D4594, trapBarOff), 3, mana, true)
}

// drawSpellBarPageCosts draws costs for one spell bar page, laid out as in nox_xxx_quickBarCreate_45E190.
// A trap uses mana for all of its spells, so each trap slot also needs mana for the slots before it.
func (c *Client) drawSpellBarPageCosts(bar unsafe.Pointer, n int, mana int, trap bool) {
	if bar == nil {
		return
	}
	slots := *(*unsafe.Pointer)(unsafe.Add(bar, 204))
	if slots == nil {
		return
	}
	r := c.r
	fnt := r.Fonts.FontByName(noxfont.NumbersName)
	fh := r.FontHeight(fnt)
	for i := 0; i < n; i++ {
		id := spell.ID(*(*int32)(unsafe.Add(slots, 8*i)))
		if trap && i > 0 {
			if prev := spell.ID(*(*int32)(unsafe.Add(slots, 8*(i-1)))); prev != 0 {
				mana -= c.srv.Spells.ManaCost(prev, 2)
			}
		}
		if id == 0 {
			continue
		}
		win := legacy.AsWindowP(*(*unsafe.Pointer)(unsafe.Add(bar, 212+4*i)))
		if win == nil || !windowVisible(win) {
			continue
		}
		cost := c.srv.Spells.ManaCost(id, 1)
		if cost <= 0 {
			continue
		}
		var cl color.Color = spellCostColor
		if mana < cost {
			cl = nox_color_red
		}
		s := strconv.Itoa(cost)
		pos := win.GlobalPos()
		x := pos.X + spellSlotSize - 1 - r.GetStringSizeWrapped(fnt, s, 0).X
		y := pos.Y + spellSlotSize - fh
		r.Data().SetTextColor(cl)
		r.DrawStringWrapped(fnt, s, image.Rect(x, y, x, y))
	}
}
