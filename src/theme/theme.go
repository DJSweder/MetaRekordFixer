package theme

import (
	"MetaRekordFixer/assets"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type customTheme struct {
	fyne.Theme
}

func (t *customTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case "ErrorMessagesColor": // Specific color for error messages
		return color.RGBA{R: 255, G: 0, B: 0, A: 255} // Red
	case theme.ColorNameBackground: // Application background color
		return color.RGBA{R: 30, G: 30, B: 30, A: 255} // #1E1E1E
	case theme.ColorNameButton: // Buttons color
		return color.RGBA{R: 30, G: 30, B: 30, A: 255} // #1E1E1E
	case theme.ColorNameDisabledButton: // Disabled button color
		return color.NRGBA{R: 0xe5, G: 0xe5, B: 0xe5, A: 0xff} // #96969
	case theme.ColorNameDisabled: // Disabled elements color
		return color.RGBA{R: 150, G: 150, B: 150, A: 255} // #969696
	case theme.ColorNameError: // Error color
		return color.NRGBA{R: 0xc2, G: 0x14, B: 0x3d, A: 0xff} // #C2143D
	case theme.ColorNameFocus: // Focus color
		return color.RGBA{R: 194, G: 20, B: 61, A: 255} // #C2143D
	case theme.ColorNameForeground: // Text color
		return color.RGBA{R: 255, G: 255, B: 255, A: 255} // #FFFFFF
	case theme.ColorNameForegroundOnError: // Text color on error color
		return color.RGBA{R: 255, G: 255, B: 255, A: 255} // #FFFFFF
	case theme.ColorNameForegroundOnPrimary: // Text color on primary color
		return color.RGBA{R: 255, G: 255, B: 255, A: 255} // #FFFFFF
	case theme.ColorNameHeaderBackground: // Header background color
		return color.RGBA{R: 58, G: 58, B: 58, A: 255} // #3A3A3A
	case theme.ColorNameHover: // Color when hovering over an element
		return color.RGBA{R: 71, G: 71, B: 71, A: 255} // #474747
	case theme.ColorNameInputBackground: // Input background color
		return color.RGBA{R: 0, G: 0, B: 0, A: 255} // #000000
	case theme.ColorNameMenuBackground: // Menu background color
		return color.RGBA{R: 41, G: 41, B: 46, A: 255} // ##28292E
	case theme.ColorNamePlaceHolder: // Color of placeholder text in input fields
		return color.RGBA{R: 179, G: 179, B: 179, A: 255} // #B3B3B3
	case theme.ColorNamePressed: // Color when an element is pressed
		return color.RGBA{R: 33, G: 33, B: 33, A: 255} // #212121
	case theme.ColorNamePrimary: // Main application color (buttons, highlights)
		return color.RGBA{R: 194, G: 20, B: 61, A: 255} // #C2143D
	case theme.ColorNameScrollBar: // Color of scrollbars
		return color.RGBA{R: 66, G: 66, B: 66, A: 255} // #424242
	case theme.ColorNameShadow: // Color of shadows
		return color.RGBA{A: 66} // #000000
	case theme.ColorNameSelection: // Selection color (dropdown menu)
		return color.RGBA{R: 194, G: 20, B: 61, A: 255} // #C2143D (Same as main application color)
	case theme.ColorNameSeparator: // Separator color
		return color.RGBA{R: 0, G: 0, B: 0, A: 255} // #000000
	case theme.ColorNameInputBorder: // Input borders color
		return color.RGBA{R: 33, G: 33, B: 33, A: 255} // #212121
	case theme.ColorNameOverlayBackground: // Dropdown menu background color
		return color.RGBA{R: 33, G: 33, B: 33, A: 255} // #000000
	// return color.RGBA{R: 194, G: 20, B: 61, A: 255} // #C2143D
	case theme.ColorNameSuccess: // Success color
		return color.RGBA{R: 67, G: 244, B: 54, A: 255} // #43F436
	case theme.ColorNameWarning: // Warning color
		return color.RGBA{R: 255, G: 152, B: 0, A: 255} // #FF9800
	default:
		return t.Theme.Color(name, variant)
	}
}

