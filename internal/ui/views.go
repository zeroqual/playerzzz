package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/zeroqual/playerzzz/internal/config"
	"github.com/zeroqual/playerzzz/internal/library"
)

type rowAction int

const (
	actionAddToPlaylist rowAction = iota
	actionRemove
)

type trackRow struct {
	num    *widget.Label
	eq     *widget.Icon
	cover  *fyne.Container
	title  *widget.Label
	artist *widget.Label
	action *widget.Button
}

func (a *App) buildTrackList(action rowAction) *widget.List {
	rows := map[fyne.CanvasObject]*trackRow{}

	list := widget.NewList(
		func() int { return len(a.shown) },
		func() fyne.CanvasObject {
			r := &trackRow{
				num:    muted("00"),
				eq:     widget.NewIcon(theme.NewPrimaryThemedResource(theme.MediaPlayIcon())),
				cover:  container.NewStack(),
				title:  ellipsis(widget.NewLabel("")),
				artist: ellipsis(muted("")),
			}
			r.eq.Hide()
			icon := theme.MoreHorizontalIcon()
			if action == actionRemove {
				icon = theme.ContentClearIcon()
			}
			r.action = widget.NewButtonWithIcon("", icon, nil)
			r.action.Importance = widget.LowImportance

			left := container.NewHBox(
				fixedSize(32, 44, container.NewCenter(container.NewStack(r.num, r.eq))),
				container.NewCenter(r.cover),
			)
			text := container.New(layout.NewCustomPaddedVBoxLayout(-14), r.title, r.artist)
			root := container.NewBorder(nil, nil, left, container.NewCenter(r.action), text)
			rows[root] = r
			return root
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			r, ok := rows[o]
			if !ok || id >= len(a.shown) {
				return
			}
			t := a.shown[id]
			r.num.SetText(fmt.Sprint(id + 1))
			r.title.SetText(t.Title)
			r.artist.SetText(a.artistOf(t))

			playing := t.Path == a.nowPath
			if playing {
				r.num.Hide()
				r.eq.Show()
				r.title.Importance = widget.HighImportance
				r.title.TextStyle.Bold = true
			} else {
				r.eq.Hide()
				r.num.Show()
				r.title.Importance = widget.MediumImportance
				r.title.TextStyle.Bold = false
			}
			r.title.Refresh()

			r.cover.Objects = []fyne.CanvasObject{a.coverView(a.covers.get(t, a.coversReady), 44, 8)}
			r.cover.Refresh()

			path := t.Path
			if action == actionRemove {
				r.action.OnTapped = func() { a.removeFromPlaylist(path) }
			} else {
				btn := r.action
				r.action.OnTapped = func() { a.trackMenu(path, btn) }
			}
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		if id < len(a.shown) && a.shown[id].Path != a.nowPath {
			a.playFrom(pathsOf(a.shown), id)
		}
	}
	return list
}

func (a *App) listHeader() fyne.CanvasObject {
	return container.NewVBox(
		container.NewBorder(nil, nil, fixedSize(32+44+8, 20, container.NewCenter(a.caption("#"))), nil,
			container.NewHBox(container.NewCenter(a.caption(a.tr.S("lib.colTitle"))))),
		widget.NewSeparator(),
	)
}

func (a *App) buildLibraryView() fyne.CanvasObject {
	t := a.tr
	if len(a.tracks) == 0 {
		a.list, a.shown = nil, nil
		return a.buildEmptyState()
	}

	title := heading(t.S("lib.title"), theme.SizeNameHeadingText)
	count := muted(t.Tracks(len(a.tracks)))

	search := widget.NewEntry()
	search.SetPlaceHolder(t.S("lib.search"))
	search.ActionItem = widget.NewIcon(theme.NewColoredResource(theme.SearchIcon(), theme.ColorNameDisabled))
	search.SetText(a.search)
	search.OnChanged = func(s string) {
		a.search = s
		a.applyFilter()
	}

	yt := widget.NewButtonWithIcon(t.S("lib.youtube"), theme.DownloadIcon(), a.showYouTubeDialog)
	yt.Importance = widget.HighImportance

	actions := container.NewHBox(fixedSize(280, search.MinSize().Height, search), yt)
	header := container.NewBorder(nil, nil, nil, container.NewVBox(layout.NewSpacer(), actions),
		container.New(layout.NewCustomPaddedVBoxLayout(-8), title, count))

	noResults := widget.NewLabelWithStyle(t.S("lib.noResults"), fyne.TextAlignCenter, fyne.TextStyle{})
	noResults.Importance = widget.LowImportance
	a.noResults = noResults

	a.list = a.buildTrackList(actionAddToPlaylist)
	a.applyFilter()

	return container.NewPadded(container.NewPadded(container.NewBorder(
		container.NewVBox(header, spacer(12), a.listHeader()), nil, nil, nil,
		container.NewStack(a.list, container.NewCenter(noResults)),
	)))
}

func (a *App) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(a.search))
	a.shown = a.shown[:0:0]
	for _, t := range a.tracks {
		if q == "" || strings.Contains(strings.ToLower(t.Title), q) || strings.Contains(strings.ToLower(t.Artist), q) {
			a.shown = append(a.shown, t)
		}
	}
	if a.list != nil {
		a.list.UnselectAll()
		a.list.Refresh()
		a.selectCurrent()
	}
	if a.noResults != nil {
		if len(a.shown) == 0 {
			a.noResults.Show()
		} else {
			a.noResults.Hide()
		}
	}
}

