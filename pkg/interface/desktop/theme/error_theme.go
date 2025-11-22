// pkg/interface/desktop/theme/error_theme.go
package theme

import (
	"image/color"
)

var (
	ErrorColorInfo     = color.NRGBA{R: 33, G: 150, B: 243, A: 255} // Blue
	ErrorColorWarning  = color.NRGBA{R: 255, G: 193, B: 7, A: 255}  // Amber
	ErrorColorCritical = color.NRGBA{R: 244, G: 67, B: 54, A: 255}  // Red
	ErrorColorSuccess  = color.NRGBA{R: 76, G: 175, B: 80, A: 255}  // Green
)
