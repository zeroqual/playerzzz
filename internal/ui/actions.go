package ui

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/zeroqual/playerzzz/internal/library"
	"github.com/zeroqual/playerzzz/internal/trash"
)

func (a *App) showMenuUnder(menu *fyne.Menu, under fyne.CanvasObject) {
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(under)
	widget.ShowPopUpMenuAtPosition(menu, a.win.Canvas(), pos.AddXY(0, under.Size().Height))
}

func (a *App) trackMenu(path string, under fyne.CanvasObject) {
	t := a.tr
	add := fyne.NewMenuItem(t.S("lib.addToPlaylist"), func() { a.addToPlaylistDialog(path) })
	add.Icon = theme.ContentAddIcon()
	del := fyne.NewMenuItem(t.S("lib.delete"), func() { a.confirmDeleteTrack(path) })
	del.Icon = theme.DeleteIcon()
	a.showMenuUnder(fyne.NewMenu("", add, fyne.NewMenuItemSeparator(), del), under)
}

func (a *App) confirmDeleteTrack(path string) {
	t := a.tr
	tr, _ := a.trackFor(path)
	msg := t.F("lib.deleteConfirm", tr.Title)
	if runtime.GOOS == "windows" {
		msg = t.F("lib.deleteConfirmBin", tr.Title)
	}
	a.confirm(t.S("lib.deleteTitle"), msg, t.S("pl.delete"), true, func() { a.deleteTrack(path) })
}

func (a *App) deleteTrack(path string) {
	if a.nowPath == path {
		a.player.Stop()
		a.nowPath = ""
		a.resetProgress()
	}
	if err := trash.Delete(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		a.showErr(err)
		return
	}

	if a.pls.RemoveTrack(path) {
		a.savePlaylists()
	}
	a.removeFromLibrary(path)
	a.removeFromQueue(path)
	a.covers.forget(path)

	a.refreshSidebar()
	a.showView()
	a.refreshNowPlaying()
	a.refreshControls()
}

func (a *App) removeFromLibrary(path string) {
	i, ok := a.byPath[path]
	if !ok {
		return
	}
	a.tracks = append(a.tracks[:i:i], a.tracks[i+1:]...)
	a.reindex()
}

func (a *App) reindex() {
	a.byPath = make(map[string]int, len(a.tracks))
	for i, t := range a.tracks {
		a.byPath[t.Path] = i
	}
}

func (a *App) removeFromQueue(path string) {
	for i := 0; i < len(a.queue); i++ {
		if a.queue[i] != path {
			continue
		}
		a.queue = append(a.queue[:i:i], a.queue[i+1:]...)
		if i < a.qpos {
			a.qpos--
		}
		i--
	}
	if a.qpos >= len(a.queue) {
		a.qpos = 0
	}
}

func (a *App) pruneMissing() {
	folder := filepath.Clean(a.cfg.Folder)
	inFolder := func(p string) bool {
		rel, err := filepath.Rel(folder, p)
		return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
	}
	changed := a.pls.Prune(func(p string) bool {
		if _, ok := a.byPath[p]; ok || !inFolder(p) {
			return false
		}
		_, err := os.Stat(p)
		return errors.Is(err, os.ErrNotExist)
	})
	if changed {
		a.savePlaylists()
	}
}

func (a *App) playlistMenu(id string, under fyne.CanvasObject) {
	t := a.tr
	pl, ok := a.pls.Get(id)
	if !ok {
		return
	}
	rename := fyne.NewMenuItem(t.S("pl.rename"), func() {
		a.askName(t.S("pl.rename"), t.S("common.save"), pl.Name, func(n string) {
			if a.pls.Rename(id, n) == nil {
				a.savePlaylists()
				a.open(viewPlaylist, id)
			}
		})
	})
	rename.Icon = theme.DocumentCreateIcon()

	cover := fyne.NewMenuItem(t.S("pl.cover"), func() { a.pickPlaylistCover(id) })
	cover.Icon = theme.MediaPhotoIcon()

	export := fyne.NewMenuItem(t.S("pl.export"), func() { a.exportDialog(id) })
	export.Icon = theme.DocumentSaveIcon()

	items := []*fyne.MenuItem{rename, cover}
	if pl.Cover != "" {
		remove := fyne.NewMenuItem(t.S("pl.coverRemove"), func() {
			if p, ok := a.pls.Get(id); ok {
				a.removePlaylistCoverFile(p.Cover)
				p.Cover = ""
				a.savePlaylists()
				a.open(viewPlaylist, id)
			}
		})
		remove.Icon = theme.ContentClearIcon()
		items = append(items, remove)
	}

	del := fyne.NewMenuItem(t.S("pl.delete"), func() {
		a.confirm(t.S("pl.delete"), t.F("pl.deleteConfirm", pl.Name), t.S("pl.delete"), true, func() {
			if p, ok := a.pls.Get(id); ok {
				a.removePlaylistCoverFile(p.Cover)
			}
			a.pls.Delete(id)
			a.savePlaylists()
			a.open(viewLibrary, "")
		})
	})
	del.Icon = theme.DeleteIcon()
	items = append(items, export, fyne.NewMenuItemSeparator(), del)

	a.showMenuUnder(fyne.NewMenu("", items...), under)
}

