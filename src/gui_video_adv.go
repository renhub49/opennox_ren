package opennox

import (
	"fmt"
	"image/color"
	"math"

	noxcolor "github.com/opennox/libs/color"
	"github.com/opennox/libs/noxfont"
	"github.com/opennox/libs/strman"

	"github.com/opennox/opennox/v1/client/gui"
	noxflags "github.com/opennox/opennox/v1/common/flags"
	"github.com/opennox/opennox/v1/common/sound"
	"github.com/opennox/opennox/v1/legacy"
)

type videoOpt struct {
	ID     uint
	Flag   noxflags.EngineFlag
	Bool   *bool
	CFlag  *uint32
	TextID strman.ID
	Text   string
	Get    func() bool
	Toggle func()
	Def    bool
	Hidden bool
}

// videoScaleOpt is a slider for one of the GUI scale settings.
type videoScaleOpt struct {
	TextID   uint
	SliderID uint
	Text     string
	Tooltip  string
	Ptr      *float32
	Key      string
}

const (
	// scale sliders work in percent above the minimal scale,
	// because the slider widget reports values without adding its min
	videoScaleSliderMax = 100 * (guiScaleMax - guiScaleMin)
	// scale sliders snap to this step
	videoScaleStep = 0.25
)

func videoScaleFromSlider(v int) float32 {
	return guiScaleMin + float32(math.Round(float64(v)/100/videoScaleStep)*videoScaleStep)
}

func videoScaleToSlider(v float32) int {
	return int(math.Round(float64(v-guiScaleMin) * 100))
}

func (opt *videoScaleOpt) label() string {
	return fmt.Sprintf("%s %.2f", opt.Text, *opt.Ptr)
}

type guiAdvOptions struct {
	cli                          *Client
	nox_win_advVideoOpts_1522600 *gui.Window
	noxVideoAdvList              []*videoOpt
	noxVideoAdvOpts              map[uint]*videoOpt
	noxVideoScaleOpts            []*videoScaleOpt
}

func (c *guiAdvOptions) Init(cli *Client) {
	c.cli = cli
	c.noxVideoAdvOpts = make(map[uint]*videoOpt)
	c.noxVideoAdvList = []*videoOpt{
		{ID: 2051, Get: c.cli.GetFiltering, Toggle: c.cli.ToggleFiltering, Text: "Smooth image", TextID: "AdVidOpt.wnd:Filtering"},
		{ID: 2050, Get: c.cli.GetStretch, Toggle: c.cli.ToggleStretch, Text: "Stretch image", TextID: "AdVidOpt.wnd:Stretched"},
		{ID: 2010, Bool: &legacyUnlockSurface, TextID: "AdVidOpt.wnd:ClipWalls", Def: true, Hidden: true},
		{ID: 2012, Flag: noxflags.EngineSoftShadowEdge, TextID: "AdVidOpt.wnd:GouradShading"},
		{ID: 2014, Bool: &guiCon.translucent, TextID: "AdVidOpt.wnd:TranslucentConsole"},
		{ID: 2015, Bool: &c.cli.r.Part.RenderGlow, TextID: "AdVidOpt.wnd:RenderGlow"},
		{ID: 2016, CFlag: legacy.Get_nox_client_fadeObjects_80836_ptr(), TextID: "AdVidOpt.wnd:FadeObjects"},
		{ID: 2017, Bool: &nox_client_showTooltips_80840, TextID: "AdVidOpt.wnd:ShowTooltips"},
		{ID: 2020, Bool: &nox_client_drawFrontWalls_80812, TextID: "AdVidOpt.wnd:DrawFrontWalls"},
		{ID: 2021, CFlag: legacy.Get_nox_client_translucentFrontWalls_805844_ptr(), TextID: "AdVidOpt.wnd:TranslucentFrontWalls"},
		{ID: 2022, CFlag: legacy.Get_nox_client_highResFrontWalls_80820_ptr(), TextID: "AdVidOpt.wnd:InterlacedFrontWalls"},
		{ID: 2031, CFlag: legacy.Get_nox_client_highResFloors_154952_ptr(), TextID: "AdVidOpt.wnd:InterlacedFloors"},
		{ID: 2032, Bool: &nox_client_lockHighResFloors_1193152, TextID: "AdVidOpt.wnd:LockHiResFloors"},
		{ID: 2033, Bool: &nox_client_texturedFloors2_154960, TextID: "AdVidOpt.wnd:FlatShadedFloors"},
		{ID: 2052, Flag: noxflags.EngineNoSoftLights, Text: "Disable Soft Light", TextID: "AdVidOpt.wnd:NoSoftLights"},
		//{ID: 2041, CFlag: &C.nox_client_renderBubbles_80844, TextID: "AdVidOpt.wnd:RenderBubbles"},
		{ID: 2040, Bool: &nox_client_renderGUI_80828, TextID: "AdVidOpt.wnd:RenderGUI"},
	}
	for _, opt := range c.noxVideoAdvList {
		if opt.ID == 0 {
			panic("ID must be set")
		}
		c.noxVideoAdvOpts[opt.ID] = opt
	}
	c.noxVideoScaleOpts = []*videoScaleOpt{
		{TextID: 2060, SliderID: 2061, Text: "GUI", Tooltip: "GUI scale: applies when a game is loaded or started", Ptr: &noxGUIScale, Key: configVideoGUIScale},
		{TextID: 2062, SliderID: 2063, Text: "Cursor", Tooltip: "Cursor and tooltip scale", Ptr: &noxCursorScale, Key: configVideoCursorScale},
	}
}

