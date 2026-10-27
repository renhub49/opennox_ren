package opennox

import (
	"context"
	"image"
	"strconv"

	"github.com/opennox/libs/client/seat"
	"github.com/opennox/libs/console"
	"github.com/opennox/libs/log"
	"github.com/spf13/viper"
)

const (
	configVideoGUIScale    = "video.gui_scale"
	configVideoCursorScale = "video.cursor_scale"

	guiScaleMin = 1.0
	guiScaleMax = 3.0
)

const (
	noxDefaultWidth  = seat.DefaultWidth
	noxDefaultHeight = seat.DefaultHeight
	noxDefaultDepth  = seat.DefaultDepth
)

const (
	gammaMin = 0.1
	gammaDef = 1.0
	gammaMax = 3.0
	gammaInc = 0.1
)

var (
	videoLog         = log.New("video")
	noxVideoModeMenu = image.Point{
		X: noxDefaultWidth,
		Y: noxDefaultHeight,
	}
	noxVideoMax = image.Point{
		X: noxMaxWidth,
		Y: noxMaxHeight,
	}
	noxVideoModes = []image.Point{
		{640, 480},
		{800, 600},
		{1024, 768},
		{1280, 720},
		{1920, 1080},
		{2560, 1440},
		{3840, 2160},
	}
	noxVideoGamma = float32(gammaDef)

	nox_video_cutSize = 100

	g_scaled_cfg     = 0
	g_fullscreen_cfg = 0

	// noxGUIScale is the requested GUI scale. It is applied when the game switches to the game video mode.
	noxGUIScale = float32(1)

	// noxCursorScale is the requested scale of the cursor and tooltips.
	noxCursorScale = float32(1)

	// noxScreen tracks the screen space used by the legacy code (nox_win_width/height and mouse position).
	// The world is always rendered at full resolution, while GUI can use a smaller screen space that is upscaled.
	noxScreen = struct {
		world image.Point // size of the world render
		gui   float32     // GUI scale for the current video mode
		cur   float32     // scale of the current screen space
	}{gui: 1, cur: 1}
)

func init() {
	noxConsole.Register(&console.Command{
		Token:  "gamma",
		HelpID: "gammahelp",
		Flags:  console.ClientServer,
		Func: func(ctx context.Context, c *console.Console, tokens []string) bool {
			if len(tokens) == 0 {
				c.Printf(console.ColorLightYellow, "gamma = %d", getGammaSlider())
				return true
			} else if len(tokens) > 1 {
				return false
			}
			s := tokens[0]
			if len(s) == 0 {
				return false
			}
			add := 0
			if s[0] == '+' {
				add = +1
				s = s[1:]
			} else if s[0] == '-' {
				add = -1
				s = s[1:]
			}
			v, err := strconv.Atoi(s)
			if err != nil {
				return false
			}
			gamma := v
			if add != 0 {
				gamma = getGammaSlider() + add*gamma
			}
			setGammaSlider(gamma)
			return true
		},
	})
	noxConsole.Register(&console.Command{
		Token: "guiscale",
		Help:  "guiscale [1.0-3.0]: sets GUI scale, applies on the next game",
		Flags: console.ClientServer,
		Func: func(ctx context.Context, c *console.Console, tokens []string) bool {
			return consoleScaleCmd(c, "guiscale", tokens, &noxGUIScale, configVideoGUIScale)
		},
	})
	noxConsole.Register(&console.Command{
		Token: "cursorscale",
		Help:  "cursorscale [1.0-3.0]: sets cursor and tooltip scale",
		Flags: console.ClientServer,
		Func: func(ctx context.Context, c *console.Console, tokens []string) bool {
			return consoleScaleCmd(c, "cursorscale", tokens, &noxCursorScale, configVideoCursorScale)
		},
	})
	viper.SetDefault(configVideoGUIScale, 1.0)
	viper.SetDefault(configVideoCursorScale, 1.0)
	registerOnConfigRead(func() {
		noxGUIScale = clampGUIScale(float32(viper.GetFloat64(configVideoGUIScale)))
		noxCursorScale = clampGUIScale(float32(viper.GetFloat64(configVideoCursorScale)))
	})
}

func consoleScaleCmd(c *console.Console, name string, tokens []string, ptr *float32, key string) bool {
	if len(tokens) == 0 {
		c.Printf(console.ColorLightYellow, "%s = %.2f", name, *ptr)
		return true
	} else if len(tokens) > 1 {
		return false
	}
	v, err := strconv.ParseFloat(tokens[0], 32)
	if err != nil {
		return false
	}
	setGUIScaleOpt(ptr, key, float32(v))
	c.Printf(console.ColorLightYellow, "%s = %.2f", name, *ptr)
	return true
}

func setGUIScaleOpt(ptr *float32, key string, v float32) {
	*ptr = clampGUIScale(v)
	viper.Set(key, *ptr)
	writeConfigLater()
}

func clampGUIScale(v float32) float32 {
	if v < guiScaleMin {
		v = guiScaleMin
	}
	if v > guiScaleMax {
		v = guiScaleMax
	}
	return v
}

