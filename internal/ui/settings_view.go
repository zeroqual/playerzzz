package ui

import (
	"net/url"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/zeroqual/playerzzz/internal/i18n"
	"github.com/zeroqual/playerzzz/internal/winframe"
)

func (a *App) buildSettingsView() fyne.CanvasObject {
	t := a.tr
	title := heading(t.S("settings.title"), theme.SizeNameHeadingText)

	sections := container.NewVBox(
		title,
		a.profileSection(),
		a.languageSection(),
		a.themeSection(),
		a.librarySection(),
		a.coversSection(),
	)
	if winframe.Supported() {
		sections.Add(a.windowSection())
	}
	sections.Add(a.aboutSection())
	return container.NewVScroll(container.NewPadded(container.NewPadded(sections)))
}

func sectionTitle(s string) *widget.Label {
	return widget.NewLabelWithStyle(s, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

func hint(s string) *widget.Label {
	l := muted(s)
	l.Wrapping = fyne.TextWrapWord
	return l
}

func (a *App) profileSection() fyne.CanvasObject {
	t := a.tr
	change := widget.NewButtonWithIcon(t.S("settings.avatarChange"), theme.AccountIcon(), a.pickAvatar)
	remove := widget.NewButton(t.S("settings.avatarRemove"), func() {
		_ = os.Remove(filepath.Join(a.store.Dir, avatarFile))
		a.cfg.Profile.Avatar = ""
		a.avatarRes = nil
		a.saveSettings()
		a.refreshSidebar()
		a.showView()
	})
	remove.Importance = widget.LowImportance
	if a.cfg.Profile.Avatar == "" {
		remove.Hide()
	}

	nick := widget.NewEntry()
	nick.SetPlaceHolder(t.S("profile.default"))
	nick.SetText(a.cfg.Profile.Nickname)
	nick.OnChanged = func(s string) {
		if len([]rune(s)) > 32 {
			return
		}
		a.cfg.Profile.Nickname = s
		a.saveSettings()
		a.refreshSidebar()
	}

	avatar := container.NewCenter(a.avatarView(72))
	right := container.NewVBox(
		container.NewHBox(change, remove),
		container.NewBorder(nil, nil, widget.NewLabel(t.S("settings.nickname")), nil, nick),
	)
	return a.card(container.NewVBox(
		sectionTitle(t.S("settings.profile")),
		container.NewBorder(nil, nil, avatar, nil, right),
		hint(t.S("settings.profileHint")),
	))
}

func (a *App) pickAvatar() {
	a.pickImage(func(data []byte) {
		if err := saveAvatar(a.store.Dir, data); err != nil {
			a.showErr(err)
			return
		}
		a.cfg.Profile.Avatar = avatarFile
		a.avatarRes = nil
		a.saveSettings()
		a.refreshSidebar()
		a.showView()
	})
}

func (a *App) languageSection() fyne.CanvasObject {
	t := a.tr
	buttons := container.NewHBox()
	for _, l := range i18n.Languages {
		code := l.Code
		b := widget.NewButton(l.Name, func() {
			if a.cfg.Language == code {
				return
			}
			a.cfg.Language = code
			a.saveSettings()
			a.applyLook()
			a.rebuild()
		})
		if code == a.tr.Lang() {
			b.Importance = widget.HighImportance
		}
		buttons.Add(b)
	}
	return a.card(container.NewVBox(
		sectionTitle(t.S("settings.language")),
		hint(t.S("settings.languageHint")),
		buttons,
	))
}

func (a *App) themeSection() fyne.CanvasObject {
	t := a.tr
	grid := container.New(layout.NewGridWrapLayout(fyne.NewSize(150, 122)))
	for _, p := range Palettes {
		id := p.ID
		grid.Add(newTappable(a.pal, container.NewPadded(container.NewVBox(
			themePreview(p),
			widget.NewLabelWithStyle(t.S("theme."+id), fyne.TextAlignCenter, fyne.TextStyle{}),
		)), a.pal.ID == id, func() {
			if a.cfg.Theme == id {
				return
			}
			a.cfg.Theme = id
			a.saveSettings()
			a.applyLook()
			a.rebuild()
		}))
	}
	return a.card(container.NewVBox(sectionTitle(t.S("settings.theme")), grid))
}

func themePreview(p Palette) fyne.CanvasObject {
	bg := canvas.NewRectangle(p.Bg)
	bg.CornerRadius = 10
	bg.StrokeColor = p.Border
	bg.StrokeWidth = 1

	side := canvas.NewRectangle(p.Sidebar)
	side.SetMinSize(fyne.NewSize(28, 1))
	line := func(c Palette, w float32) fyne.CanvasObject {
		r := canvas.NewRectangle(c.Muted)
		r.CornerRadius = 2
		r.SetMinSize(fyne.NewSize(w, 4))
		return container.NewHBox(r)
	}
	accent := canvas.NewCircle(p.Accent)
	dot := fixedSize(14, 14, accent)
	content := container.NewVBox(
		container.NewHBox(dot, container.NewCenter(line(p, 40))),
		line(p, 56), line(p, 32),
	)
	bar := canvas.NewRectangle(p.Panel)
	bar.SetMinSize(fyne.NewSize(1, 12))

	inner := container.NewBorder(nil, bar, side, nil, container.NewPadded(content))
	return fixedSize(126, 70, container.NewStack(bg, container.NewPadded(inner)))
}

func (a *App) librarySection() fyne.CanvasObject {
	t := a.tr
	path := a.cfg.Folder
	if path == "" {
		path = t.S("settings.folderNone")
	}
	folder := ellipsis(widget.NewLabelWithStyle(path, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	change := widget.NewButtonWithIcon(t.S("settings.change"), theme.FolderOpenIcon(), a.pickFolder)
	return a.card(container.NewVBox(
		sectionTitle(t.S("settings.library")),
		container.NewBorder(nil, nil, widget.NewIcon(theme.FolderIcon()), change, folder),
		hint(t.S("settings.folderHint")),
	))
}

func (a *App) coversSection() fyne.CanvasObject {
	t := a.tr
	check := widget.NewCheck(t.S("settings.covers"), nil)
	check.SetChecked(a.cfg.Covers)
	check.OnChanged = func(on bool) {
		a.cfg.Covers = on
		a.saveSettings()
		a.refreshSidebar()
		a.refreshList()
		a.refreshNowPlaying()
	}
	return a.card(container.NewVBox(check, hint(t.S("settings.coversHint"))))
}

func (a *App) windowSection() fyne.CanvasObject {
	t := a.tr
	check := widget.NewCheck(t.S("settings.systemFrame"), nil)
	check.SetChecked(a.cfg.SystemFrame)
	check.OnChanged = func(on bool) {
		a.cfg.SystemFrame = on
		a.saveSettings()
	}
	return a.card(container.NewVBox(sectionTitle(t.S("settings.window")), check, hint(t.S("settings.systemFrameHint"))))
}

func (a *App) aboutSection() fyne.CanvasObject {
	t := a.tr
	src, _ := url.Parse("https://github.com/zeroqual/playerzzz")
	return a.card(container.NewVBox(
		sectionTitle(t.S("settings.about")),
		container.NewHBox(muted("playerzzz · "+t.F("settings.version", appVersion)), layout.NewSpacer(),
			widget.NewHyperlink(t.S("settings.source"), src)),
	))
}
