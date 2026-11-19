package ui

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"os"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/zeroqual/playerzzz/internal/fsutil"
	"github.com/zeroqual/playerzzz/internal/share"
	"github.com/zeroqual/playerzzz/internal/youtube"
)

func (a *App) shareDialog(id string) {
	t := a.tr
	pl, ok := a.pls.Get(id)
	if !ok {
		return
	}
	sp := share.Playlist{Name: pl.Name}
	downloadable := 0
	for _, tr := range a.playlistTracks(pl) {
		sp.Tracks = append(sp.Tracks, share.Track{Title: tr.Title, Artist: tr.Artist, YouTube: tr.YouTube})
		if tr.YouTube != "" {
			downloadable++
		}
	}
	code, err := share.Encode(sp)
	if err != nil {
		a.showErr(err)
		return
	}

	box := widget.NewMultiLineEntry()
	box.SetText(code)
	box.Wrapping = fyne.TextWrapBreak
	box.SetMinRowsVisible(5)

	box.OnChanged = func(s string) {
		if s != code {
			box.SetText(code)
		}
	}

	copied := muted("")
	copyBtn := widget.NewButtonWithIcon(t.S("share.copy"), theme.ContentCopyIcon(), func() {
		a.win.Clipboard().SetContent(code)
		copied.SetText(t.S("share.copied"))
	})
	copyBtn.Importance = widget.HighImportance

	var d *dialog.CustomDialog
	exportBtn := widget.NewButtonWithIcon(t.S("pl.export"), theme.DocumentSaveIcon(), func() {
		d.Hide()
		a.exportDialog(id)
	})

	content := container.NewVBox(
		hint(t.S("share.hint")),
		box,
		hint(t.F("share.stats", downloadable, len(sp.Tracks))),
		container.NewHBox(copyBtn, copied),
		widget.NewSeparator(),
		hint(t.S("share.orFile")),
		container.NewHBox(exportBtn),
	)
	d = dialog.NewCustom(t.S("share.title")+" — "+pl.Name, t.S("pl.done"), content, a.win)
	d.Resize(fyne.NewSize(580, d.MinSize().Height))
	d.Show()
}

func (a *App) importDialog() {
	t := a.tr
	box := widget.NewMultiLineEntry()
	box.SetPlaceHolder("pzz1:…")
	box.Wrapping = fyne.TextWrapBreak
	box.SetMinRowsVisible(5)

	d := dialog.NewCustomConfirm(t.S("import.title"), t.S("import.next"), t.S("common.cancel"),
		container.NewVBox(hint(t.S("import.paste")), box),
		func(ok bool) {
			if !ok {
				return
			}
			p, err := share.Decode(box.Text)
			if err != nil {
				a.alert(t.S("import.title"), t.S("import.bad"))
				return
			}
			a.importPreview(p.Name, share.MatchTracks(p, a.localTracks()), nil)
		}, a.win)
	d.Resize(fyne.NewSize(560, d.MinSize().Height))
	d.Show()
	a.win.Canvas().Focus(box)
}

func (a *App) localTracks() []share.LocalTrack {
	local := make([]share.LocalTrack, len(a.tracks))
	for i, tr := range a.tracks {
		local[i] = share.LocalTrack{Path: tr.Path, Title: tr.Title, Artist: tr.Artist, YouTube: tr.YouTube}
	}
	return local
}

func (a *App) exportDialog(id string) {
	t := a.tr
	pl, ok := a.pls.Get(id)
	if !ok {
		return
	}
	tracks := a.playlistTracks(pl)

	sizeOf := func(small bool) int64 {
		var n int64
		for _, tr := range tracks {
			if small && tr.YouTube != "" {
				continue
			}
			if st, err := os.Stat(tr.Path); err == nil {
				n += st.Size()
			}
		}
		return n
	}

	size := muted("")
	small := widget.NewCheck(t.S("export.small"), func(on bool) {
		size.SetText(t.F("export.size", a.mib(sizeOf(on), 0)))
	})
	small.SetChecked(false)
	size.SetText(t.F("export.size", a.mib(sizeOf(false), 0)))

	hasYT := false
	for _, tr := range tracks {
		hasYT = hasYT || tr.YouTube != ""
	}
	content := container.NewVBox(hint(t.S("export.hint")))
	if hasYT {
		content.Add(small)
	}
	content.Add(size)

	d := dialog.NewCustomConfirm(t.S("export.title")+" — "+pl.Name, t.S("export.save"), t.S("common.cancel"), content,
		func(ok bool) {
			if ok {
				a.saveExport(id, small.Checked)
			}
		}, a.win)
	d.Resize(fyne.NewSize(560, d.MinSize().Height))
	d.Show()
}