// guiScaleFor limits the scale, so that the scaled screen is not smaller than the original game resolution.
func guiScaleFor(world image.Point, s float32) float32 {
	if s <= 1 || world.X <= 0 || world.Y <= 0 {
		return 1
	}
	if m := float32(world.X) / noxDefaultWidth; s > m {
		s = m
	}
	if m := float32(world.Y) / noxDefaultHeight; s > m {
		s = m
	}
	if s < 1 {
		return 1
	}
	return s
}

// screenSizeFor returns the size of the screen space with a given scale.
func screenSizeFor(world image.Point, s float32) image.Point {
	if s <= 1 {
		return world
	}
	return image.Point{
		X: int(float32(world.X)/s + 0.5),
		Y: int(float32(world.Y)/s + 0.5),
	}
}

// videoGetWorldSize returns the size of the world render, which is not affected by GUI scale.
func videoGetWorldSize() image.Point {
	if noxScreen.world == (image.Point{}) {
		return videoGetWindowSize()
	}
	return noxScreen.world
}

// setScreenScale switches nox_win_width/height and the mouse position to a screen space
// that is s times smaller than the world render. It returns the previous scale.
func setScreenScale(s float32) float32 {
	prev := noxScreen.cur
	if s == prev {
		return prev
	}
	noxScreen.cur = s
	sz := screenSizeFor(noxScreen.world, s)
	videoSetWindowSize(sz)
	if noxClient != nil {
		noxClient.SetMouseScreen(noxScreen.world, sz)
	}
	return prev
}

// withScreenScale runs the function in a screen space with a given scale.
func withScreenScale(s float32, fnc func()) {
	prev := setScreenScale(s)
	defer setScreenScale(prev)
	fnc()
}

// withWorldScreen runs the function in the screen space of the world render.
func withWorldScreen(fnc func()) {
	withScreenScale(1, fnc)
}

// applyGUIScale switches the default screen space to the requested GUI scale.
// It must be called after the video mode is set, but before the game GUI is created.
func applyGUIScale() {
	noxScreen.gui = guiScaleFor(noxScreen.world, noxGUIScale)
	setScreenScale(noxScreen.gui)
}

func videoModeID() int {
	mode := noxClient.videoGetGameMode()
	for i, res := range noxVideoModes {
		if mode == res {
			return i
		}
	}
	for i, res := range noxVideoModes {
		if mode.X == res.X {
			return i
		}
	}
	for i, res := range noxVideoModes {
		if mode.Y == res.Y {
			return i
		}
	}
	area := mode.X * mode.Y
	min := 0
	ind := -1
	for i, res := range noxVideoModes {
		area2 := res.X * res.Y
		if ind == -1 || abs(area-area2) < min {
			ind = i
			min = abs(area - area2)
		}
	}
	return ind
}

func videoModeByID(code byte) (image.Point, bool) {
	if int(code) >= len(noxVideoModes) {
		return image.Point{}, false
	}
	return noxVideoModes[code], true
}

func videoGetMenuMode() image.Point {
	return noxVideoModeMenu
}

func videoSetMenuMode(mode image.Point) {
	noxVideoModeMenu = mode
}

func videoSetMaxSize(sz image.Point) {
	noxVideoMax = sz
}

func videoGetMaxSize() image.Point {
	return noxVideoMax
}

func videoResizeView(mode image.Point) {
	max := videoGetMaxSize()
	if mode.X > max.X {
		mode.X = max.X
	}
	if mode.Y > max.Y {
		mode.Y = max.Y
	}
	noxScreen.world = mode
	noxScreen.gui = 1
	noxScreen.cur = 1
	videoSetWindowSize(mode)
	if noxClient != nil {
		noxClient.SetMouseScreen(mode, mode)
	}
}

func changeWindowedOrFullscreen() {
	cfgUpdateFullScreen()
}

func nox_video_setCutSize(v int) {
	if v < 65 {
		v = 65
	}
	if v > 100 {
		v = 100
	}
	nox_video_cutSize = v
}

func nox_video_getCutSize() int {
	return nox_video_cutSize
}

func getGammaSlider() int {
	return int(100 * (getGamma() - gammaMin) / gammaMax)
}

func setGammaSlider(v int) {
	fv := float32(v) / 100
	setGamma(gammaMin + gammaMax*fv)
}

func setGammaSliderOpts(v int) {
	setGammaSlider(v)
	viper.Set(configVideoGamma, getGamma())
	writeConfigLater()
}

func updateGamma(value int) {
	var dv float32
	if value > 0 {
		dv = +gammaInc
	} else if value < 0 {
		dv = -gammaInc
	}
	setGamma(getGamma() + dv)
}

func getGamma() float32 {
	return noxVideoGamma
}

func setGamma(v float32) {
	if v < gammaMin {
		v = gammaMin
	} else if v > gammaMax {
		v = gammaMax
	}
	if noxVideoGamma == v {
		return
	}
	videoLog.Printf("gamma: %v", v)
	noxVideoGamma = v
	noxClient.SetScreenGamma(v)
}