func (c *guiAdvOptions) scaleOptBySlider(id uint) *videoScaleOpt {
	for _, opt := range c.noxVideoScaleOpts {
		if opt.SliderID == id {
			return opt
		}
	}
	return nil
}

func (c *guiAdvOptions) updateScaleText(opt *videoScaleOpt) {
	if w := c.nox_win_advVideoOpts_1522600.ChildByID(opt.TextID); w != nil {
		w.Func94(&gui.StaticTextSetText{Str: opt.label(), Val: -1})
	}
}

func (c *Client) NewGUIAdvOptsOn(par *gui.Window) {
	c.guiAdv.NewOn(par)
}

func (c *Client) GUIAdvVideoOptsLoad() {
	c.guiAdv.nox_client_advVideoOptsLoad()
}

func (c *guiAdvOptions) NewOn(par *gui.Window) {
	c.nox_win_advVideoOpts_1522600 = c.newAdvVideoOpts()
	c.nox_win_advVideoOpts_1522600.SetParent(par)
	c.nox_client_advVideoOptsLoad()
}

func (c *guiAdvOptions) nox_client_advVideoOptsLoad() {
	for id, opt := range c.noxVideoAdvOpts {
		if w := c.nox_win_advVideoOpts_1522600.ChildByID(id); w != nil {
			v := opt.Def
			if opt.Get != nil {
				v = opt.Get()
			} else if opt.Flag != 0 {
				v = noxflags.HasEngine(opt.Flag)
			} else if opt.Bool != nil {
				v = *opt.Bool
			} else if opt.CFlag != nil {
				v = *opt.CFlag != 0
			}
			w.DrawData().Field0Set(0x4, v)
		}
	}
	for _, opt := range c.noxVideoScaleOpts {
		if w := c.nox_win_advVideoOpts_1522600.ChildByID(opt.SliderID); w != nil {
			w.Func94(gui.AsWindowEvent(0x400A, uintptr(videoScaleToSlider(*opt.Ptr)), 0))
		}
		c.updateScaleText(opt)
	}
}

