//go:build !server

package opennox

import (
	"errors"
	"image"
	"image/color"
	"slices"

	"github.com/opennox/libs/client/seat"
	"github.com/opennox/libs/ifs"
	"github.com/opennox/libs/noximage"

	"github.com/opennox/opennox/v1/client/gui"
	"github.com/opennox/opennox/v1/client/noxmovie"
	"github.com/opennox/opennox/v1/common/memmap"
	"github.com/opennox/opennox/v1/legacy"
)

func (c *Client) DrawGeneral(a1 bool) error {
	dword_5d4594_1311936 = true
	*memmap.PtrUint32(0x5D4594, 1311932) = uint32(bool2int(a1))
	// FIXME
	v1 := false
	videoLog.Println("DrawGeneralStart")
	if /*noxflags.HasEngine(noxflags.EngineWindowed) ||*/ v1 /*|| nox_video_renderTargetFlags&0x10 != 0*/ {
		videoLog.Println("DrawGeneralSkip")
		sub_4B05D0()
		return nil
	}
	legacy.Sub_431290()
	legacy.MusicModule.Sub_43DBD0()
	legacy.Dialogs.Sub_44D8F0()
	for legacy.MusicModule.Sub_43DC40() || legacy.Dialogs.Sub_44D930() {
		c.sub4312C0()
	}
	v12 := sub_48B3E0(false)
	//inpHandler.UnacquireMouse()

	c.playMovieFile(movieFilesStack[0])

	legacy.MusicModule.Sub_43DBE0()
	//inpHandler.AcquireMouse()
	sub_48B3E0(v12)
	sub_4B05D0()
	return nil
}

func (c *Client) playMovieFile(name string) {
	videoLog.Printf("playMovieFile: %q", name)
	if f, err := ifs.Open(name); err == nil {
		defer f.Close()
		if plr, err := noxmovie.NewPlayerWithHandle(f, noxClient.Seat, audioDev); err == nil {
			defer plr.Close()
			plr.SetAudioGain(configGetVolumeGain(VolumeMusic))
			plr.Start()
			plr.Play()
		}
	}
}

func (c *Client) clientDraw() bool {
	return c.nox_xxx_client_435F80_draw()
}

func (c *Client) copyPixBuffer() {
	c.Win.CopyBuffer(noxPixBuffer.img)
	*memmap.PtrUint32(0x973A20, 496)++
}

func (c *Client) resetRenderer(sz image.Point, init bool) error {
	if nox_video_renderTargetFlags&4 == 0 && !init {
		if err := c.Win.Reset(sz); err != nil {
			return err
		}
	}
	if nox_video_setBackBufSizes_48A3D0(sz) == 0 {
		return errors.New("nox_video_setBackBufSizes_48A3D0 failed")
	}
	return nil
}

func (c *Client) gameResetVideoMode(inMenu, force bool) error {
	var mode image.Point
	if inMenu {
		if debugMainloop {
			videoLog.Printf("gameUpdateVideoMode: menu (%s)", caller(1))
		}
		mode = videoGetMenuMode()
	} else {
		if debugMainloop {
			videoLog.Printf("gameUpdateVideoMode: game (%s)", caller(1))
		}
		mode = c.videoGetGameMode()
	}
	videoLog.Printf("mode switch: %+v (menu: %v)", mode, inMenu)
	videoResizeView(mode)
	if !force && mode == noxPixBuffer.img.Size() {
		if !inMenu {
			applyGUIScale()
		}
		return nil
	}
	if err := recreateBuffersAndTarget(mode); err != nil {
		return err
	}
	c.nox_xxx_loadPal_4A96C0_video_read_palette()
	legacy.Sub_461520()
	if !inMenu {
		applyGUIScale()
	}
	return nil
}

func init() {
	guiScaleChanged = func() {
		guiScalePending = true
	}
}

// guiScalePending is set when the GUI scale setting changes, but it's not applied yet.
var guiScalePending bool

// applyPendingGUIScale applies a changed GUI scale after the mouse button is released.
// The GUI moves under the mouse when the scale changes, which would confuse sliders that are dragged.
func (c *Client) applyPendingGUIScale() {
	if !guiScalePending || c.Inp == nil || c.Inp.IsMousePressed(seat.MouseButtonLeft) {
		return
	}
	guiScalePending = false
	applyGUIScaleLive()
}

