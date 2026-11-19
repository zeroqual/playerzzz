package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type playerBar struct {
	cover    *fyne.Container
	title    *widget.Label
	artist   *widget.Label
	add      *widget.Button
	play     *widget.Button
	prev     *widget.Button
	next     *widget.Button
	progress *widget.Slider
	pos      *widget.Label
	length   *widget.Label
	volume   *widget.Slider
}

func (a *App) buildPlayerBar() fyne.CanvasObject {
	b := &a.bar

	b.cover = container.NewStack()
	b.title = ellipsis(widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	b.artist = ellipsis(muted(""))
	b.add = widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		if a.nowPath != "" {
			a.addToPlaylistDialog(a.nowPath)
		}
	})
	b.add.Importance = widget.LowImportance
	text := container.New(layout.NewCustomPaddedVBoxLayout(-14), b.title, b.artist)
	left := withWidth(320, container.NewBorder(nil, nil,
		container.NewCenter(b.cover), container.NewCenter(b.add),
		container.NewVBox(layout.NewSpacer(), text, layout.NewSpacer())))

	b.prev = widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), a.prev)
	b.prev.Importance = widget.LowImportance
	b.play = widget.NewButtonWithIcon("", theme.MediaPlayIcon(), a.togglePlay)
	b.play.Importance = widget.HighImportance
	b.next = widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), a.next)
	b.next.Importance = widget.LowImportance
	buttons := container.NewCenter(container.NewHBox(b.prev, b.play, b.next))

	b.progress = widget.NewSlider(0, 1)
	b.progress.Step = 0.1
	b.progress.OnChanged = func(v float64) {
		if a.updatingProgress {
			return
		}
		a.lastUserSeek = time.Now()
		b.pos.SetText(fmtDur(secs(v)))
	}
	b.progress.OnChangeEnded = func(v float64) {
		if a.updatingProgress {
			return
		}
		a.lastUserSeek = time.Now()
		if err := a.player.Seek(secs(v)); err != nil {
			a.showErr(err)
		}
	}
	b.pos = muted("0:00")
	b.length = muted("0:00")
	progress := container.NewThemeOverride(b.progress, sliderTheme{playerTheme{a.pal}, a.pal.Accent})
	center := container.NewVBox(buttons, container.NewBorder(nil, nil, b.pos, b.length, progress))

	b.volume = widget.NewSlider(0, 1)
	b.volume.Step = 0.01
	b.volume.Value = a.cfg.Volume
	b.volume.OnChanged = func(v float64) { a.player.SetVolume(v) }
	b.volume.OnChangeEnded = func(v float64) {
		a.cfg.Volume = v
		a.saveSettings()
	}
	right := withWidth(200, container.NewCenter(container.NewBorder(nil, nil,
		widget.NewIcon(theme.VolumeUpIcon()), nil, fixedSize(150, b.volume.MinSize().Height,
			container.NewThemeOverride(b.volume, sliderTheme{playerTheme{a.pal}, a.pal.Text})))))

	bar := container.NewBorder(nil, nil, left, right, center)

	bg := canvas.NewRectangle(a.pal.Panel)
	top := canvas.NewRectangle(a.pal.Border)
	top.SetMinSize(fyne.NewSize(1, 1))
	return container.NewBorder(top, nil, nil, nil,
		container.NewStack(bg, container.NewPadded(container.NewPadded(bar))))
}

func (a *App) refreshNowPlaying() {
	b := &a.bar
	if b.title == nil {
		return
	}
	if a.nowPath == "" {
		b.title.SetText(a.tr.S("player.nothing"))
		b.artist.SetText("")
		b.cover.Objects = []fyne.CanvasObject{a.coverView(nil, 56, 10)}
		b.cover.Refresh()
		b.add.Hide()
		return
	}
	t, _ := a.trackFor(a.nowPath)
	b.title.SetText(t.Title)
	b.artist.SetText(a.artistOf(t))
	b.cover.Objects = []fyne.CanvasObject{a.coverView(a.covers.get(t, a.coversReady), 56, 10)}
	b.cover.Refresh()
	b.add.Show()
}

func (a *App) refreshControls() {
	b := &a.bar
	if b.play == nil {
		return
	}
	if a.player.Playing() {
		b.play.SetIcon(theme.MediaPauseIcon())
	} else {
		b.play.SetIcon(theme.MediaPlayIcon())
	}
	can := len(a.queue) > 0 || len(a.shown) > 0
	for _, btn := range []*widget.Button{b.play, b.prev, b.next} {
		if can {
			btn.Enable()
		} else {
			btn.Disable()
		}
	}
	if a.nowPath == "" {
		b.progress.Disable()
	} else {
		b.progress.Enable()
	}
}

func (a *App) tick() {
	for range time.Tick(250 * time.Millisecond) {
		fyne.Do(a.updateProgress)
	}
}

func (a *App) updateProgress() {
	b := &a.bar
	if b.progress == nil {
		return
	}
	pos, length := a.player.Progress()
	if length <= 0 {
		return
	}

	if time.Since(a.lastUserSeek) < time.Second {
		return
	}
	a.updatingProgress = true
	b.progress.Max = length.Seconds()
	b.progress.SetValue(pos.Seconds())
	a.updatingProgress = false

	b.pos.SetText(fmtDur(pos))
	b.length.SetText(fmtDur(length))
}

func (a *App) resetProgress() {
	b := &a.bar
	if b.progress == nil {
		return
	}
	a.updatingProgress = true
	b.progress.SetValue(0)
	a.updatingProgress = false
	b.pos.SetText("0:00")
	b.length.SetText("0:00")
}

func secs(v float64) time.Duration { return time.Duration(v * float64(time.Second)) }

func fmtDur(d time.Duration) string {
	s := int(d.Round(time.Second).Seconds())
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}
