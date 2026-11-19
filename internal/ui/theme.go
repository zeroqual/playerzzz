package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type Palette struct {
	ID       string
	Light    bool
	Bg       color.NRGBA
	Sidebar  color.NRGBA
	Panel    color.NRGBA
	Raised   color.NRGBA
	Border   color.NRGBA
	Sep      color.NRGBA
	Text     color.NRGBA
	Muted    color.NRGBA
	Accent   color.NRGBA
	OnAccent color.NRGBA
}

func rgb(hex uint32) color.NRGBA {
	return color.NRGBA{R: uint8(hex >> 16), G: uint8(hex >> 8), B: uint8(hex), A: 0xFF}
}

func withAlpha(c color.NRGBA, a uint8) color.NRGBA { c.A = a; return c }

var Palettes = []Palette{
	{
		ID: "dark", Bg: rgb(0x0E1013), Sidebar: rgb(0x121418), Panel: rgb(0x15181D), Raised: rgb(0x1C2027),
		Border: rgb(0x262B33), Sep: rgb(0x1F232A), Text: rgb(0xECEEF1), Muted: rgb(0x8B93A1),
		Accent: rgb(0x6EE7A8), OnAccent: rgb(0x0B1A12),
	},
	{
		ID: "black", Bg: rgb(0x000000), Sidebar: rgb(0x050505), Panel: rgb(0x0B0B0B), Raised: rgb(0x171717),
		Border: rgb(0x242424), Sep: rgb(0x1A1A1A), Text: rgb(0xF5F5F5), Muted: rgb(0x8E8E8E),
		Accent: rgb(0xFFFFFF), OnAccent: rgb(0x000000),
	},
	{
		ID: "white", Light: true, Bg: rgb(0xF7F7F5), Sidebar: rgb(0xEFEFEC), Panel: rgb(0xFFFFFF), Raised: rgb(0xE9E9E5),
		Border: rgb(0xDADAD4), Sep: rgb(0xE4E4DF), Text: rgb(0x15171A), Muted: rgb(0x5F6670),
		Accent: rgb(0x1F8A5B), OnAccent: rgb(0xFFFFFF),
	},
	{
		ID: "neon", Bg: rgb(0x0B0617), Sidebar: rgb(0x100A20), Panel: rgb(0x150D2A), Raised: rgb(0x1F143A),
		Border: rgb(0x31215A), Sep: rgb(0x241846), Text: rgb(0xF3EEFF), Muted: rgb(0xA497CC),
		Accent: rgb(0xFF3DCB), OnAccent: rgb(0x1A0014),
	},
}

func paletteByID(id string) Palette {
	for _, p := range Palettes {
		if p.ID == id {
			return p
		}
	}
	return Palettes[0]
}

type playerTheme struct{ p Palette }

var _ fyne.Theme = playerTheme{}

func (t playerTheme) variant() fyne.ThemeVariant {
	if t.p.Light {
		return theme.VariantLight
	}
	return theme.VariantDark
}

func (t playerTheme) Color(n fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	p := t.p
	hover, pressed := uint8(0x0F), uint8(0x1A)
	tint := color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF}
	if p.Light {
		tint = color.NRGBA{}
	}
	switch n {
	case theme.ColorNameBackground:
		return p.Bg
	case theme.ColorNameButton, theme.ColorNameDisabledButton:
		return p.Raised
	case theme.ColorNameDisabled:

		return p.Muted
	case theme.ColorNameForeground:
		return p.Text
	case theme.ColorNameForegroundOnPrimary:
		return p.OnAccent
	case theme.ColorNamePrimary, theme.ColorNameHyperlink:
		return p.Accent
	case theme.ColorNameFocus:
		return withAlpha(p.Accent, 0x80)
	case theme.ColorNameSelection:
		return withAlpha(p.Accent, 0x22)
	case theme.ColorNameHover:
		return withAlpha(tint, hover)
	case theme.ColorNamePressed:
		return withAlpha(tint, pressed)
	case theme.ColorNameInputBackground, theme.ColorNameHeaderBackground,
		theme.ColorNameOverlayBackground, theme.ColorNameMenuBackground:
		return p.Panel
	case theme.ColorNameInputBorder:
		return p.Border
	case theme.ColorNamePlaceHolder:
		return p.Muted
	case theme.ColorNameSeparator:
		return p.Sep
	case theme.ColorNameScrollBar:
		return withAlpha(tint, 0x40)
	case theme.ColorNameShadow:
		return color.NRGBA{A: 0x66}
	}
	return theme.DefaultTheme().Color(n, t.variant())
}

func (playerTheme) Font(s fyne.TextStyle) fyne.Resource { return theme.DefaultTheme().Font(s) }

func (playerTheme) Icon(n fyne.ThemeIconName) fyne.Resource { return theme.DefaultTheme().Icon(n) }

func (playerTheme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNameInputRadius, theme.SizeNameButtonRadius, theme.SizeNameSelectionRadius:
		return 10
	case theme.SizeNameHeadingText:
		return 28
	case theme.SizeNameSubHeadingText:
		return 17
	case theme.SizeNameScrollBarRadius:
		return 4
	}
	return theme.DefaultTheme().Size(n)
}

type sliderTheme struct {
	playerTheme
	active color.NRGBA
}

func (t sliderTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch n {
	case theme.ColorNameInputBackground:
		return t.p.Border
	case theme.ColorNameForeground:
		return t.active
	}
	return t.playerTheme.Color(n, v)
}