// applyGUIScaleLive applies a new GUI scale while in the game, by moving the GUI to the new screen size.
func applyGUIScaleLive() {
	if !noxScreen.game {
		return // applied when the game starts
	}
	world := noxScreen.world
	old := screenSizeFor(world, noxScreen.gui)
	noxScreen.gui = guiScaleFor(world, noxGUIScale)
	cur := screenSizeFor(world, noxScreen.gui)
	setScreenScale(noxScreen.gui)
	if old == cur {
		return
	}
	noxClient.GUI.Relayout(old, cur)
	relayoutGUIValues(old, cur)
	// the quit menu is not anchored to an edge
	if win := legacy.Get_nox_wnd_quitMenu_825760(); win != nil {
		setQuitMenuPos(win)
	}
}

// guiScreenRects are offsets of screen bounds, stored by GUI elements when they are created.
var guiScreenRects = []uintptr{1049740, 1091916, 1098500, 1319116, 1320196}

// relayoutGUIValues updates positions, which GUI elements compute from the screen size when they are created.
// Each value is updated only if it still has the value computed for the old screen size.
func relayoutGUIValues(old, cur image.Point) {
	upd := func(p *int32, was, now int) {
		if int(*p) == was {
			*p = int32(now)
		}
	}
	updVar := func(get func() int, set func(int), was, now int) {
		if get() == was {
			set(now)
		}
	}
	for _, off := range guiScreenRects {
		upd(memmap.PtrInt32(0x5D4594, off), old.X, cur.X)
		upd(memmap.PtrInt32(0x5D4594, off+4), old.Y, cur.Y)
		upd(memmap.PtrInt32(0x5D4594, off+24), old.X, cur.X)
		upd(memmap.PtrInt32(0x5D4594, off+28), old.Y, cur.Y)
	}
	upd(memmap.PtrInt32(0x5D4594, 831284), (old.X-noxDefaultWidth)/2, (cur.X-noxDefaultWidth)/2)
	upd(memmap.PtrInt32(0x5D4594, 831288), (old.Y-noxDefaultHeight)/2, (cur.Y-noxDefaultHeight)/2)
	upd(memmap.PtrInt32(0x5D4594, 1064876), old.X/2, cur.X/2)
	upd(memmap.PtrInt32(0x5D4594, 1064880), 2*old.Y/3, 2*cur.Y/3)
	upd(memmap.PtrInt32(0x587000, 133668), old.Y, cur.Y)
	// spell bar
	updVar(legacy.Get_dword_5d4594_1047548, legacy.Set_dword_5d4594_1047548, (old.X-320)/2, (cur.X-320)/2)
	updVar(legacy.Get_dword_5d4594_1047552, legacy.Set_dword_5d4594_1047552, old.Y-74, cur.Y-74)
	switch legacy.Get_dword_5d4594_1049536() {
	case old.Y - 74: // shown
		legacy.Set_dword_5d4594_1049536(cur.Y - 74)
	case old.Y + 1: // hidden
		legacy.Set_dword_5d4594_1049536(cur.Y + 1)
	}
	// summon creature panel
	updVar(legacy.Get_dword_5d4594_1320988, legacy.Set_dword_5d4594_1320988, old.X-95, cur.X-95)
}

// guiLayer is an offscreen buffer for drawing a part of the UI in a smaller screen space.
// It is upscaled onto the world buffer after drawing.
type guiLayer struct {
	img   *noximage.Image16
	base  []uint16 // layer contents before drawing, used to find changed pixels
	world image.Point
	xs    []int // layer column for each world column
	wxs   []int // world column for each layer column
	rows  []bool
}

var (
	guiLayerMain   guiLayer
	guiLayerCursor guiLayer
)

func (l *guiLayer) reset(world, sz image.Point) {
	if l.img == nil || l.img.Rect.Size() != sz {
		l.img = noximage.NewImage16(image.Rectangle{Max: sz})
		l.base = make([]uint16, len(l.img.Pix))
		l.rows = make([]bool, sz.Y)
		l.world = image.Point{}
	}
	if l.world != world {
		l.world = world
		l.xs = make([]int, world.X)
		for x := range l.xs {
			l.xs[x] = x * sz.X / world.X
		}
		l.wxs = make([]int, sz.X)
		for x := range l.wxs {
			l.wxs[x] = x * world.X / sz.X
		}
	}
}

