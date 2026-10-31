package opennox

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
	"regexp"
	"strconv"
	"strings"

	noxcolor "github.com/opennox/libs/color"
	"github.com/opennox/libs/console"
	"github.com/opennox/libs/object"
	"github.com/spf13/viper"

	"github.com/opennox/opennox/v1/client"
	noxflags "github.com/opennox/opennox/v1/common/flags"
	"github.com/opennox/opennox/v1/common/memmap"
	"github.com/opennox/opennox/v1/legacy"
)

const (
	configItemTooltip = "game.extensions.item_tooltip"

	itemTooltipMaxWidth = 240
	itemTooltipPad      = 3
)

var (
	itemTooltipEnabled = true

	itemTooltipNameColor   = nox_color_yellow_2589772
	itemTooltipTextColor   = nox_color_white_2523948
	itemTooltipBetterColor = noxcolor.RGB5551Color(120, 230, 120)
	itemTooltipWorseColor  = noxcolor.RGB5551Color(240, 100, 100)
	itemTooltipGoldColor   = noxcolor.RGB5551Color(230, 200, 90)

	itemTooltipNumber = regexp.MustCompile(`-?[0-9]+(\.[0-9]+)?`)
)

func init() {
	configBoolPtr(configItemTooltip, "", true, &itemTooltipEnabled)
	noxConsole.Register(&console.Command{
		Token: "itemtooltip",
		Help:  "itemtooltip [on|off]: show item stats in the tooltip and compare them with equipped items",
		Flags: console.ClientServer,
		Func: func(ctx context.Context, c *console.Console, tokens []string) bool {
			if len(tokens) > 1 {
				return false
			} else if len(tokens) == 1 {
				switch tokens[0] {
				case "on", "1":
					itemTooltipEnabled = true
				case "off", "0":
					itemTooltipEnabled = false
				default:
					return false
				}
				viper.Set(configItemTooltip, itemTooltipEnabled)
				writeConfigLater()
			}
			state := "off"
			if itemTooltipEnabled {
				state = "on"
			}
			c.Printf(console.ColorLightYellow, "itemtooltip = %s", state)
			return true
		},
	})
}

// tooltipItem returns an inventory item which stats the cursor tooltip should show.
func (c *Client) tooltipItem() *client.Drawable {
	if !itemTooltipEnabled {
		return nil
	}
	dr := legacy.TooltipItem()
	if !legacy.IsInventoryItem(dr) && !isShopItem(dr) {
		return nil
	}
	return dr
}

const (
	shopItemsOff  = 1098636 // first shop slot, see sub_478E50
	shopItemSize  = 140
	shopItemsSize = 6 * 10
)

// isShopItem checks if the item is offered by the shop.
func isShopItem(dr *client.Drawable) bool {
	if dr == nil {
		return false
	}
	for i := 0; i < shopItemsSize; i++ {
		off := uintptr(shopItemsOff + shopItemSize*i)
		if memmap.Uint32(0x5D4594, off+4) != 0 && *memmap.PtrPtr(0x5D4594, off) == dr.C() {
			return true
		}
	}
	return false
}

// itemSlotGroup returns a group of items that are equipped into the same slot.
func itemSlotGroup(dr *client.Drawable) (object.Class, object.SubClass) {
	cl := dr.Class()
	switch {
	case cl.HasAny(object.ClassWeapon | object.ClassWand):
		const ammo = object.SubClass(object.WeaponQuiver | object.WeaponArrow | object.WeaponBolt)
		if cl.Has(object.ClassWeapon) && dr.SubClass()&ammo != 0 {
			return object.ClassWeapon, ammo
		}
		return object.ClassWeapon | object.ClassWand, ^ammo
	case cl.Has(object.ClassArmor):
		return object.ClassArmor, dr.SubClass()
	}
	return 0, 0
}

// equippedCounterpart returns an equipped item that the given item would replace.
func equippedCounterpart(dr *client.Drawable) *client.Drawable {
	cl, sub := itemSlotGroup(dr)
	if cl == 0 || sub == 0 {
		return nil
	}
	var found *client.Drawable
	for _, it := range legacy.EquippedItems() {
		if it == dr {
			return nil // the item is equipped already
		}
		if found == nil && it.Class()&cl != 0 && it.SubClass()&sub != 0 {
			found = it
		}
	}
	return found
}

type itemTooltipLine struct {
	text string
	diff string
	cl   color.Color
}

// itemLineLabel returns the part of the info line before the first number, which names the stat.
func itemLineLabel(s string) (string, float64, bool) {
	loc := itemTooltipNumber.FindStringIndex(s)
	if loc == nil {
		return "", 0, false
	}
	v, err := strconv.ParseFloat(s[loc[0]:loc[1]], 64)
	if err != nil {
		return "", 0, false
	}
	return strings.TrimSpace(s[:loc[0]]), v, true
}

