package ui

import (
	"log"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/zeroqual/playerzzz/internal/audio"
	"github.com/zeroqual/playerzzz/internal/config"
	"github.com/zeroqual/playerzzz/internal/i18n"
	"github.com/zeroqual/playerzzz/internal/library"
)

const appVersion = "0.3.0"

type viewKind int

const (
	viewLibrary viewKind = iota
	viewPlaylist
	viewSettings
)

type App struct {
	fyneApp fyne.App
	win     fyne.Window
	player  *audio.Player

	store *config.Store
	cfg   config.Settings
	pls   *config.Playlists

	tr  *i18n.T
	pal Palette

	covers        *coverCache
	coversPending bool
	avatarRes     fyne.Resource
	avatarVer     int

	hwnd        uintptr
	customFrame bool
	maxBtn      *winButton

	tracks []library.Track
	byPath map[string]int

	view       viewKind
	playlistID string
	search     string

	shown     []library.Track
	list      *widget.List
	noResults *widget.Label

	queue   []string
	qpos    int
	nowPath string

	sidebarBox *fyne.Container
	mainBox    *fyne.Container
	bar        playerBar

	updatingProgress bool
	lastUserSeek     time.Time
}

func Run() error {
	store, err := config.Open("playerzzz")
	if err != nil {
		return err
	}
	cfg, err := store.LoadSettings()
	if err != nil {
		log.Printf("settings: %v (используем настройки по умолчанию)", err)
	}
	pls, err := store.LoadPlaylists()
	if err != nil {
		log.Printf("playlists: %v", err)
	}

	p, err := audio.New()
	if err != nil {
		return err
	}
	defer p.Close()

	a := &App{
		fyneApp: app.NewWithID("io.github.zeroqual.playerzzz"),
		player:  p,
		store:   store,
		cfg:     cfg,
		pls:     pls,
		covers:  newCoverCache(),
		byPath:  map[string]int{},
	}
	a.applyLook()
	a.fyneApp.SetIcon(appIcon)
	a.win = a.fyneApp.NewWindow("playerzzz")
	a.win.SetIcon(appIcon)
	a.win.Resize(fyne.NewSize(1280, 800))

	a.fyneApp.Lifecycle().SetOnStarted(a.enableFrame)

	p.OnEnd = func() { fyne.Do(a.next) }
	p.SetVolume(cfg.Volume)

	a.rebuild()
	if st, err := os.Stat(cfg.Folder); cfg.Folder != "" && err == nil && st.IsDir() {
		a.loadFolder(cfg.Folder, "")
	}

	go a.tick()
	go a.frameTick()
	a.win.ShowAndRun()
	return nil
}

func (a *App) applyLook() {
	a.tr = i18n.New(a.cfg.Language)
	a.pal = paletteByID(a.cfg.Theme)
	a.fyneApp.Settings().SetTheme(playerTheme{a.pal})
}

func (a *App) rebuild() {
	a.sidebarBox = container.NewStack()
	a.mainBox = container.NewStack()
	a.refreshSidebar()
	a.showView()

	line := canvas.NewRectangle(a.pal.Sep)
	line.SetMinSize(fyne.NewSize(1, 1))
	left := container.NewHBox(withWidth(248, a.sidebarBox), line)

	mainBg := canvas.NewRectangle(a.pal.Bg)
	body := container.NewBorder(nil, a.buildPlayerBar(), left, nil, container.NewStack(mainBg, a.mainBox))
	if a.customFrame {
		body = container.NewBorder(a.buildTitleBar(), nil, nil, nil, body)
	}
	a.win.SetContent(body)

	a.refreshNowPlaying()
	a.refreshControls()
	a.updateProgress()
}

func (a *App) open(v viewKind, playlistID string) {
	a.view, a.playlistID = v, playlistID
	a.refreshSidebar()
	a.showView()
}

func (a *App) showView() {
	var content fyne.CanvasObject
	switch a.view {
	case viewPlaylist:
		if _, ok := a.pls.Get(a.playlistID); ok {
			content = a.buildPlaylistView()
			break
		}
		a.view = viewLibrary
		content = a.buildLibraryView()
	case viewSettings:
		a.list = nil
		content = a.buildSettingsView()
	default:
		content = a.buildLibraryView()
	}
	a.mainBox.Objects = []fyne.CanvasObject{content}
	a.mainBox.Refresh()
	a.selectCurrent()
}

func (a *App) pickFolder() {
	d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			a.showErr(err)
			return
		}
		if uri == nil {
			return
		}
		a.loadFolder(uri.Path(), "")
	}, a.win)
	d.Resize(fyne.NewSize(800, 560))
	d.Show()
}

