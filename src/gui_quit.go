package opennox

import (
	"image"
	"unsafe"

	"github.com/opennox/libs/common"

	"github.com/opennox/opennox/v1/client/gui"
	noxflags "github.com/opennox/opennox/v1/common/flags"
	"github.com/opennox/opennox/v1/common/memmap"
	"github.com/opennox/opennox/v1/internal/netstr"
	"github.com/opennox/opennox/v1/legacy"
	"github.com/opennox/opennox/v1/legacy/common/alloc"
)

func nox_xxx_wndLoadQuitMenu_445790() int {
	win := newWindowFromFile(noxClient.GUI, "QuitMenu.wnd", gui.WrapFuncC(legacy.Get_nox_xxx_menuGameOnButton_445840()))
	if win == nil {
		return 0
	}
	legacy.Set_nox_wnd_quitMenu_825760(win)
	win.SetAllFuncs(func(win *gui.Window, ev gui.WindowEvent) gui.WindowEventResp {
		return gui.RawEventResp(1)
	}, func(win *gui.Window, draw *gui.WindowData) int {
		pos := win.GlobalPos()
		sz := win.Size()
		noxClient.r.DrawRectFilledAlpha(pos.X, pos.Y, sz.X, sz.Y)
		return 1
	}, nil)
	win.DrawData().SetBackgroundColor(nox_color_black_2650656)
	quitMenuFileY = win.Offs().Y
	setQuitMenuPos(win)
	legacy.Sub_445C40()
	dword_5d4594_825752 = 0
	dword_5d4594_825768 = 0
	str := strMan.GetStringInFile("Vote", "guiquit.c")
	alloc.StrCopyZero(memmap.PtrT[[256]byte](0x5D4594, 825772)[:], str)
	return 1
}

// quitMenuFileY is the vertical position of the quit menu, as defined in the window file.
var quitMenuFileY int

// setQuitMenuPos places the quit menu for the current screen size.
func setQuitMenuPos(win *gui.Window) {
	sz := win.Size()
	win.Off.X = int(nox_win_width-sz.X) / 2
	win.EndPos.X = win.Off.X + sz.X
	win.Off.Y = quitMenuFileY
	if nox_win_height > 768 {
		win.Off.Y = (nox_win_height - sz.Y - nox_win_height/3) / 2
	}
	win.EndPos.Y = win.Off.Y + sz.Y
}

// showQuitDialog shows a dialog for leaving the game: to the main menu, to the desktop, or cancel.
func showQuitDialog() {
	const file = "C:\\NoxPost\\src\\client\\Gui\\guiquit.c"
	title := strMan.GetStringInFile("GUIQuit.c:ReallyQuitTitle", file)
	text := strMan.GetStringInFile("GUIQuit.c:ReallyQuitMessage", file)
	flags := gui.DialogYesButton | gui.DialogNoButton | gui.DialogCancelButton | gui.DialogFlag6
	NewDialogWindow(legacy.Get_nox_wnd_quitMenu_825760(), title, text, flags,
		legacy.Nox_xxx_quitDialogYes_445B20, legacy.Nox_xxx_quitDialogNo_445B30)
	dia := nox_gui_curDialog_830224
	if dia == nil {
		return
	}
	dialogNoFunc = mainloopExitToDesktop
	dialogEscCancels = true

	// the dialog is made wider for larger buttons, and taller for the cancel button below them
	const (
		width   = 180
		height  = 160
		pad     = 10
		btnW    = 75
		btnY    = 100
		cancelW = 55
	)
	setRect := func(win *gui.Window, x, y, w, h int) {
		win.SizeVal = image.Pt(w, h)
		win.SetPos(image.Pt(x, y))
	}
	setImages := func(win *gui.Window, name string) {
		d := win.DrawData()
		d.SetBackgroundImage(nox_xxx_gLoadImg(name))
		d.SetHighlightImage(nox_xxx_gLoadImg(name + "Lit"))
		d.SetSelectedImage(nox_xxx_gLoadImg(name + "Lit"))
	}
	vsz := videoGetWindowSize()
	setRect(dia, (vsz.X-width)/2, (vsz.Y-height)/2, width, height)
	for _, id := range []uint{guiDialogTitleID, guiDialogTextID} {
		w := dia.ChildByID(id)
		off := w.Offs()
		setRect(w, off.X, off.Y, width-2*off.X, w.Size().Y)
	}
	menu := dia.ChildByID(guiDialogYesID)
	setRect(menu, pad, btnY, btnW, 20)
	setImages(menu, "UIButtonLg")
	menu.DrawData().SetText("Main Menu")
	desktop := dia.ChildByID(guiDialogNoID)
	setRect(desktop, width-pad-btnW, btnY, btnW, 20)
	setImages(desktop, "UIButtonLg")
	desktop.DrawData().SetText("To Desktop")
	cancel := dia.ChildByID(guiDialogCancelID)
	setRect(cancel, (width-cancelW)/2, btnY+20+6, cancelW, 20)
	dia.Focus()
}

