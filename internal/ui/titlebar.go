package ui

import (
	_ "embed"
	"image/color"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/zeroqual/playerzzz/internal/winframe"
)

//go:embed assets/icon.png
var iconPNG []byte

var appIcon = fyne.NewStaticResource("icon.png", iconPNG)

const (
	titleBarHeight = 36
	winButtonWidth = 46
)

func (a *App) wantCustomFrame() bool {
	return winframe.Supported() && !a.cfg.SystemFrame
}

func (a *App) enableFrame() {
	if !a.wantCustomFrame() {
		return
	}
	nw, ok := a.win.(driver.NativeWindow)
	if !ok {
		return
	}
	nw.RunNative(func(ctx any) {
		if wc, ok := ctx.(driver.WindowsWindowContext); ok {
			a.hwnd = wc.HWND
		}
	})
	if a.hwnd == 0 {
		log.Printf("winframe: нет HWND, оставляем системную рамку")
		return
	}
	a.updateFrameMetrics()
	if err := winframe.Enable(a.hwnd); err != nil {
		log.Printf("winframe: %v (оставляем системную рамку)", err)
		return
	}
	a.customFrame = true
	a.rebuild()
}

func (a *App) updateFrameMetrics() {
	scale := a.win.Canvas().Scale()
	winframe.SetMetrics(int32(titleBarHeight*scale), int32(3*winButtonWidth*scale))
	if a.maxBtn != nil {
		icon := theme.WindowMaximizeIcon()
		if winframe.IsMaximized(a.hwnd) {
			icon = theme.ViewRestoreIcon()
		}
		a.maxBtn.setIcon(icon)
	}
}

func (a *App) buildTitleBar() fyne.CanvasObject {
	icon := canvas.NewImageFromResource(appIcon)
	icon.FillMode = canvas.ImageFillContain
	name := canvas.NewText("playerzzz", a.pal.Muted)
	name.TextSize = 12
	name.TextStyle = fyne.TextStyle{Bold: true}
	brand := container.NewHBox(hspace(12), container.NewCenter(fixedSize(16, 16, icon)), container.NewCenter(name))

	leftBg := canvas.NewRectangle(a.pal.Sidebar)
	left := withWidth(249, container.NewStack(leftBg, brand))

	minBtn := newWinButton(a.pal, theme.WindowMinimizeIcon(), false, func() { winframe.Minimize(a.hwnd) })
	a.maxBtn = newWinButton(a.pal, theme.WindowMaximizeIcon(), false, func() {
		winframe.ToggleMaximize(a.hwnd)
		a.updateFrameMetrics()
	})
	closeBtn := newWinButton(a.pal, theme.WindowCloseIcon(), true, func() { a.win.Close() })
	buttons := container.NewHBox(minBtn, a.maxBtn, closeBtn)
	a.updateFrameMetrics()

	bg := canvas.NewRectangle(a.pal.Bg)
	return container.NewBorder(nil, nil, left, buttons, bg)
}

type winButton struct {
	widget.BaseWidget
	icon    *widget.Icon
	bg      *canvas.Rectangle
	onTap   func()
	danger  bool
	pal     Palette
	hovered bool
}

func newWinButton(pal Palette, icon fyne.Resource, danger bool, onTap func()) *winButton {
	b := &winButton{icon: widget.NewIcon(icon), onTap: onTap, danger: danger, pal: pal}
	b.bg = canvas.NewRectangle(color.Transparent)
	b.ExtendBaseWidget(b)
	return b
}

func (b *winButton) setIcon(res fyne.Resource) {
	b.icon.SetResource(res)
}

func (b *winButton) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(b.bg, container.NewCenter(fixedSize(14, 14, b.icon))))
}

func (b *winButton) MinSize() fyne.Size { return fyne.NewSize(winButtonWidth, titleBarHeight) }

func (b *winButton) Tapped(*fyne.PointEvent) {
	if b.onTap != nil {
		b.onTap()
	}
}

func (b *winButton) MouseIn(*desktop.MouseEvent) {
	b.hovered = true
	if b.danger {
		b.bg.FillColor = color.NRGBA{R: 0xE8, G: 0x11, B: 0x23, A: 0xFF}
	} else {
		b.bg.FillColor = withAlpha(b.pal.Raised, 0xFF)
	}
	b.bg.Refresh()
}

func (b *winButton) MouseMoved(*desktop.MouseEvent) {}

func (b *winButton) MouseOut() {
	b.hovered = false
	b.bg.FillColor = color.Transparent
	b.bg.Refresh()
}

func (a *App) frameTick() {
	for range time.Tick(500 * time.Millisecond) {
		fyne.Do(func() {
			if a.customFrame {
				a.updateFrameMetrics()
			}
		})
	}
}
