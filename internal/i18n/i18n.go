package i18n

import "fmt"

const (
	EN = "en"
	RU = "ru"
)

var Languages = []struct{ Code, Name string }{
	{EN, "English"},
	{RU, "Русский"},
}

type T struct{ lang string }

func New(lang string) *T {
	if _, ok := dict[lang]; !ok {
		lang = EN
	}
	return &T{lang: lang}
}

func (t *T) Lang() string { return t.lang }

func (t *T) S(key string) string {
	if s, ok := dict[t.lang][key]; ok {
		return s
	}
	if s, ok := dict[EN][key]; ok {
		return s
	}
	return key
}

func (t *T) F(key string, args ...any) string { return fmt.Sprintf(t.S(key), args...) }

func (t *T) Tracks(n int) string {
	if t.lang == RU {
		return fmt.Sprintf("%d %s", n, ruPlural(n, "трек", "трека", "треков"))
	}
	if n == 1 {
		return "1 track"
	}
	return fmt.Sprintf("%d tracks", n)
}

func ruPlural(n int, one, few, many string) string {
	n %= 100
	if n >= 11 && n <= 14 {
		return many
	}
	switch n % 10 {
	case 1:
		return one
	case 2, 3, 4:
		return few
	}
	return many
}

var dict = map[string]map[string]string{
	EN: {
		"nav.library":   "Library",
		"nav.settings":  "Settings",
		"nav.playlists": "PLAYLISTS",
		"nav.newList":   "New playlist",

		"profile.local":   "Local profile",
		"profile.default": "Listener",

		"lib.title":     "Library",
		"lib.search":    "Search title or artist",
		"lib.youtube":   "From YouTube",
		"lib.colTitle":  "TITLE",
		"lib.noResults": "Nothing found",
		"lib.unknown":   "Unknown artist",

		"empty.title": "It's quiet here",
		"empty.text":  "Choose a music folder — the player will find every mp3,\nincluding subfolders.",
		"empty.pick":  "Choose folder",

		"player.nothing": "Nothing playing",

		"pl.label":         "PLAYLIST",
		"pl.new":           "New playlist",
		"pl.name":          "Name",
		"pl.create":        "Create",
		"pl.rename":        "Rename",
		"pl.delete":        "Delete",
		"pl.deleteConfirm": "Delete playlist “%s”? The tracks stay in your library.",
		"pl.empty":         "This playlist is empty.\nAdd tracks from the library with the ⋯ button.",
		"pl.play":          "Play",
		"pl.addTitle":      "Add to playlist",
		"pl.done":          "Done",

		"common.cancel": "Cancel",
		"common.error":  "Error",
		"common.save":   "Save",

		"settings.title":         "Settings",
		"settings.profile":       "Profile",
		"settings.profileHint":   "Stored on this computer only. Sharing playlists will come later.",
		"settings.avatarChange":  "Change photo",
		"settings.avatarRemove":  "Remove",
		"settings.nickname":      "Nickname",
		"settings.language":      "Language",
		"settings.languageHint":  "Menus, buttons and messages. Applies instantly.",
		"settings.theme":         "Theme",
		"settings.library":       "Library",
		"settings.folderHint":    "YouTube downloads are saved here too",
		"settings.folderNone":    "No folder selected",
		"settings.change":        "Change",
		"settings.covers":        "Show cover art",
		"settings.coversHint":    "From mp3 tags; YouTube tracks get the video thumbnail.",
		"settings.about":         "About",
		"settings.version":       "version %s",
		"settings.source":        "Source code on GitHub",
		"settings.avatarBadFile": "Couldn't read this image. Use PNG or JPEG.",

		"theme.dark":  "Dark",
		"theme.black": "Black",
		"theme.white": "White",
		"theme.neon":  "Neon",

		"yt.title":      "Add from YouTube",
		"yt.link":       "Link",
		"yt.hint":       "Videos up to %d minutes. The file is saved to your library folder.",
		"yt.download":   "Download",
		"yt.needFolder": "Choose a library folder first — the mp3 will be saved there.",
		"yt.loading":    "Downloading from YouTube",
		"yt.probing":    "Getting video info…",
		"yt.progress":   "Downloading and converting… %s",
		"yt.mb":         "%.1f MB",
		"yt.mbOf":       "%.1f of %.1f MB",

		"err.badURL":     "This is not a YouTube video link",
		"err.tooLong":    "The video is longer than %d minutes",
		"err.noDuration": "Couldn't get the video length (is it a live stream?)",
		"err.blocked":    "YouTube blocked the download (403). Try again later or install yt-dlp — the player uses it as a fallback.",
		"nav.import":     "Import playlist",

		"lib.addToPlaylist":    "Add to playlist…",
		"lib.delete":           "Delete from library",
		"lib.deleteTitle":      "Delete track",
		"lib.deleteConfirmBin": "Delete “%s”?\nThe file goes to the Recycle Bin and disappears from all playlists.",
		"lib.deleteConfirm":    "Delete “%s”?\nThe file will be deleted and disappear from all playlists.",

		"pl.addTracks":   "Add tracks",
		"pl.addSelected": "Add (%d)",
		"pl.allAdded":    "All library tracks are already in this playlist.",
		"pl.cover":       "Change cover",
		"pl.coverRemove": "Remove cover",
		"pl.share":       "Share",

		"share.title":  "Share playlist",
		"share.hint":   "Send this code to a friend. In playerzzz they press “+” → “Import playlist” and paste it.",
		"share.stats":  "%d of %d tracks can be downloaded from YouTube by the receiver; the rest have to be in their library already.",
		"share.copy":   "Copy code",
		"share.copied": "Code copied",

		"import.title":      "Import playlist",
		"import.paste":      "Paste the code you received",
		"import.next":       "Next",
		"import.bad":        "This doesn't look like a playerzzz playlist code.",
		"import.found":      "In your library: %d",
		"import.download":   "Can be downloaded from YouTube: %d",
		"import.missing":    "Not found: %d",
		"import.doDownload": "Download missing tracks from YouTube",
		"import.go":         "Import",
		"import.progress":   "Downloading %d of %d…",
		"import.done":       "Playlist imported: %d of %d tracks.",

		"settings.window":          "Window",
		"settings.systemFrame":     "Use the standard Windows title bar",
		"settings.systemFrameHint": "Takes effect after restart.",
		"pl.export":                "Export to file…",
		"nav.importFile":           "Import from file…",
		"export.title":             "Export playlist",
		"export.hint":              "All tracks are packed into one .pzz file — send it via Telegram, a USB stick or the cloud. The receiver opens it in playerzzz: “+” → “Import from file”.",
		"export.small":             "Smaller file: keep YouTube tracks as links (the receiver downloads them)",
		"export.size":              "Size: about %s",
		"export.save":              "Save…",
		"export.progress":          "Packing… %s",
		"export.done":              "Saved:\n%s",
		"share.orFile":             "Not every track is on YouTube? Send the playlist as a file — all tracks inside.",
		"import.packed":            "Inside the file: %d",
		"import.copying":           "Copying %d of %d…",
		"import.badFile":           "This file is not a playerzzz playlist.",
	},
	RU: {
		"nav.library":   "Медиатека",
		"nav.settings":  "Настройки",
		"nav.playlists": "ПЛЕЙЛИСТЫ",
		"nav.newList":   "Новый плейлист",

		"profile.local":   "Локальный профиль",
		"profile.default": "Слушатель",

		"lib.title":     "Медиатека",
		"lib.search":    "Название или исполнитель",
		"lib.youtube":   "С YouTube",
		"lib.colTitle":  "НАЗВАНИЕ",
		"lib.noResults": "Ничего не найдено",
		"lib.unknown":   "Неизвестный исполнитель",

		"empty.title": "Здесь пока тихо",
		"empty.text":  "Выберите папку с музыкой — плеер найдёт все mp3-файлы,\nвключая вложенные папки.",
		"empty.pick":  "Выбрать папку",

		"player.nothing": "Ничего не играет",

		"pl.label":         "ПЛЕЙЛИСТ",
		"pl.new":           "Новый плейлист",
		"pl.name":          "Название",
		"pl.create":        "Создать",
		"pl.rename":        "Переименовать",
		"pl.delete":        "Удалить",
		"pl.deleteConfirm": "Удалить плейлист «%s»? Треки останутся в медиатеке.",
		"pl.empty":         "Плейлист пуст.\nДобавляйте треки из медиатеки кнопкой ⋯.",
		"pl.play":          "Слушать",
		"pl.addTitle":      "Добавить в плейлист",
		"pl.done":          "Готово",

		"common.cancel": "Отмена",
		"common.error":  "Ошибка",
		"common.save":   "Сохранить",

		"settings.title":         "Настройки",
		"settings.profile":       "Профиль",
		"settings.profileHint":   "Хранится только на этом компьютере. Делиться плейлистами можно будет позже.",
		"settings.avatarChange":  "Сменить фото",
		"settings.avatarRemove":  "Убрать",
		"settings.nickname":      "Никнейм",
		"settings.language":      "Язык интерфейса",
		"settings.languageHint":  "Меню, кнопки и сообщения. Меняется сразу.",
		"settings.theme":         "Тема",
		"settings.library":       "Медиатека",
		"settings.folderHint":    "Сюда же сохраняются треки с YouTube",
		"settings.folderNone":    "Папка не выбрана",
		"settings.change":        "Изменить",
		"settings.covers":        "Показывать обложки",
		"settings.coversHint":    "Из тегов mp3, а для треков с YouTube — превью ролика.",
		"settings.about":         "О программе",
		"settings.version":       "версия %s",
		"settings.source":        "Исходный код на GitHub",
		"settings.avatarBadFile": "Не получилось прочитать картинку. Подойдёт PNG или JPEG.",

		"theme.dark":  "Тёмная",
		"theme.black": "Чёрная",
		"theme.white": "Светлая",
		"theme.neon":  "Неон",

		"yt.title":      "Добавить с YouTube",
		"yt.link":       "Ссылка",
		"yt.hint":       "Ролик до %d минут. Файл сохранится в папку медиатеки.",
		"yt.download":   "Скачать",
		"yt.needFolder": "Сначала выберите папку медиатеки — туда сохранится mp3.",
		"yt.loading":    "Загрузка с YouTube",
		"yt.probing":    "Получаю информацию о ролике…",
		"yt.progress":   "Скачиваю и конвертирую… %s",
		"yt.mb":         "%.1f МБ",
		"yt.mbOf":       "%.1f из %.1f МБ",

		"err.badURL":     "Это не ссылка на ролик YouTube",
		"err.tooLong":    "Ролик длиннее %d минут",
		"err.noDuration": "Не удалось узнать длительность (прямой эфир?)",
		"err.blocked":    "YouTube заблокировал загрузку (403). Попробуйте позже или установите yt-dlp — плеер использует его как запасной путь.",
		"nav.import":     "Импорт плейлиста",

		"lib.addToPlaylist":    "Добавить в плейлист…",
		"lib.delete":           "Удалить из медиатеки",
		"lib.deleteTitle":      "Удалить трек",
		"lib.deleteConfirmBin": "Удалить «%s»?\nФайл уйдёт в Корзину и пропадёт из всех плейлистов.",
		"lib.deleteConfirm":    "Удалить «%s»?\nФайл будет удалён и пропадёт из всех плейлистов.",

		"pl.addTracks":   "Добавить треки",
		"pl.addSelected": "Добавить (%d)",
		"pl.allAdded":    "Все треки медиатеки уже в этом плейлисте.",
		"pl.cover":       "Сменить обложку",
		"pl.coverRemove": "Убрать обложку",
		"pl.share":       "Поделиться",

		"share.title":  "Поделиться плейлистом",
		"share.hint":   "Отправьте этот код другу. В playerzzz он нажмёт «+» → «Импорт плейлиста» и вставит его.",
		"share.stats":  "%d из %d треков получатель сможет скачать с YouTube, остальные должны уже быть в его медиатеке.",
		"share.copy":   "Скопировать код",
		"share.copied": "Код скопирован",

		"import.title":      "Импорт плейлиста",
		"import.paste":      "Вставьте код, который вам прислали",
		"import.next":       "Далее",
		"import.bad":        "Это не похоже на код плейлиста playerzzz.",
		"import.found":      "Есть в медиатеке: %d",
		"import.download":   "Можно скачать с YouTube: %d",
		"import.missing":    "Не найдено: %d",
		"import.doDownload": "Докачать недостающие треки с YouTube",
		"import.go":         "Импортировать",
		"import.progress":   "Скачиваю %d из %d…",
		"import.done":       "Плейлист импортирован: %d из %d треков.",

		"settings.window":          "Окно",
		"settings.systemFrame":     "Стандартная рамка Windows",
		"settings.systemFrameHint": "Применится после перезапуска.",
		"pl.export":                "Экспорт в файл…",
		"nav.importFile":           "Импорт из файла…",
		"export.title":             "Экспорт плейлиста",
		"export.hint":              "Все треки упакуются в один файл .pzz — отправьте его в Telegram, на флешке или через облако. Получатель откроет его в playerzzz: «+» → «Импорт из файла».",
		"export.small":             "Файл поменьше: YouTube-треки оставить ссылками (получатель скачает их сам)",
		"export.size":              "Размер: примерно %s",
		"export.save":              "Сохранить…",
		"export.progress":          "Упаковываю… %s",
		"export.done":              "Сохранено:\n%s",
		"share.orFile":             "Не все треки есть на YouTube? Отправьте плейлист файлом — все треки внутри.",
		"import.packed":            "Внутри файла: %d",
		"import.copying":           "Копирую %d из %d…",
		"import.badFile":           "Этот файл — не плейлист playerzzz.",
	},
}