func (a *App) buildEmptyState() fyne.CanvasObject {
	t := a.tr
	iconBg := canvas.NewRectangle(withAlpha(a.pal.Accent, 0x22))
	iconBg.CornerRadius = 24
	icon := widget.NewIcon(theme.NewPrimaryThemedResource(theme.MediaMusicIcon()))
	badge := fixedSize(88, 88, container.NewStack(iconBg, container.NewPadded(container.NewPadded(icon))))

	head := widget.NewLabelWithStyle(t.S("empty.title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	head.SizeName = theme.SizeNameSubHeadingText
	text := widget.NewLabelWithStyle(t.S("empty.text"), fyne.TextAlignCenter, fyne.TextStyle{})
	text.Importance = widget.LowImportance

	btn := widget.NewButtonWithIcon(t.S("empty.pick"), theme.FolderOpenIcon(), a.pickFolder)
	btn.Importance = widget.HighImportance

	return container.NewCenter(container.NewVBox(container.NewCenter(badge), head, text, container.NewCenter(btn)))
}

func (a *App) playlistTracks(pl *config.Playlist) []library.Track {
	var out []library.Track
	for _, p := range pl.Tracks {
		if t, ok := a.trackFor(p); ok {
			out = append(out, t)
		}
	}
	return out
}

func (a *App) buildPlaylistView() fyne.CanvasObject {
	t := a.tr
	pl, _ := a.pls.Get(a.playlistID)
	a.shown = a.playlistTracks(pl)
	a.noResults = nil

	label := a.caption(t.S("pl.label"))
	name := heading(pl.Name, theme.SizeNameHeadingText)
	count := muted(t.Tracks(len(a.shown)))

	play := widget.NewButtonWithIcon(t.S("pl.play"), theme.MediaPlayIcon(), func() {
		if len(a.shown) > 0 {
			a.playFrom(pathsOf(a.shown), 0)
		}
	})
	play.Importance = widget.HighImportance
	if len(a.shown) == 0 {
		play.Disable()
	}
	id := pl.ID
	add := widget.NewButtonWithIcon(t.S("pl.addTracks"), theme.ContentAddIcon(), func() { a.addTracksDialog(id) })
	share := widget.NewButtonWithIcon(t.S("pl.share"), theme.MailForwardIcon(), func() { a.shareDialog(id) })
	more := widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), nil)
	more.OnTapped = func() { a.playlistMenu(id, more) }

	label = container.NewHBox(hspace(theme.InnerPadding()), label)
	info := container.NewVBox(layout.NewSpacer(), label,
		container.New(layout.NewCustomPaddedVBoxLayout(-10), name, count),
		container.NewHBox(play, add, share, more))
	header := container.NewBorder(nil, nil, container.NewHBox(a.playlistArt(pl, a.shown, 168), hspace(16)), nil, info)

	var body fyne.CanvasObject
	if len(a.shown) == 0 {
		a.list = nil
		empty := widget.NewLabelWithStyle(t.S("pl.empty"), fyne.TextAlignCenter, fyne.TextStyle{})
		empty.Importance = widget.LowImportance
		body = container.NewCenter(empty)
	} else {
		a.list = a.buildTrackList(actionRemove)
		body = container.NewBorder(a.listHeader(), nil, nil, nil, a.list)
	}

	return container.NewPadded(container.NewPadded(container.NewBorder(
		container.NewVBox(header, spacer(16)), nil, nil, nil, body)))
}

