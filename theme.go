package main

import (
	_ "embed"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"image/color"
)

type myTheme struct{}

var _ fyne.Theme = (*myTheme)(nil)

//go:embed fonts/NanumGothic.ttf
var font []byte

func (m myTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, variant)
}

func (m myTheme) Font(_ fyne.TextStyle) fyne.Resource {
	return fyne.NewStaticResource("font.ttf", font)
}

func (m myTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m myTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
