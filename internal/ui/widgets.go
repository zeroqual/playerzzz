package ui

import (
	"image/color"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type tappable struct {
	widget.BaseWidget
	content  fyne.CanvasObject
	onTap    func()
	bg       *canvas.Rectangle
	selected bool
	hovered  bool
	pal      Palette
	radius   float32
}

func newTappable(pal Palette, content fyne.CanvasObject, selected bool, onTap func()) *tappable {
	t := &tappable{content: content, onTap: onTap, selected: selected, pal: pal, radius: 10}
	t.bg = canvas.NewRectangle(color.Transparent)
	t.bg.CornerRadius = t.radius
	t.ExtendBaseWidget(t)
	t.updateBg()
	return t
}

func (t *tappable) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(t.bg, t.content))
}

func (t *tappable) Tapped(*fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *tappable) Cursor() desktop.Cursor { return desktop.PointerCursor }

func (t *tappable) MouseIn(*desktop.MouseEvent) { t.hovered = true; t.updateBg() }

func (t *tappable) MouseMoved(*desktop.MouseEvent) {}

func (t *tappable) MouseOut() { t.hovered = false; t.updateBg() }

func (t *tappable) updateBg() {
	switch {
	case t.selected:
		t.bg.FillColor = t.pal.Raised
	case t.hovered:
		t.bg.FillColor = withAlpha(t.pal.Raised, 0xA0)
	default:
		t.bg.FillColor = color.Transparent
	}
	t.bg.Refresh()
}

type fixedWidth struct{ w float32 }

func (f fixedWidth) MinSize(objs []fyne.CanvasObject) fyne.Size {
	h := float32(0)
	for _, o := range objs {
		h = max(h, o.MinSize().Height)
	}
	return fyne.NewSize(f.w, h)
}

func (f fixedWidth) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(size)
	}
}

func withWidth(w float32, o fyne.CanvasObject) fyne.CanvasObject {
	return container.New(fixedWidth{w}, o)
}

func fixedSize(w, h float32, o fyne.CanvasObject) fyne.CanvasObject {
	return container.NewGridWrap(fyne.NewSize(w, h), o)
}

func (a *App) card(content fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(a.pal.Panel)
	bg.CornerRadius = 16
	bg.StrokeColor = a.pal.Border
	bg.StrokeWidth = 1
	return container.NewStack(bg, container.NewPadded(container.NewPadded(content)))
}

func (a *App) caption(s string) fyne.CanvasObject {
	t := canvas.NewText(s, a.pal.Muted)
	t.TextSize = 11
	t.TextStyle = fyne.TextStyle{Bold: true}
	return t
}

func heading(s string, size fyne.ThemeSizeName) *widget.Label {
	l := widget.NewLabelWithStyle(s, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	l.SizeName = size
	l.Truncation = fyne.TextTruncateEllipsis
	return l
}

func muted(s string) *widget.Label {
	l := widget.NewLabel(s)
	l.Importance = widget.LowImportance
	return l
}

func ellipsis(l *widget.Label) *widget.Label {
	l.Truncation = fyne.TextTruncateEllipsis
	return l
}

func (a *App) coverView(res fyne.Resource, size, radius float32) fyne.CanvasObject {
	if res != nil && a.cfg.Covers {
		img := canvas.NewImageFromResource(res)
		img.FillMode = canvas.ImageFillCover
		img.CornerRadius = radius
		img.ScaleMode = canvas.ImageScaleSmooth
		return fixedSize(size, size, img)
	}
	return fixedSize(size, size, a.coverPlaceholder(radius))
}

func (a *App) coverPlaceholder(radius float32) fyne.CanvasObject {
	bg := canvas.NewRectangle(a.pal.Raised)
	bg.CornerRadius = radius
	icon := widget.NewIcon(theme.NewPrimaryThemedResource(theme.MediaMusicIcon()))
	return container.NewStack(bg, container.NewPadded(container.NewPadded(icon)))
}

func (a *App) avatarView(size float32) fyne.CanvasObject {
	if res := a.avatarResource(); res != nil {
		img := canvas.NewImageFromResource(res)
		img.FillMode = canvas.ImageFillCover
		img.CornerRadius = size / 2
		img.ScaleMode = canvas.ImageScaleSmooth
		return fixedSize(size, size, img)
	}
	circle := canvas.NewCircle(a.pal.Accent)
	letter := canvas.NewText(initial(a.nickname()), a.pal.OnAccent)
	letter.TextStyle = fyne.TextStyle{Bold: true}
	letter.TextSize = size * 0.45
	letter.Alignment = fyne.TextAlignCenter
	return fixedSize(size, size, container.NewStack(circle, container.NewCenter(letter)))
}

func initial(name string) string {
	r, _ := utf8.DecodeRuneInString(strings.TrimSpace(name))
	if r == utf8.RuneError {
		return "?"
	}
	return strings.ToUpper(string(r))
}