func (a *App) playlistArt(pl *config.Playlist, ts []library.Track, size float32) fyne.CanvasObject {
	radius := size / 10
	if res := a.playlistCover(pl.Cover); res != nil {
		img := canvas.NewImageFromResource(res)
		img.FillMode = canvas.ImageFillCover
		img.CornerRadius = radius
		img.ScaleMode = canvas.ImageScaleSmooth
		return fixedSize(size, size, img)
	}
	var covers []fyne.Resource
	for _, t := range ts {
		if res := a.covers.get(t, a.coversReady); res != nil {
			covers = append(covers, res)
			if len(covers) == 4 {
				break
			}
		}
	}
	switch {
	case len(covers) == 0:
		return a.coverView(nil, size, radius)
	case len(covers) < 4:
		return a.coverView(covers[0], size, radius)
	}
	res := a.covers.mosaic(covers)
	if res == nil {
		return a.coverView(covers[0], size, radius)
	}
	return a.coverView(res, size, radius)
}

func (a *App) removeFromPlaylist(path string) {
	pl, ok := a.pls.Get(a.playlistID)
	if !ok {
		return
	}
	pl.Remove(path)
	a.savePlaylists()
	a.open(viewPlaylist, pl.ID)
}

func (a *App) addToPlaylistDialog(path string) {
	t := a.tr
	checks := container.NewVBox()
	addCheck := func(id string) {
		pl, _ := a.pls.Get(id)
		c := widget.NewCheck(pl.Name, func(on bool) {
			p, ok := a.pls.Get(id)
			if !ok {
				return
			}
			if on {
				p.Add(path)
			} else {
				p.Remove(path)
			}
			a.savePlaylists()
			a.refreshSidebar()
		})
		c.SetChecked(pl.Contains(path))
		checks.Add(c)
	}
	for _, pl := range a.pls.Items {
		addCheck(pl.ID)
	}

	newName := widget.NewEntry()
	newName.SetPlaceHolder(t.S("pl.new"))
	create := widget.NewButton(t.S("pl.create"), nil)
	create.OnTapped = func() {
		id, err := a.pls.Create(newName.Text)
		if err != nil {
			return
		}
		pl, _ := a.pls.Get(id)
		pl.Add(path)
		a.savePlaylists()
		a.refreshSidebar()
		addCheck(id)
		newName.SetText("")
	}
	newName.OnSubmitted = func(string) { create.OnTapped() }

	tr, _ := a.trackFor(path)
	head := container.NewBorder(nil, nil, a.coverView(a.covers.get(tr, func() {}), 44, 8), nil,
		container.New(layout.NewCustomPaddedVBoxLayout(-14),
			ellipsis(widget.NewLabelWithStyle(tr.Title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})),
			ellipsis(muted(a.artistOf(tr)))))

	content := container.NewBorder(
		container.NewVBox(head, widget.NewSeparator()),
		container.NewBorder(nil, nil, nil, create, newName),
		nil, nil,
		container.NewVScroll(checks),
	)
	d := dialog.NewCustom(t.S("pl.addTitle"), t.S("pl.done"), content, a.win)
	d.Resize(fyne.NewSize(440, 480))
	d.Show()
}