// downsample fills the layer with the world image, so that translucent GUI elements are blended correctly.
func (l *guiLayer) downsample(world *noximage.Image16) {
	sz := l.img.Rect.Size()
	wsz := world.Rect.Size()
	for y := 0; y < sz.Y; y++ {
		src := world.Row(y * wsz.Y / sz.Y)
		dst := l.row(y)
		for x, wx := range l.wxs {
			dst[x] = src[wx]
		}
	}
	copy(l.base, l.img.Pix)
}

func (l *guiLayer) row(y int) []uint16 {
	off := y * l.img.Stride
	return l.img.Pix[off : off+l.img.Rect.Dx()]
}

func (l *guiLayer) baseRow(y int) []uint16 {
	off := y * l.img.Stride
	return l.base[off : off+l.img.Rect.Dx()]
}

// upscale copies pixels changed by drawing on the layer back to the world image.
func (l *guiLayer) upscale(world *noximage.Image16) {
	sz := l.img.Rect.Size()
	wsz := world.Rect.Size()
	for y := 0; y < sz.Y; y++ {
		l.rows[y] = !slices.Equal(l.row(y), l.baseRow(y))
	}
	for y := 0; y < wsz.Y; y++ {
		ly := y * sz.Y / wsz.Y
		if !l.rows[ly] {
			continue
		}
		src := l.row(ly)
		base := l.baseRow(ly)
		dst := world.Row(y)
		for x, lx := range l.xs {
			if v := src[lx]; v != base[lx] {
				dst[x] = v
			}
		}
	}
}

func scaleRect(r image.Rectangle, from, to image.Point) image.Rectangle {
	return image.Rectangle{
		Min: image.Point{X: r.Min.X * to.X / from.X, Y: r.Min.Y * to.Y / from.Y},
		Max: image.Point{X: r.Max.X * to.X / from.X, Y: r.Max.Y * to.Y / from.Y},
	}
}

// drawScaled runs the draw function in a screen space with a given scale.
// If the scale is larger than 1, drawing goes to an offscreen layer which is then upscaled onto the world buffer.
func (c *Client) drawScaled(l *guiLayer, s float32, fnc func()) {
	world := noxPixBuffer.img
	if s <= 1 || world == nil {
		withScreenScale(1, fnc)
		return
	}
	wsz := world.Rect.Size()
	sz := screenSizeFor(wsz, s)
	l.reset(wsz, sz)
	l.downsample(world)

	d := c.r.Data()
	clip, clip2, rect3 := d.ClipRect(), d.ClipRect2(), d.Rect3()
	lrect := image.Rectangle{Max: sz}
	d.SetRect3(lrect)
	d.SetClipRect(scaleRect(clip, wsz, sz).Intersect(lrect))
	d.SetClipRect2(scaleRect(clip2, wsz, sz).Intersect(lrect))
	c.r.SetPixBuffer(l.img)

	withScreenScale(s, fnc)

	c.r.SetPixBuffer(world)
	d.SetClipRect(clip)
	d.SetClipRect2(clip2)
	d.SetRect3(rect3)

	l.upscale(world)
}

// drawGUI draws GUI elements using the GUI scale.
func (c *Client) drawGUI(fnc func()) {
	c.drawScaled(&guiLayerMain, noxScreen.gui, fnc)
}

// drawCursor draws the cursor and tooltips using the cursor scale.
func (c *Client) drawCursor(fnc func()) {
	c.drawScaled(&guiLayerCursor, guiScaleFor(videoGetWorldSize(), noxCursorScale), fnc)
}

func (c *Client) nox_xxx_loadPal_4A96C0_video_read_palette() {
	nox_xxx_loadDefColor_4A94A0()
	for i := 0; i < 6; i++ {
		c.r.Data().SetMaterial(i, color.White)
	}
}

func (c *Client) sub_43C060() bool {
	c.nox_xxx_loadPal_4A96C0_video_read_palette()
	c.guiLoadBorderImages()
	c.nox_xxx_wndLoadMainBG_4A2210()
	c.Nox_client_setCursorType(gui.CursorSelect)
	sub_48B3E0(true)
	c.Nox_video_stopAllFades44E040()
	legacy.Sub_43E8C0(0)
	return legacy.Nox_xxx_compassGenStrings_4A9C80() != 0
}

func nox_video_setBackBufSizes_48A3D0(sz image.Point) int {
	*memmap.PtrUint32(0x973F18, 2368) = 0

	*memmap.PtrUint32(0x973F18, 136) = uint32(2 * sz.X)
	*memmap.PtrUint32(0x973F18, 2368) = 1

	*memmap.PtrUint32(0x973F18, 2348) = uint32(sz.X / 2)
	return 1
}
