package theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type MonochromeTheme struct{}

func (m MonochromeTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.White
	case theme.ColorNameButton:
		return color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 220, G: 220, B: 220, A: 255}
	case theme.ColorNameForeground:
		return color.Black
	case theme.ColorNameHover:
		return color.NRGBA{R: 200, G: 200, B: 200, A: 255}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 250, G: 250, B: 250, A: 255}
	case theme.ColorNamePrimary:
		return color.Black
	case theme.ColorNameFocus:
		return color.NRGBA{R: 100, G: 100, B: 100, A: 255}
	case theme.ColorNameShadow:
		return color.NRGBA{R: 0, G: 0, B: 0, A: 50}
	default:
		return color.Gray{Y: 128}
	}
}

func (m MonochromeTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m MonochromeTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m MonochromeTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