func (t *customTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameSeparatorThickness: // Separator thickness
		return 1
	case theme.SizeNameInlineIcon: // Size of inline icons
		return 20
	case theme.SizeNameInnerPadding: // Inner padding of elements
		return 8
	case theme.SizeNameLineSpacing: // Line spacing
		return 6
	case theme.SizeNamePadding: // Padding around elements
		return 2
	case theme.SizeNameScrollBar: // Scrollbar size
		return 12
	case theme.SizeNameScrollBarSmall: // Small scrollbar size
		return 12
	case theme.SizeNameText: // Main text size
		return 15
	case theme.SizeNameHeadingText: // Size of headings
		return 24
	case theme.SizeNameSubHeadingText: // Size of subheadings
		return 18
	case theme.SizeNameCaptionText: // Size of captions
		return 11
	case theme.SizeNameInputBorder: // Width of input borders
		return 1
	case theme.SizeNameInputRadius: // Radius of input corners
		return 8
	case theme.SizeNameSelectionRadius: // Radius of selection corners
		return 8
	case theme.SizeNameScrollBarRadius: // Radius of scrollbar corners
		return 8
	default:
		return t.Theme.Size(name)
	}
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
	switch name {
	case "ErrorMessagesColor": // Specific color for error messages
		return color.RGBA{R: 255, G: 0, B: 0, A: 255} // Red
	case theme.ColorNameBackground: // Application background color
		return color.RGBA{R: 30, G: 30, B: 30, A: 255} // #1E1E1E
	case theme.ColorNameButton: // Buttons color
		return color.RGBA{R: 30, G: 30, B: 30, A: 255} // #1E1E1E
	case theme.ColorNameDisabledButton: // Disabled button color
		return color.NRGBA{R: 0xe5, G: 0xe5, B: 0xe5, A: 0xff} // #96969
	case theme.ColorNameDisabled: // Disabled elements color
		return color.RGBA{R: 150, G: 150, B: 150, A: 255} // #969696
	case theme.ColorNameError: // Error color
		return color.NRGBA{R: 0xc2, G: 0x14, B: 0x3d, A: 0xff} // #C2143D
	case theme.ColorNameFocus: // Focus color
		return color.RGBA{R: 194, G: 20, B: 61, A: 255} // #C2143D
	case theme.ColorNameForeground: // Text color
		return color.RGBA{R: 255, G: 255, B: 255, A: 255} // #FFFFFF
	case theme.ColorNameForegroundOnError: // Text color on error color
		return color.RGBA{R: 255, G: 255, B: 255, A: 255} // #FFFFFF
	case theme.ColorNameForegroundOnPrimary: // Text color on primary color
		return color.RGBA{R: 255, G: 255, B: 255, A: 255} // #FFFFFF
	case theme.ColorNameHeaderBackground: // Header background color
		return color.RGBA{R: 58, G: 58, B: 58, A: 255} // #3A3A3A
	case theme.ColorNameHover: // Color when hovering over an element
		return color.RGBA{R: 71, G: 71, B: 71, A: 255} // #474747
	case theme.ColorNameInputBackground: // Input background color
		return color.RGBA{R: 0, G: 0, B: 0, A: 255} // #000000
	case theme.ColorNameMenuBackground: // Menu background color
		return color.RGBA{R: 41, G: 41, B: 46, A: 255} // ##28292E
	case theme.ColorNamePlaceHolder: // Color of placeholder text in input fields
		return color.RGBA{R: 179, G: 179, B: 179, A: 255} // #B3B3B3
	case theme.ColorNamePressed: // Color when an element is pressed
		return color.RGBA{R: 33, G: 33, B: 33, A: 255} // #212121
	case theme.ColorNamePrimary: // Main application color (buttons, highlights)
		return color.RGBA{R: 194, G: 20, B: 61, A: 255} // #C2143D
	case theme.ColorNameScrollBar: // Color of scrollbars
		return color.RGBA{R: 66, G: 66, B: 66, A: 255} // #424242
	case theme.ColorNameShadow: // Color of shadows
		return color.RGBA{A: 66} // #000000
	case theme.ColorNameSelection: // Selection color (dropdown menu)
		return color.RGBA{R: 194, G: 20, B: 61, A: 255} // #C2143D (Same as main application color)
	case theme.ColorNameSeparator: // Separator color
		return color.RGBA{R: 0, G: 0, B: 0, A: 255} // #000000
	case theme.ColorNameInputBorder: // Input borders color
		return color.RGBA{R: 33, G: 33, B: 33, A: 255} // #212121
	case theme.ColorNameOverlayBackground: // Dropdown menu background color
		return color.RGBA{R: 33, G: 33, B: 33, A: 255} // #000000
	// return color.RGBA{R: 194, G: 20, B: 61, A: 255} // #C2143D
	case theme.ColorNameSuccess: // Success color
		return color.RGBA{R: 67, G: 244, B: 54, A: 255} // #43F436
	case theme.ColorNameWarning: // Warning color
		return color.RGBA{R: 255, G: 152, B: 0, A: 255} // #FF9800
	default:
		return theme.DefaultTheme().Color(name, theme.VariantDark)
	}
}

func (t *darkTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *darkTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
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

func (t *darkTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