func (c *Client) itemTooltipLines(name string, dr *client.Drawable) []itemTooltipLine {
	lines := []itemTooltipLine{{text: name, cl: itemTooltipNameColor}}
	info := legacy.ItemInfoLines(dr)
	var other map[string]float64
	if eq := equippedCounterpart(dr); eq != nil {
		other = make(map[string]float64)
		for _, s := range legacy.ItemInfoLines(eq) {
			if label, v, ok := itemLineLabel(s); ok {
				if _, dup := other[label]; !dup {
					other[label] = v
				}
			}
		}
	}
	weight := c.Strings().GetStringInFile("IdentifyWeight", "C:\\NoxPost\\src\\Client\\Gui\\guiinv.c")
	if i := strings.IndexByte(weight, '%'); i >= 0 {
		weight = weight[:i]
	}
	weight = strings.TrimSpace(weight)
	for _, s := range info {
		if strings.TrimSpace(s) == "" {
			continue
		}
		line := itemTooltipLine{text: s, cl: itemTooltipTextColor}
		if label, v, ok := itemLineLabel(s); ok && other != nil {
			if ov, ok := other[label]; ok && label != "" && math.Abs(v-ov) > 1e-6 {
				d := v - ov
				if d == math.Trunc(d) {
					line.diff = fmt.Sprintf("(%+d)", int(d))
				} else {
					line.diff = fmt.Sprintf("(%+.1f)", d)
				}
				better := d > 0
				if label == weight {
					better = !better
				}
				if better {
					line.cl = itemTooltipBetterColor
				} else {
					line.cl = itemTooltipWorseColor
				}
			}
		}
		lines = append(lines, line)
	}
	if !isShopItem(dr) {
		if s := c.itemSellLine(dr); s != "" {
			lines = append(lines, itemTooltipLine{text: s, cl: itemTooltipGoldColor})
		}
	}
	return lines
}

// itemSellLine returns the price the shop pays for the inventory item, or its full value when not trading.
// The price is only known when the game is hosted locally.
func (c *Client) itemSellLine(dr *client.Drawable) string {
	if !noxflags.HasGame(noxflags.GameHost) {
		return ""
	}
	u := c.srv.Players.HostUnit()
	if u == nil {
		return ""
	}
	for _, it := range u.Inventory() {
		if it.NetCode != dr.NetCode32 {
			continue
		}
		if sess := u.UpdateDataPlayer().Trade70; sess != nil {
			return fmt.Sprintf("Sell price: %d", legacy.ShopSellPrice(sess, it))
		}
		return fmt.Sprintf("Value: %d", legacy.ShopSellPrice(nil, it))
	}
	return ""
}

// drawItemTooltip draws a tooltip with item stats, compared with the equipped item.
func (c *Client) drawItemTooltip(mpos image.Point, name string, dr *client.Drawable) {
	lines := c.itemTooltipLines(name, dr)
	r := c.r
	spaceW := r.GetStringSizeWrapped(nil, " ", 0).X
	w, h := 0, 0
	sizes := make([]image.Point, len(lines))
	for i, l := range lines {
		sz := r.GetStringSizeWrapped(nil, l.text, itemTooltipMaxWidth)
		if l.diff != "" {
			sz.X += spaceW + r.GetStringSizeWrapped(nil, l.diff, 0).X
		}
		sizes[i] = sz
		w = max(w, sz.X)
		h += sz.Y
	}
	w += 2 * itemTooltipPad
	h += 2 * itemTooltipPad
	px := mpos.X - c.pos1097204.X
	py := mpos.Y - c.pos1097204.Y
	if px+w >= nox_win_width {
		px -= w
	}
	if px < 0 {
		px = 0
	}
	if py+h >= nox_win_height {
		py = nox_win_height - h
	}
	if py < 0 {
		py = 0
	}
	r.DrawRectFilledAlpha(px, py, w, h)
	y := py + itemTooltipPad
	x := px + itemTooltipPad
	for i, l := range lines {
		r.Data().SetTextColor(itemTooltipTextColor)
		if l.diff == "" {
			r.Data().SetTextColor(l.cl)
		}
		r.DrawStringWrapped(nil, l.text, image.Rect(x, y, x+itemTooltipMaxWidth, y))
		if l.diff != "" {
			dx := r.GetStringSizeWrapped(nil, l.text, itemTooltipMaxWidth).X + spaceW
			r.Data().SetTextColor(l.cl)
			r.DrawStringWrapped(nil, l.diff, image.Rect(x+dx, y, x+dx, y))
		}
		y += sizes[i].Y
	}
}