func (a *App) pickPlaylistCover(id string) {
	a.pickImage(func(data []byte) {
		if err := a.setPlaylistCover(id, data); err != nil {
			a.showErr(err)
			return
		}
		a.savePlaylists()
		a.open(viewPlaylist, id)
	})
}

func (a *App) pickImage(done func([]byte)) {
	d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil {
			a.showErr(err)
			return
		}
		if r == nil {
			return
		}
		defer r.Close()
		data, err := io.ReadAll(io.LimitReader(r, 32<<20))
		if err != nil {
			a.showErr(err)
			return
		}
		done(data)
	}, a.win)
	d.SetFilter(storage.NewExtensionFileFilter([]string{".png", ".jpg", ".jpeg", ".gif"}))
	d.Resize(fyne.NewSize(800, 560))
	d.Show()
}

func (a *App) addTracksDialog(id string) {
	t := a.tr
	pl, ok := a.pls.Get(id)
	if !ok {
		return
	}
	var candidates []library.Track
	for _, tr := range a.tracks {
		if !pl.Contains(tr.Path) {
			candidates = append(candidates, tr)
		}
	}
	if len(candidates) == 0 {
		a.alert(t.S("pl.addTracks"), t.S("pl.allAdded"))
		return
	}

	picked := map[string]bool{}
	shown := candidates
	var confirm *widget.Button

	list := widget.NewList(
		func() int { return len(shown) },
		func() fyne.CanvasObject {
			return widget.NewCheck("", nil)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			c := o.(*widget.Check)
			tr := shown[i]
			c.OnChanged = nil
			c.Text = tr.Title + " — " + a.artistOf(tr)
			c.SetChecked(picked[tr.Path])
			c.Refresh()
			path := tr.Path
			c.OnChanged = func(on bool) {
				picked[path] = on
				n := 0
				for _, v := range picked {
					if v {
						n++
					}
				}
				confirm.SetText(t.F("pl.addSelected", n))
				if n > 0 {
					confirm.Enable()
				} else {
					confirm.Disable()
				}
			}
		},
	)

	search := widget.NewEntry()
	search.SetPlaceHolder(t.S("lib.search"))
	search.OnChanged = func(q string) {
		q = strings.ToLower(strings.TrimSpace(q))
		shown = shown[:0:0]
		for _, tr := range candidates {
			if q == "" || strings.Contains(strings.ToLower(tr.Title), q) || strings.Contains(strings.ToLower(tr.Artist), q) {
				shown = append(shown, tr)
			}
		}
		list.Refresh()
	}

	var d *dialog.CustomDialog
	confirm = widget.NewButtonWithIcon(t.F("pl.addSelected", 0), theme.ContentAddIcon(), func() {
		if p, ok := a.pls.Get(id); ok {
			for _, tr := range candidates {
				if picked[tr.Path] {
					p.Add(tr.Path)
				}
			}
			a.savePlaylists()
		}
		d.Hide()
		a.open(viewPlaylist, id)
	})
	confirm.Importance = widget.HighImportance
	confirm.Disable()
	cancel := widget.NewButton(t.S("common.cancel"), func() { d.Hide() })

	content := container.NewBorder(search, nil, nil, nil, list)
	d = dialog.NewCustomWithoutButtons(t.S("pl.addTracks")+" — "+pl.Name, content, a.win)
	d.SetButtons([]fyne.CanvasObject{cancel, confirm})
	d.Resize(fyne.NewSize(560, 560))
	d.Show()
	a.win.Canvas().Focus(search)
}