func (a *App) saveExport(id string, small bool) {
	t := a.tr
	pl, ok := a.pls.Get(id)
	if !ok {
		return
	}
	name := pl.Name
	var cover []byte
	if res := a.playlistCover(pl.Cover); res != nil {
		cover = res.Content()
	}
	var tracks []share.ExportTrack
	var total int64
	for _, tr := range a.playlistTracks(pl) {
		include := !(small && tr.YouTube != "")
		tracks = append(tracks, share.ExportTrack{
			Track:   share.Track{Title: tr.Title, Artist: tr.Artist, YouTube: tr.YouTube},
			Path:    tr.Path,
			Include: include,
		})
		if st, err := os.Stat(tr.Path); err == nil && include {
			total += st.Size()
		}
	}

	save := dialog.NewFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil {
			a.showErr(err)
			return
		}
		if w == nil {
			return
		}

		status := widget.NewLabel(t.F("export.progress", a.mib(0, total)))
		bar := widget.NewProgressBar()
		wait := dialog.NewCustomWithoutButtons(t.S("export.title"), container.NewVBox(status, bar), a.win)
		wait.Resize(fyne.NewSize(460, wait.MinSize().Height))
		wait.Show()

		go func() {
			err := share.WritePack(w, name, cover, tracks, func(done int64) {
				fyne.Do(func() {
					status.SetText(t.F("export.progress", a.mib(done, total)))
					if total > 0 {
						bar.SetValue(float64(done) / float64(total))
					}
				})
			})
			if cerr := w.Close(); err == nil {
				err = cerr
			}
			path := w.URI().Path()
			fyne.Do(func() {
				wait.Hide()
				if err != nil {
					_ = os.Remove(path)
					a.showErr(err)
					return
				}
				a.alert(t.S("export.title"), t.F("export.done", path))
			})
		}()
	}, a.win)
	save.SetFileName(fsutil.SafeName(name, "playlist") + share.PackExt)
	save.SetFilter(storage.NewExtensionFileFilter([]string{share.PackExt}))
	save.Resize(fyne.NewSize(800, 560))
	save.Show()
}

func (a *App) importFileDialog() {
	t := a.tr
	open := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil {
			a.showErr(err)
			return
		}
		if r == nil {
			return
		}
		defer r.Close()

		var pack *share.Pack
		if r.URI().Scheme() == "file" {
			pack, err = share.OpenPack(r.URI().Path())
		} else {
			var data []byte
			if data, err = io.ReadAll(r); err == nil {
				pack, err = share.ReadPack(bytes.NewReader(data), int64(len(data)))
			}
		}
		if err != nil {
			a.alert(t.S("import.title"), t.S("import.badFile"))
			return
		}
		a.importPreview(pack.Manifest.Name, share.MatchPack(pack, a.localTracks()), pack)
	}, a.win)
	open.SetFilter(storage.NewExtensionFileFilter([]string{share.PackExt}))
	open.Resize(fyne.NewSize(800, 560))
	open.Show()
}

