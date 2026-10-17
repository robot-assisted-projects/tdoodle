package palette

import (
	"reflect"
	"testing"

	"github.com/gdamore/tcell/v3/color"

	"tdoodle/internal/canvas"
)

func TestDetectedCapabilities(t *testing.T) {
	tests := []struct {
		name             string
		reported         int
		linux            bool
		fg, bg           int
		fgCount, bgCount int
	}{
		{"monochrome", 0, false, 0, 0, 1, 1},
		{"monochrome linux", 0, true, 0, 0, 1, 1},
		{"eight", 8, false, 8, 8, 8, 8},
		{"sixteen", 16, false, 16, 16, 10, 10},
		{"eighty-eight approximates sixteen", 88, false, 16, 16, 10, 10},
		{"indexed", 256, false, 256, 256, 11, 11},
		{"true color", 1 << 24, false, 256, 256, 11, 11},
		{"linux terminfo eight", 8, true, 16, 8, 10, 8},
		{"linux capped", 256, true, 16, 8, 10, 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Detect(tt.reported, tt.linux)
			if p.MaxColors(false) != tt.fg || p.MaxColors(true) != tt.bg {
				t.Fatalf("effective capabilities = %d/%d, want %d/%d", p.MaxColors(false), p.MaxColors(true), tt.fg, tt.bg)
			}
			for _, background := range []bool{false, true} {
				want := tt.fgCount
				if background {
					want = tt.bgCount
				}
				choices := p.Colors(background)
				if len(choices) != want {
					t.Fatalf("background=%t choices = %v, want %d", background, choices, want)
				}
				seen := map[color.Color]bool{}
				for _, c := range choices {
					mapped := p.Map(c, background)
					if seen[mapped] {
						t.Fatalf("background=%t repeated rendered color %s", background, c)
					}
					seen[mapped] = true
				}
			}
		})
	}
}

func TestCanonicalAliasesAndOrdering(t *testing.T) {
	eight := Detect(8, false)
	want := []canvas.Color{canvas.Red, canvas.Brown, canvas.Green, canvas.Blue, canvas.Violet, canvas.White, canvas.Black, canvas.Cyan}
	if got := eight.Colors(false); !reflect.DeepEqual(got, want) {
		t.Fatalf("eight-color order = %v, want %v", got, want)
	}
	for _, c := range []canvas.Color{canvas.Orange, canvas.Yellow, canvas.Brown} {
		if eight.Resolve(c, false) != canvas.Brown {
			t.Fatalf("%s should anchor at brown", c)
		}
	}
	if eight.Resolve(canvas.Gray, false) != canvas.White {
		t.Fatal("gray should anchor at white on eight-color terminals")
	}
	sixteen := Detect(16, false)
	if sixteen.Resolve(canvas.Orange, false) != canvas.Yellow {
		t.Fatal("orange should anchor at yellow on sixteen-color terminals")
	}
	for _, n := range []int{0, 2, 8, 16, 256} {
		p := Detect(n, false)
		for _, c := range candidates {
			if p.Map(c, false) != p.Map(p.Resolve(c, false), false) {
				t.Errorf("%d colors: %s and its canonical choice render differently", n, c)
			}
		}
	}
}

func TestLinuxChannelsUseTheirActualCapabilities(t *testing.T) {
	p := Detect(8, true)
	if p.Map(canvas.Red, false) != color.Red || p.Map(canvas.Red, true) != color.Maroon {
		t.Fatal("Linux should use bright red foreground and normal red background")
	}
	if p.Map(canvas.Cyan, false) != color.Aqua || p.Map(canvas.Cyan, true) != color.Teal {
		t.Fatal("Linux should provide cyan in both channels")
	}
	for _, c := range candidates {
		mapped := p.Map(c, true)
		if mapped < color.XTerm0 || mapped > color.XTerm7 {
			t.Fatalf("Linux background %s exceeds eight shades: %v", c, mapped)
		}
	}
}

func TestOverrides(t *testing.T) {
	for _, tt := range []struct {
		mode   string
		linux  bool
		fg, bg int
	}{
		{"auto", true, 16, 8},
		{"mono", true, 0, 0},
		{"8", true, 8, 8},
		{"16", true, 16, 8},
		{"256", true, 16, 8},
		{"256", false, 256, 256},
	} {
		p, err := ForMode(tt.mode, 8, tt.linux)
		if err != nil {
			t.Fatal(err)
		}
		if p.MaxColors(false) != tt.fg || p.MaxColors(true) != tt.bg {
			t.Errorf("ForMode(%q, linux=%t) = %d/%d", tt.mode, tt.linux, p.MaxColors(false), p.MaxColors(true))
		}
	}
	if _, err := ForMode("16777216", 8, false); err == nil {
		t.Fatal("invalid override should be rejected")
	}
}

func TestApproximationPreservesDocumentAndPaletteOwnership(t *testing.T) {
	doc, err := canvas.New(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	original := canvas.Cell{Rune: 'A', FG: canvas.Orange, BG: canvas.Gray}
	doc.Set(canvas.Point{}, original)
	p := Detect(8, false)
	cell := doc.At(canvas.Point{})
	if p.Resolve(cell.FG, false) != canvas.Brown || p.Resolve(cell.BG, true) != canvas.White {
		t.Fatal("test cell should require foreground and background approximation")
	}
	_ = p.Map(cell.FG, false)
	_ = p.Map(cell.BG, true)
	if got := doc.At(canvas.Point{}); got != original {
		t.Fatalf("display approximation changed saved logical colors: %+v", got)
	}
	choices := p.Colors(false)
	choices[0] = canvas.Black
	if p.Colors(false)[0] != canvas.Red {
		t.Fatal("caller modified future palette choices")
	}
	if Detect(8, true) != Detect(16, true) {
		t.Fatal("equivalent detected profiles should compare equal")
	}
}
