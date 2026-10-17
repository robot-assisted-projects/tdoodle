// Package palette chooses a small drawing palette for a terminal's capabilities.
// It only approximates logical drawing colors; saved documents keep their colors.
package palette

import (
	"fmt"

	"github.com/gdamore/tcell/v3/color"

	"tdoodle/internal/canvas"
)

// Profile is an immutable, comparable description of foreground and background
// capabilities. Its zero value describes a monochrome terminal.
type Profile struct {
	foreground uint16
	background uint16
}

// Detect interprets the color count reported by tcell. A Linux virtual console
// supports sixteen foreground colors and eight backgrounds even when terminfo
// reports only eight colors. A report of zero still means monochrome.
func Detect(reportedColors int, linux bool) Profile {
	if reportedColors <= 0 {
		return Profile{}
	}
	if linux {
		return Profile{foreground: 16, background: 8}
	}
	n := normalize(reportedColors)
	return Profile{foreground: n, background: n}
}

// ForMode applies an optional capability override. Linux console overrides stay
// within its physical sixteen-foreground/eight-background limit.
func ForMode(mode string, reportedColors int, linux bool) (Profile, error) {
	var n uint16
	switch mode {
	case "auto":
		return Detect(reportedColors, linux), nil
	case "mono":
		return Profile{}, nil
	case "8":
		n = 8
	case "16":
		n = 16
	case "256":
		n = 256
	default:
		return Profile{}, fmt.Errorf("unknown color mode %q (use auto, mono, 8, 16, or 256)", mode)
	}
	p := Profile{foreground: n, background: n}
	if linux {
		if p.foreground > 16 {
			p.foreground = 16
		}
		if p.background > 8 {
			p.background = 8
		}
	}
	return p, nil
}

func normalize(n int) uint16 {
	switch {
	case n >= 256:
		return 256
	case n >= 16:
		return 16
	case n >= 8:
		return 8
	case n >= 2:
		return 2
	default:
		return 0
	}
}

// MaxColors returns the effective color capability for a drawing channel.
// Rich-color terminals use the same curated RGB palette, whether they report
// 256 indexed colors or true color; this method reports 256 for either.
func (p Profile) MaxColors(background bool) int {
	if background {
		return int(p.background)
	}
	return int(p.foreground)
}

var candidates = [...]canvas.Color{
	canvas.Red, canvas.Orange, canvas.Yellow, canvas.Green, canvas.Blue,
	canvas.Violet, canvas.Gray, canvas.Brown, canvas.White, canvas.Black,
	canvas.Cyan,
}

var rgbColors = map[canvas.Color]color.Color{
	canvas.Red: color.NewRGBColor(230, 80, 80), canvas.Orange: color.NewRGBColor(245, 158, 66),
	canvas.Yellow: color.NewRGBColor(242, 207, 74), canvas.Green: color.NewRGBColor(102, 191, 115),
	canvas.Blue: color.NewRGBColor(89, 142, 222), canvas.Violet: color.NewRGBColor(183, 123, 224),
	canvas.Gray: color.NewRGBColor(144, 144, 144), canvas.Brown: color.NewRGBColor(149, 101, 69),
	canvas.White: color.NewRGBColor(244, 244, 244), canvas.Black: color.NewRGBColor(16, 16, 16),
	canvas.Cyan: color.NewRGBColor(64, 191, 191),
}

var ansi16Colors = map[canvas.Color]color.Color{
	canvas.Red: color.Red, canvas.Orange: color.Yellow,
	canvas.Yellow: color.Yellow, canvas.Green: color.Lime,
	canvas.Blue: color.Blue, canvas.Violet: color.Fuchsia,
	canvas.Gray: color.Gray, canvas.Brown: color.Olive,
	canvas.White: color.White, canvas.Black: color.Black,
	canvas.Cyan: color.Aqua,
}

var ansi8Colors = map[canvas.Color]color.Color{
	canvas.Red: color.Maroon, canvas.Orange: color.Olive,
	canvas.Yellow: color.Olive, canvas.Green: color.Green,
	canvas.Blue: color.Navy, canvas.Violet: color.Purple,
	canvas.Gray: color.Silver, canvas.Brown: color.Olive,
	canvas.White: color.Silver, canvas.Black: color.Black,
	canvas.Cyan: color.Teal,
}

// Map returns the rendered approximation of a logical drawing color.
func (p Profile) Map(c canvas.Color, background bool) color.Color {
	n := p.MaxColors(background)
	if n == 0 {
		return color.Default
	}
	if n < 8 {
		if c == canvas.Black {
			return color.Black
		}
		return color.Silver
	}
	colors, fallback := ansi8Colors, color.Silver
	if n >= 256 {
		colors, fallback = rgbColors, rgbColors[canvas.White]
	} else if n >= 16 {
		colors, fallback = ansi16Colors, color.White
	}
	if mapped, ok := colors[c]; ok {
		return mapped
	}
	return fallback
}

// Resolve returns the canonical selectable color with the same approximation.
// In particular, existing saved aliases anchor the knob at their visible shade
// without changing the original logical color in the drawing.
func (p Profile) Resolve(c canvas.Color, background bool) canvas.Color {
	if !c.Valid() {
		c = canvas.White
	}
	switch n := p.MaxColors(background); {
	case n == 0:
		return canvas.White
	case n < 8:
		if c == canvas.Black {
			return canvas.Black
		}
		return canvas.White
	case n < 16:
		switch c {
		case canvas.Orange, canvas.Yellow:
			return canvas.Brown
		case canvas.Gray:
			return canvas.White
		}
	case n < 256:
		if c == canvas.Orange {
			return canvas.Yellow
		}
	}
	return c
}

// Colors returns a small, stable list with one choice per rendered shade.
// Each call owns its returned slice. Cyan is offered in both drawing channels.
func (p Profile) Colors(background bool) []canvas.Color {
	result := make([]canvas.Color, 0, len(candidates))
	seen := make(map[color.Color]bool, len(candidates))
	for _, candidate := range candidates {
		canonical := p.Resolve(candidate, background)
		mapped := p.Map(canonical, background)
		if !seen[mapped] {
			seen[mapped] = true
			result = append(result, canonical)
		}
	}
	return result
}
