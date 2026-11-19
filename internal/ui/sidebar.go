package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/zeroqual/playerzzz/internal/config"
)

func (a *App) refreshSidebar() {
	if a.sidebarBox == nil {
		return
	}
	t := a.tr

	name := ellipsis(widget.NewLabelWithStyle(a.nickname(), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	profileText := container.New(layout.NewCustomPaddedVBoxLayout(-14), name, muted(t.S("profile.local")))
	profile := newTappable(a.pal,
		container.NewPadded(container.NewBorder(nil, nil, container.NewCenter(a.avatarView(36)), nil, profileText)),
		false, func() { a.open(viewSettings, "") })

	library := a.navItem(theme.MediaMusicIcon(), t.S("nav.library"), a.view == viewLibrary,
		func() { a.open(viewLibrary, "") })
	settings := a.navItem(theme.SettingsIcon(), t.S("nav.settings"), a.view == viewSettings,
		func() { a.open(viewSettings, "") })

	addBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), nil)
	addBtn.Importance = widget.LowImportance
	addBtn.OnTapped = func() {
		create := fyne.NewMenuItem(t.S("nav.newList"), a.newPlaylistDialog)
		create.Icon = theme.ContentAddIcon()
		imp := fyne.NewMenuItem(t.S("nav.import"), a.importDialog)
		imp.Icon = theme.DownloadIcon()
		impFile := fyne.NewMenuItem(t.S("nav.importFile"), a.importFileDialog)
		impFile.Icon = theme.FolderOpenIcon()
		a.showMenuUnder(fyne.NewMenu("", create, imp, impFile), addBtn)
	}
	plHeader := container.NewBorder(nil, nil, nil, addBtn,
		container.NewHBox(hspace(10), container.NewCenter(a.caption(t.S("nav.playlists")))))

	rows := container.NewVBox()
	for _, pl := range a.pls.Items {
		rows.Add(a.playlistRow(pl))
	}

	top := container.NewVBox(profile, spacer(8), library, spacer(12), plHeader)
	body := container.NewBorder(top, settings, nil, nil, container.NewVScroll(rows))

	bg := canvas.NewRectangle(a.pal.Sidebar)
	a.sidebarBox.Objects = []fyne.CanvasObject{container.NewStack(bg, container.NewPadded(body))}
	a.sidebarBox.Refresh()
}

func (a *App) navItem(icon fyne.Resource, label string, selected bool, onTap func()) fyne.CanvasObject {
	if selected {
		icon = theme.NewPrimaryThemedResource(icon)
	}
	l := widget.NewLabel(label)
	if selected {
		l.TextStyle = fyne.TextStyle{Bold: true}
	}
	row := container.NewHBox(widget.NewIcon(icon), l)
	return newTappable(a.pal, container.NewPadded(row), selected, onTap)
}

func (a *App) playlistRow(pl config.Playlist) fyne.CanvasObject {
	name := ellipsis(widget.NewLabel(pl.Name))
	count := muted(a.tr.Tracks(len(pl.Tracks)))
	text := container.New(layout.NewCustomPaddedVBoxLayout(-14), name, count)
	art := a.playlistArt(&pl, a.playlistTracks(&pl), 36)
	row := container.NewBorder(nil, nil, container.NewCenter(art), nil, text)

	selected := a.view == viewPlaylist && a.playlistID == pl.ID
	id := pl.ID
	return newTappable(a.pal, container.NewPadded(row), selected, func() { a.open(viewPlaylist, id) })
}

func spacer(h float32) fyne.CanvasObject {
	r := canvas.NewRectangle(nil)
	r.SetMinSize(fyne.NewSize(1, h))
	return r
}

func hspace(w float32) fyne.CanvasObject {
	r := canvas.NewRectangle(nil)
	r.SetMinSize(fyne.NewSize(w, 1))
	return r
}

func (a *App) newPlaylistDialog() {
	a.askName(a.tr.S("pl.new"), a.tr.S("pl.create"), "", func(name string) {
		id, err := a.pls.Create(name)
		if err != nil {
			return
		}
		a.savePlaylists()
		a.open(viewPlaylist, id)
	})
}

func (a *App) askName(title, confirm, initial string, done func(string)) {
	entry := widget.NewEntry()
	entry.SetText(initial)
	entry.SetPlaceHolder(a.tr.S("pl.name"))
	entry.Validator = func(s string) error {
		if len([]rune(s)) == 0 || len(s) > 200 {
			return config.ErrEmptyName
		}
		return nil
	}
	d := dialog.NewForm(title, confirm, a.tr.S("common.cancel"),
		[]*widget.FormItem{widget.NewFormItem(a.tr.S("pl.name"), entry)},
		func(ok bool) {
			if ok {
				done(entry.Text)
			}
		}, a.win)
	d.Resize(fyne.NewSize(420, d.MinSize().Height))
	d.Show()
	a.win.Canvas().Focus(entry)
}
