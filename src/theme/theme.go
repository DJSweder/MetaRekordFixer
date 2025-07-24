package theme

import (
	"MetaRekordFixer/assets"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Sdílená mapa barev pro oba motivy
var sharedColorMap = map[fyne.ThemeColorName]color.Color{
	"ErrorMessagesColor":               color.RGBA{R: 255, G: 0, B: 0, A: 255},          // Red
	theme.ColorNameBackground:          color.RGBA{R: 30, G: 30, B: 30, A: 255},         // #1E1E1E
	theme.ColorNameButton:              color.RGBA{R: 30, G: 30, B: 30, A: 255},         // #1E1E1E
	theme.ColorNameDisabledButton:      color.NRGBA{R: 0xe5, G: 0xe5, B: 0xe5, A: 0xff}, // #96969
	theme.ColorNameDisabled:            color.RGBA{R: 150, G: 150, B: 150, A: 255},      // #969696
	theme.ColorNameError:               color.NRGBA{R: 0xc2, G: 0x14, B: 0x3d, A: 0xff}, // #C2143D
	theme.ColorNameFocus:               color.RGBA{R: 194, G: 20, B: 61, A: 255},        // #C2143D
	theme.ColorNameForeground:          color.RGBA{R: 255, G: 255, B: 255, A: 255},      // #FFFFFF
	theme.ColorNameForegroundOnError:   color.RGBA{R: 255, G: 255, B: 255, A: 255},      // #FFFFFF
	theme.ColorNameForegroundOnPrimary: color.RGBA{R: 255, G: 255, B: 255, A: 255},      // #FFFFFF
	theme.ColorNameHeaderBackground:    color.RGBA{R: 58, G: 58, B: 58, A: 255},         // #3A3A3A
	theme.ColorNameHover:               color.RGBA{R: 71, G: 71, B: 71, A: 255},         // #474747
	theme.ColorNameInputBackground:     color.RGBA{R: 0, G: 0, B: 0, A: 255},            // #000000
	theme.ColorNameMenuBackground:      color.RGBA{R: 41, G: 41, B: 46, A: 255},         // ##28292E
	theme.ColorNamePlaceHolder:         color.RGBA{R: 179, G: 179, B: 179, A: 255},      // #B3B3B3
	theme.ColorNamePressed:             color.RGBA{R: 33, G: 33, B: 33, A: 255},         // #212121
	theme.ColorNamePrimary:             color.RGBA{R: 194, G: 20, B: 61, A: 255},        // #C2143D
	theme.ColorNameScrollBar:           color.RGBA{R: 66, G: 66, B: 66, A: 255},         // #424242
	theme.ColorNameShadow:              color.RGBA{A: 66},                               // #000000
	theme.ColorNameSelection:           color.RGBA{R: 194, G: 20, B: 61, A: 255},        // #C2143D (Same as main application color)
	theme.ColorNameSeparator:           color.RGBA{R: 0, G: 0, B: 0, A: 255},            // #000000
	theme.ColorNameInputBorder:         color.RGBA{R: 33, G: 33, B: 33, A: 255},         // #212121
	theme.ColorNameOverlayBackground:   color.RGBA{R: 33, G: 33, B: 33, A: 255},         // #000000
	theme.ColorNameSuccess:             color.RGBA{R: 67, G: 244, B: 54, A: 255},        // #43F436
	theme.ColorNameWarning:             color.RGBA{R: 255, G: 152, B: 0, A: 255},        // #FF9800
}

// Mapa velikostí pro customTheme
var customSizeMap = map[fyne.ThemeSizeName]float32{
	theme.SizeNameSeparatorThickness: 1,  // Separator thickness
	theme.SizeNameInlineIcon:         20, // Size of inline icons
	theme.SizeNameInnerPadding:       8,  // Inner padding of elements
	theme.SizeNameLineSpacing:        6,  // Line spacing
	theme.SizeNamePadding:            2,  // Padding around elements
	theme.SizeNameScrollBar:          12, // Scrollbar size
	theme.SizeNameScrollBarSmall:     12, // Small scrollbar size
	theme.SizeNameText:               15, // Main text size
	theme.SizeNameHeadingText:        24, // Size of headings
	theme.SizeNameSubHeadingText:     18, // Size of subheadings
	theme.SizeNameCaptionText:        11, // Size of captions
	theme.SizeNameInputBorder:        1,  // Width of input borders
	theme.SizeNameInputRadius:        8,  // Radius of input corners
	theme.SizeNameSelectionRadius:    8,  // Radius of selection corners
	theme.SizeNameScrollBarRadius:    8,  // Radius of scrollbar corners
}

type customTheme struct {
	fyne.Theme
}

func (t *customTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if customColor, exists := sharedColorMap[name]; exists {
		return customColor
	}
	return t.Theme.Color(name, variant)
}

func (t *customTheme) Size(name fyne.ThemeSizeName) float32 {
	if size, exists := customSizeMap[name]; exists {
		return size
	}
	return t.Theme.Size(name)
}

func (t *customTheme) Font(style fyne.TextStyle) fyne.Resource {
	if style.Bold {
		return assets.ResourceRobotoCondensedBold // Bold text
	}
	if style.Italic {
		return assets.ResourceRobotoCondensedItalic // Italic text
	}
	return assets.ResourceRobotoCondensedRegular // Regular text
}

func AppIcon() fyne.Resource {
	return assets.ResourceAppLogo
}

// New custom theme with dark look regardless of system settings
func NewCustomTheme() fyne.Theme {
	return &customTheme{Theme: &darkTheme{}}
}

// New structure for dark look
type darkTheme struct{}

func (t *darkTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	if customColor, exists := sharedColorMap[name]; exists {
		return customColor
	}
	return theme.DefaultTheme().Color(name, theme.VariantDark)
}

func (t *darkTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *darkTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *darkTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

// InfoIcon returns the info icon from theme
func InfoIcon() fyne.Resource {
	// Returns the info icon from the theme
	return theme.InfoIcon()
}

// WarningIcon returns the warning icon from theme
func WarningIcon() fyne.Resource {
	// Returns the warning icon from the theme
	return theme.WarningIcon()
}

// ErrorIcon returns the error icon from theme
func ErrorIcon() fyne.Resource {
	// Returns the error icon from the theme
	return theme.ErrorIcon()
}
