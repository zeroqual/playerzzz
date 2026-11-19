package ui

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/zeroqual/playerzzz/internal/youtube"
)

func (a *App) showYouTubeDialog() {
	t := a.tr
	if a.cfg.Folder == "" {
		a.alert(t.S("yt.title"), t.S("yt.needFolder"))
		return
	}

	link := widget.NewEntry()
	link.SetPlaceHolder("https://www.youtube.com/watch?v=…")
	link.Validator = func(s string) error {
		if err := youtube.ValidateURL(s); err != nil {
			return errors.New(a.errText(err))
		}
		return nil
	}

	h := hint(t.F("yt.hint", int(youtube.MaxDuration.Minutes())))

	form := dialog.NewForm(t.S("yt.title"), t.S("yt.download"), t.S("common.cancel"),
		[]*widget.FormItem{
			widget.NewFormItem(t.S("yt.link"), link),
			widget.NewFormItem("", h),
		},
		func(ok bool) {
			if ok {
				a.downloadFromYouTube(link.Text)
			}
		}, a.win)
	form.Resize(fyne.NewSize(580, form.MinSize().Height))
	form.Show()
	a.win.Canvas().Focus(link)
}

func (a *App) downloadFromYouTube(link string) {
	t := a.tr
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	status := widget.NewLabel(t.S("yt.probing"))
	bar := widget.NewProgressBar()
	wait := dialog.NewCustom(t.S("yt.loading"), t.S("common.cancel"), container.NewVBox(status, bar), a.win)
	wait.SetOnClosed(cancel)
	wait.Resize(fyne.NewSize(460, wait.MinSize().Height))
	wait.Show()

	var lastUpdate atomic.Int64
	progress := func(done, total int64) {
		now := time.Now().UnixMilli()
		if now-lastUpdate.Load() < 100 {
			return
		}
		lastUpdate.Store(now)
		fyne.Do(func() {
			status.SetText(t.F("yt.progress", a.mib(done, total)))
			if total > 0 {
				bar.SetValue(float64(done) / float64(total))
			}
		})
	}

	dir := a.cfg.Folder
	go func() {
		defer cancel()
		path, err := youtube.Download(ctx, link, dir, progress)

		userCanceled := errors.Is(ctx.Err(), context.Canceled)

		fyne.Do(func() {
			wait.Hide()
			if err != nil {
				if !userCanceled {
					a.showErr(err)
				}
				return
			}
			a.loadFolder(dir, path)
		})
	}()
}

func (a *App) mib(done, total int64) string {
	const m = 1 << 20
	if total <= 0 {
		return a.tr.F("yt.mb", float64(done)/m)
	}
	return a.tr.F("yt.mbOf", float64(done)/m, float64(total)/m)
}

func (a *App) errText(err error) string {
	t := a.tr
	minutes := int(youtube.MaxDuration.Minutes())
	switch {
	case errors.Is(err, youtube.ErrBadURL):
		return t.S("err.badURL")
	case errors.Is(err, youtube.ErrTooLong):
		return t.F("err.tooLong", minutes)
	case errors.Is(err, youtube.ErrNoDuration):
		return t.S("err.noDuration")
	case errors.Is(err, youtube.ErrBlocked):
		return t.S("err.blocked")
	case errors.Is(err, errBadImage):
		return t.S("settings.avatarBadFile")
	}
	return err.Error()
}
