//go:build !server

package opennox

import (
	"image"
	"image/color"
	"strings"

	noxcolor "github.com/opennox/libs/color"
	"github.com/opennox/libs/noximage"
	"github.com/opennox/libs/object"

	"github.com/opennox/opennox/v1/client"
	"github.com/opennox/opennox/v1/common/memmap"
	"github.com/opennox/opennox/v1/legacy"
)

const (
	configClassMarks = "game.extensions.class_marks"

	allClassesMask = 0x7 // warrior, wizard and conjurer bits, see sub_57B370

	itemCellSize  = 50
	classGemSize  = 7 // with the dark border
	classGemInset = 3

	// the inventory tray is drawn by nox_xxx_guiDrawInventoryTray_4643B0, relative to the inventory window
	invTrayX, invTrayY = 254, 13
	invTrayW, invTrayH = 260, 150
	invTrayCellsX      = 60

	shopGridOff = 1098380 // grid rect, see sub_478C80
	shopCols    = 6
	shopRows    = 10
)

var (
	classMarksEnabled = true

	// by player class: warrior, wizard, conjurer
	classNames      = [3]string{"Warrior", "Wizard", "Conjurer"}
	classGemColors  = [3]color.Color{noxcolor.RGB5551Color(210, 40, 30), noxcolor.RGB5551Color(50, 110, 250), noxcolor.RGB5551Color(50, 190, 60)}
	classGemShine   = [3]color.Color{noxcolor.RGB5551Color(255, 150, 130), noxcolor.RGB5551Color(170, 200, 255), noxcolor.RGB5551Color(170, 250, 170)}
	classGemOutline = noxcolor.RGB5551Color(16, 16, 16)
)

func init() {
	configToggle(configClassMarks, "classmarks", "mark gear with gems of the classes that can use it, and grey out gear your class cannot use", true, &classMarksEnabled)
}

// itemClasses returns a mask of player classes that can use the gear (weapon, wand or armor).
func itemClasses(dr *client.Drawable) (byte, bool) {
	if dr == nil || !dr.Class().HasAny(object.ClassWeapon|object.ClassWand|object.ClassArmor) {
		return 0, false
	}
	mask := sub_57B370(dr.Class(), dr.SubClass(), int(dr.TypeIDVal)) & allClassesMask
	return mask, mask != 0
}

// ownClassCanUse checks if the class of the local player is in the mask.
func ownClassCanUse(mask byte) bool {
	info := clientPlayerInfo()
	return info == nil || mask&(1<<info.PlayerClass()) != 0
}

// itemClassLine returns a tooltip line with the classes that can use the gear.
func itemClassLine(dr *client.Drawable) (string, bool, bool) {
	mask, ok := itemClasses(dr)
	if !ok {
		return "", false, false
	}
	if mask == allClassesMask {
		return "Usable by: all classes", true, true
	}
	var names []string
	for i, name := range classNames {
		if mask&(1<<i) != 0 {
			names = append(names, name)
		}
	}
	return "Usable by: " + strings.Join(names, ", "), ownClassCanUse(mask), true
}

// drawClassMarks marks gear in the inventory and the shop with gems of the classes that can use it,
// and greys out gear the player class cannot use.
func (c *Client) drawClassMarks() {
	// the HUD flag (nox_client_renderGUI_80828) is not checked, shops turn it off while they are open
	if !classMarksEnabled || gameGetPlayState() != 3 || legacy.Get_dword_5d4594_1319268() {
		return // the amount dialog may cover the items
	}
	// inventory, when it is open on the items tab
	if memmap.Uint8(0x5D4594, 1049868) != 0 && memmap.Uint8(0x5D4594, 1049869) == 0 {
		if win := legacy.Get_dword_5d4594_1062456(); win != nil && windowVisible(win) {
			pos := win.GlobalPos()
			clip := image.Rect(0, 0, invTrayW, invTrayH).Add(pos.Add(image.Pt(invTrayX, invTrayY)))
			x0, y0 := clip.Min.X+invTrayCellsX, clip.Min.Y-legacy.Get_dword_5d4594_1062512()
			for row := 0; row < legacy.InventoryRows; row++ {
				for col := 0; col < legacy.InventoryCols; col++ {
					if dr, _ := legacy.InventoryCell(row, col); dr != nil {
						c.drawClassMark(dr, image.Pt(x0+itemCellSize*col, y0+itemCellSize*row), clip)
					}
				}
			}
		}
	}
	// shop, when buying
	if win := legacy.Get_dword_5d4594_1098576(); win != nil && windowVisible(win) && legacy.Get_dword_5d4594_1098628() == 2 {
		clip := image.Rect(
			int(memmap.Int32(0x5D4594, shopGridOff)), int(memmap.Int32(0x5D4594, shopGridOff+4)),
			int(memmap.Int32(0x5D4594, shopGridOff+8)), int(memmap.Int32(0x5D4594, shopGridOff+12)),
		)
		y0 := clip.Min.Y - legacy.Get_dword_5d4594_1107036()
		for col := 0; col < shopCols; col++ {
			for row := 0; row < shopRows; row++ {
				off := uintptr(shopItemsOff + shopItemSize*(row+shopRows*col))
				if memmap.Uint32(0x5D4594, off+4) == 0 {
					continue
				}
				dr := legacy.AsDrawableP(*memmap.PtrPtr(0x5D4594, off))
				c.drawClassMark(dr, image.Pt(clip.Min.X+itemCellSize*col, y0+itemCellSize*row), clip)
			}
		}
	}
}

// drawClassMark draws gems of the classes in the bottom right corner of the item cell.
func (c *Client) drawClassMark(dr *client.Drawable, pos image.Point, clip image.Rectangle) {
	mask, ok := itemClasses(dr)
	if !ok {
		return
	}
	cell := image.Rect(0, 0, itemCellSize, itemCellSize).Add(pos)
	if !cell.Overlaps(clip) {
		return
	}
	if !ownClassCanUse(mask) {
		greyOutRect(c.r.PixBuffer(), cell.Intersect(clip))
	}
	if mask == allClassesMask {
		return // gems only mark gear restricted to some classes
	}
	n := 0
	for i := range classNames {
		if mask&(1<<i) != 0 {
			n++
		}
	}
	x := cell.Max.X - classGemInset - n*classGemSize
	y := cell.Max.Y - classGemInset - classGemSize
	for i := range classNames {
		if mask&(1<<i) == 0 {
			continue
		}
		if gem := image.Rect(x, y, x+classGemSize, y+classGemSize); gem.In(clip) {
			c.r.DrawRectFilledOpaque(x, y, classGemSize, classGemSize, classGemOutline)
			c.r.DrawRectFilledOpaque(x+1, y+1, classGemSize-2, classGemSize-2, classGemColors[i])
			c.r.DrawRectFilledOpaque(x+2, y+2, 2, 1, classGemShine[i])
		}
		x += classGemSize
	}
}

// greyOutRect turns the image area into darker greyscale.
func greyOutRect(pix *noximage.Image16, rect image.Rectangle) {
	if pix == nil {
		return
	}
	rect = rect.Intersect(pix.Rect)
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		row := pix.Pix[pix.PixOffset(rect.Min.X, y):][:rect.Dx()]
		for x, p := range row {
			r := uint32(p>>10) & 31
			g := uint32(p>>5) & 31
			b := uint32(p) & 31
			l := uint16((r*77 + g*150 + b*29) >> 8 * 3 / 4)
			row[x] = p&0x8000 | l<<10 | l<<5 | l
		}
	}
}