func sub_467440(a1 int) {
	*memmap.PtrUint32(0x5D4594, 1062544) = uint32(a1)
}

func sub_446190() {
	s := noxServer
	nox_xxx_serverIsClosing_825764 = false
	if noxflags.HasGame(noxflags.GameHost) {
		str := strMan.GetStringInFile("ServerManualShutdown", "guiquit.c")
		netstr.Log.Println(str)
	}
	if noxflags.HasGame(noxflags.GameFlag26) {
		sub_41E300(9)
		legacy.Nox_xxx____setargv_4_44B000()
	} else if s.nox_xxx_isQuest_4D6F50() {
		if sub4D6F30() {
			noxClient.nox_game_checkStateMenu_43C2F0()
		}
		legacy.Sub_4D70B0()
		sub4D6F40(false)
		sub_4D6F90(0)
	}
	if dword_5d4594_825768 == 0 {
		dword_5d4594_825752 = 0
		dword_5d4594_825768 = noxServer.Frame()
		if noxflags.HasGame(noxflags.GameHost) {
			if false && noxflags.HasGame(noxflags.GameFlag15|noxflags.GameFlag16) {
				sub_416150(15, 0)
			}
			legacy.Sub_509CB0()
		}
		if noxflags.HasGame(noxflags.GameHost) {
			dword_5d4594_825752 = s.Players.Count() - 1
			sub_467440(0)
			if dword_5d4594_825752 > 0 {
				s.NetSendServerQuit()
				v3 := strMan.GetStringInFile("ShuttingDown", "C:\\NoxPost\\src\\client\\Gui\\guiquit.c")
				v2 := strMan.GetStringInFile("Wolchat.c:PleaseWait", "C:\\NoxPost\\src\\client\\Gui\\guiquit.c")
				NewDialogWindow(nil, v2, v3, 0, nil, nil)
				sub_44A360(1)
				goto LABEL_20
			}
		} else if !noxflags.HasGame(noxflags.GameOnline) {
			goto LABEL_20
		}
		sub_446380()
	}
LABEL_20:
	if legacy.Get_dword_5d4594_2650652() != 0 {
		if legacy.Sub_41E2F0() == 9 {
			legacy.Sub_41F4B0()
			legacy.Sub_41EC30()
			legacy.Sub_446490(0)
			legacy.Nox_xxx____setargv_4_44B000()
		}
	}
}

func sub_446380() {
	sub_44A400()
	if noxflags.HasGame(noxflags.GameOnline) && !noxflags.HasGame(noxflags.GameModeQuest) && !sub4D6F30() {
		legacy.Sub_41CEE0(unsafe.Pointer(clientCurSave()), 1)
	}
	legacy.Nox_xxx_guiServerOptionsHide_4597E0(1)
	nox_game_exit_xxx2()
	sub_446060()
}

func clientLoadCoopAuto445B40() int {
	sub_413A00(0)
	if clientLoadCoopGame(common.SaveAuto) {
		return 1
	}
	str := strMan.GetStringInFile("GUISave.c:SaveErrorTitle", "guiquit.c")
	NewDialogWindow(nil, str, str, gui.DialogOKButton|gui.DialogFlag6, nil, nil)
	return 1
}
