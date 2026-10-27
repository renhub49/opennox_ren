//go:build !server

package opennox

import (
	"errors"
	"image"
	"image/color"
	"slices"

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