func (c *guiAdvOptions) nox_client_advVideoOptsProc_4CB5D0(win *gui.Window, ev gui.WindowEvent) gui.WindowEventResp {
	switch ev := ev.(type) {
	case *WindowEvent0x4005:
		clientPlaySoundSpecial(sound.SoundShellSelect, 100)
		return gui.RawEventResp(1)
	case *WindowEvent0x4009:
		if opt := c.scaleOptBySlider(ev.Win.ID()); opt != nil {
			setGUIScaleOpt(opt.Ptr, opt.Key, videoScaleFromSlider(ev.Val))
			c.updateScaleText(opt)
		}
		return nil
	case *WindowEvent0x4007:
		clientPlaySoundSpecial(sound.SoundButtonPress, 100)
		id := ev.Win.ID()
		switch id {
		case 2010:
			// always enabled
			legacyUnlockSurface = true
			win.DrawData().Field0Set(0x4, true)
			return nil
		case 2033:
			v := nox_client_texturedFloors2_154960
			nox_client_texturedFloors2_154960 = !v
			nox_client_texturedFloors_154956 = !v
			nox_xxx_tileSetDrawFn_481420()
			legacy.Set_dword_5d4594_1193156(0)
			return nil
		case 2099:
			detectBestVideoSettings()
			c.nox_client_advVideoOptsLoad()
			if par := win.Parent(); par != nil {
				par.Func94(ev)
			}
			return nil
		}
		if opt, ok := c.noxVideoAdvOpts[id]; ok {
			if opt.Toggle != nil {
				opt.Toggle()
			} else if opt.Flag != 0 {
				noxflags.ToggleEngine(opt.Flag)
			} else if opt.Bool != nil {
				v := *opt.Bool
				*opt.Bool = !v
			} else if opt.CFlag != nil {
				v := *opt.CFlag != 0
				*opt.CFlag = uint32(bool2int(!v))
			}
		}
		switch id {
		case 2020:
			legacy.Sub_49B3C0()
		case 2040:
			nox_xxx_xxxRenderGUI_587000_80832 = nox_client_renderGUI_80828
		}
		return nil
	}
	return nil
}

func (c *guiAdvOptions) newAdvVideoOpts() *gui.Window {
	draw, drawFree := gui.NewWindowData()
	defer drawFree()

	draw.Status = gui.StatusEnabled | gui.StatusNoFocus

	draw.SetHighlightColor(noxcolor.RGB5551Color(192, 128, 128))
	draw.SetTextColor(noxcolor.RGB5551Color(240, 180, 42))
	draw.SetEnabledColor(color.Transparent)
	draw.SetDisabledColor(color.Transparent)
	draw.SetSelectedColor(color.Transparent)
	draw.SetBackgroundColor(color.Transparent)
	draw.SetFont(c.cli.r.Fonts.FontPtrByName(noxfont.LargeName))

	const (
		width  = 184
		height = 315
	)

	root := c.cli.GUI.NewUserWindow(nil, 2000, draw.Status, 280, 38, width, height, draw, c.nox_client_advVideoOptsProc_4CB5D0)

	const (
		pad     = 3
		cheight = 15
		crow    = 18
	)

	n := 0
	for _, opt := range c.noxVideoAdvList {
		if opt.Hidden {
			continue
		}
		y := pad + crow*n
		str := opt.Text
		if opt.TextID != "" {
			if e, ok := c.cli.Strings().GetInFile(opt.TextID, "psscript.c"); ok {
				str = e.Value().Str
			}
		}
		NewCheckBox(c.cli.GUI, root, opt.ID, pad, y, width, cheight, str)
		n++
	}

	const scaleTextWidth = 100
	for _, opt := range c.noxVideoScaleOpts {
		y := pad + crow*n
		text := c.cli.GUI.NewStaticText(root, opt.TextID, pad, y, scaleTextWidth, cheight, false, false, opt.label())
		slider := NewHorizontalSlider(root, opt.SliderID, pad+scaleTextWidth, y, width-scaleTextWidth-2*pad, cheight, 0, videoScaleSliderMax)
		if opt.Tooltip != "" {
			// tooltips are only looked up in enabled windows, and the slider knob is a separate window
			text.Flags |= gui.StatusEnabled
			text.DrawData().SetHighlightColor(color.Transparent)
			text.DrawData().SetTooltip(c.cli.Strings(), opt.Tooltip)
			slider.DrawData().SetTooltip(c.cli.Strings(), opt.Tooltip)
			if knob := slider.Field100(); knob != nil {
				knob.DrawData().SetTooltip(c.cli.Strings(), opt.Tooltip)
			}
		}
		n++
	}
	return root
}