func (a *App) importPreview(name string, matches []share.Match, pack *share.Pack) {
	t := a.tr
	var found, packed, toDownload, missing int
	for _, m := range matches {
		switch m.Status {
		case share.Found:
			found++
		case share.Packed:
			packed++
		case share.Download:
			toDownload++
		case share.Missing:
			missing++
		}
	}

	download := widget.NewCheck(t.S("import.doDownload"), nil)
	download.SetChecked(true)
	rows := container.NewVBox(
		heading(name, theme.SizeNameSubHeadingText),
		widget.NewLabel(t.Tracks(len(matches))),
		muted(t.F("import.found", found)),
	)
	if pack != nil {
		rows.Add(muted(t.F("import.packed", packed)))
	}
	rows.Add(muted(t.F("import.download", toDownload)))
	rows.Add(muted(t.F("import.missing", missing)))

	needFolder := (packed > 0 || toDownload > 0) && a.cfg.Folder == ""
	if needFolder {
		rows.Add(hint(t.S("yt.needFolder")))
	} else if toDownload > 0 {
		rows.Add(download)
	}

	closePack := func() {
		if pack != nil {
			_ = pack.Close()
		}
	}
	d := dialog.NewCustomConfirm(t.S("import.title"), t.S("import.go"), t.S("common.cancel"), rows,
		func(ok bool) {
			if !ok {
				closePack()
				return
			}
			a.runImport(name, matches, pack, download.Checked && !needFolder, !needFolder)
		}, a.win)
	d.Resize(fyne.NewSize(460, d.MinSize().Height))
	d.Show()
}

func (a *App) runImport(name string, matches []share.Match, pack *share.Pack, download, canWrite bool) {
	t := a.tr
	slots := make([]string, len(matches))
	var work []int
	for i, m := range matches {
		switch {
		case m.Status == share.Found:
			slots[i] = m.Path
		case m.Status == share.Packed && canWrite:
			work = append(work, i)
		case m.Status == share.Download && download:
			work = append(work, i)
		}
	}
	var cover []byte
	if pack != nil {
		cover = pack.Cover()
	}

	finish := func() {
		if pack != nil {
			_ = pack.Close()
		}
		id, err := a.pls.Create(a.pls.UniqueName(name))
		if err != nil {
			a.showErr(err)
			return
		}
		pl, _ := a.pls.Get(id)
		added := 0
		for _, path := range slots {
			if path != "" {
				pl.Add(path)
				added++
			}
		}
		if len(cover) > 0 {
			if err := a.setPlaylistCover(id, cover); err != nil {
				log.Printf("import: обложка: %v", err)
			}
		}
		a.savePlaylists()
		a.open(viewPlaylist, id)
		a.alert(t.S("import.title"), t.F("import.done", added, len(matches)))
	}
	if len(work) == 0 {
		finish()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	status := widget.NewLabel("")
	bar := widget.NewProgressBar()
	wait := dialog.NewCustom(t.S("import.title"), t.S("common.cancel"), container.NewVBox(status, bar), a.win)
	wait.SetOnClosed(cancel)
	wait.Resize(fyne.NewSize(460, wait.MinSize().Height))
	wait.Show()

	dir := a.cfg.Folder
	go func() {
		defer cancel()
		var last atomic.Int64
		for n, i := range work {
			if ctx.Err() != nil {
				break
			}
			m := matches[i]
			fyne.Do(func() {
				key := "import.progress"
				if m.Status == share.Packed {
					key = "import.copying"
				}
				status.SetText(t.F(key, n+1, len(work)))
				bar.SetValue(float64(n) / float64(len(work)))
			})

			var path string
			var err error
			if m.Status == share.Packed {
				path = fsutil.UniquePath(dir, fsutil.SafeName(m.Track.Title, "track"), ".mp3")
				err = pack.ExtractTrack(i, path)
			} else {
				link := "https://www.youtube.com/watch?v=" + m.Track.YouTube
				path, err = youtube.Download(ctx, link, dir, func(done, total int64) {
					now := time.Now().UnixMilli()
					if total <= 0 || now-last.Load() < 100 {
						return
					}
					last.Store(now)
					fyne.Do(func() {
						bar.SetValue((float64(n) + float64(done)/float64(total)) / float64(len(work)))
					})
				})
			}
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					log.Printf("import: %q: %v", m.Track.Title, err)
				}
				continue
			}
			slots[i] = path
		}
		fyne.Do(func() {
			wait.Hide()

			a.loadFolderThen(dir, finish)
		})
	}()
}