func (a *App) loadFolder(dir, playPath string) {
	a.loadFolderThen(dir, func() {
		if i, ok := a.byPath[playPath]; ok && playPath != "" {
			a.playFrom(a.libraryPaths(), i)
		}
	})
}

func (a *App) loadFolderThen(dir string, after func()) {
	go func() {
		tracks, err := library.Scan(dir)
		fyne.Do(func() {
			if err != nil {
				a.showErr(err)
				return
			}
			a.tracks = tracks
			a.reindex()
			if a.cfg.Folder != dir {
				a.cfg.Folder = dir
				a.saveSettings()
			}
			a.pruneMissing()
			a.refreshSidebar()
			a.showView()
			a.refreshNowPlaying()
			a.refreshControls()
			if after != nil {
				after()
			}
		})
	}()
}

func (a *App) libraryPaths() []string {
	paths := make([]string, len(a.tracks))
	for i, t := range a.tracks {
		paths[i] = t.Path
	}
	return paths
}

func (a *App) trackFor(path string) (library.Track, bool) {
	if i, ok := a.byPath[path]; ok {
		return a.tracks[i], true
	}
	if _, err := os.Stat(path); err != nil {
		return library.Track{}, false
	}
	return library.Load(path), true
}

func (a *App) artistOf(t library.Track) string {
	if t.Artist == "" {
		return a.tr.S("lib.unknown")
	}
	return t.Artist
}

func (a *App) playFrom(paths []string, i int) {
	a.queue = paths
	a.playAt(i)
}

func (a *App) playAt(i int) {
	if i < 0 || i >= len(a.queue) {
		return
	}
	path := a.queue[i]
	if err := a.player.Play(path); err != nil {
		a.showErr(err)
		return
	}
	a.qpos, a.nowPath = i, path
	a.refreshNowPlaying()
	a.refreshList()
	a.selectCurrent()
	a.refreshControls()
	a.updateProgress()
}

func (a *App) togglePlay() {
	if a.nowPath == "" {
		if len(a.shown) > 0 {
			a.playFrom(pathsOf(a.shown), 0)
		}
		return
	}
	a.player.TogglePause()
	a.refreshControls()
}

func (a *App) next() {
	if len(a.queue) == 0 {
		a.togglePlay()
		return
	}
	a.playAt((a.qpos + 1) % len(a.queue))
}

func (a *App) prev() {
	if pos, _ := a.player.Progress(); pos > 3*time.Second || len(a.queue) == 0 {
		_ = a.player.Seek(0)
		a.updateProgress()
		return
	}
	n := len(a.queue)
	a.playAt((a.qpos - 1 + n) % n)
}

func (a *App) selectCurrent() {
	if a.list == nil {
		return
	}
	for i, t := range a.shown {
		if t.Path == a.nowPath {
			a.list.Select(i)
			return
		}
	}
	a.list.UnselectAll()
}

func (a *App) refreshList() {
	if a.list != nil {
		a.list.Refresh()
	}
}

func (a *App) coversReady() {
	if a.coversPending {
		return
	}
	a.coversPending = true
	time.AfterFunc(150*time.Millisecond, func() {
		fyne.Do(func() {
			a.coversPending = false
			a.refreshList()
			a.refreshNowPlaying()
			a.refreshSidebar()
		})
	})
}

func pathsOf(ts []library.Track) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.Path
	}
	return out
}

func (a *App) saveSettings() {
	if err := a.store.SaveSettings(a.cfg); err != nil {
		log.Printf("settings: %v", err)
	}
}

func (a *App) savePlaylists() {
	if err := a.store.SavePlaylists(a.pls); err != nil {
		a.showErr(err)
	}
}

func (a *App) nickname() string {
	if a.cfg.Profile.Nickname != "" {
		return a.cfg.Profile.Nickname
	}
	return a.tr.S("profile.default")
}

func (a *App) showErr(err error) {
	if err == nil {
		return
	}
	a.alert(a.tr.S("common.error"), a.errText(err))
}

func (a *App) alert(title, msg string) {
	l := widget.NewLabel(msg)
	l.Wrapping = fyne.TextWrapWord
	d := dialog.NewCustom(title, "OK", l, a.win)
	d.Resize(fyne.NewSize(420, d.MinSize().Height))
	d.Show()
}

func (a *App) confirm(title, msg, yes string, danger bool, onYes func()) {
	l := widget.NewLabel(msg)
	l.Wrapping = fyne.TextWrapWord
	d := dialog.NewCustomConfirm(title, yes, a.tr.S("common.cancel"), l, func(ok bool) {
		if ok {
			onYes()
		}
	}, a.win)
	if danger {
		d.SetConfirmImportance(widget.DangerImportance)
	}
	d.Resize(fyne.NewSize(440, d.MinSize().Height))
	d.Show()
}
