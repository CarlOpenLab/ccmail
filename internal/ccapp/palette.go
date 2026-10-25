package ccapp

import (
	"github.com/egoist/mygo/ui"
)

// palette holds the app's own colors, one set per appearance. Widgets keep
// drawing themselves from the theme; these are for the surfaces the design
// paints itself, such as the floating card and its hairline.
type palette struct {
	Dark bool
	// Neutral is set when the chrome accent is graphite: badges and chips
	// stay quiet instead of turning into bright pills.
	Neutral bool

	Window   ui.Color
	Card     ui.Color
	Popover  ui.Color
	Sunken   ui.Color
	Hover    ui.Color
	Chip     ui.Color
	ChipDeep ui.Color
	Selected ui.Color

	Text   ui.Color
	Muted  ui.Color
	Faint  ui.Color
	Border ui.Color

	Accent     ui.Color
	AccentText ui.Color
	AccentSoft ui.Color
	Danger     ui.Color
	Success    ui.Color
	Warning    ui.Color
	Star       ui.Color
	StarSoft   ui.Color

	Shadow ui.Color
}

// accentPreset is one choice of chrome accent. The default is graphite:
// color comes from mail and accounts, not from the window itself.
type accentPreset struct {
	ID, Name    string
	Light, Dark string
}

var accentPresets = []accentPreset{
	{ID: "ink", Name: "石墨", Light: "#2b2e34", Dark: "#8b94a4"},
	{ID: "indigo", Name: "靛蓝", Light: "#3b5bdb", Dark: "#8fa4ff"},
	{ID: "moss", Name: "苔绿", Light: "#39664f", Dark: "#8cc7a8"},
	{ID: "clay", Name: "陶土", Light: "#9c4f3b", Dark: "#e8a58c"},
}

func accentFor(id string, dark bool) (ui.Color, ui.Color, string) {
	for _, a := range accentPresets {
		if a.ID != id {
			continue
		}
		if dark {
			return ui.Hex(a.Dark), ui.Hex("#141414"), a.Dark
		}
		return ui.Hex(a.Light), ui.Hex("#ffffff"), a.Light
	}
	return accentFor("ink", dark)
}

// hexOf writes a color as #rrggbb for pages the app composes.
func hexOf(c ui.Color) string {
	const digits = "0123456789abcdef"
	out := []byte("#000000")
	for i, v := range []uint8{c.R, c.G, c.B} {
		out[1+i*2] = digits[v>>4]
		out[2+i*2] = digits[v&0xf]
	}
	return string(out)
}

func hasAccent(id string) bool {
	for _, a := range accentPresets {
		if a.ID == id {
			return true
		}
	}
	return false
}

// badgeColors keeps counts readable: graphite chrome uses a chip, a colored
// accent uses the accent fill.
func (p palette) badgeColors() (ui.Color, ui.Color) {
	if p.Neutral {
		return p.ChipDeep, p.Text
	}
	return p.Accent, p.AccentText
}

func paletteFor(dark bool, accentID string) palette {
	accent, onAccent, _ := accentFor(accentID, dark)
	if dark {
		return palette{
			Dark:     true,
			Window:   ui.Hex("#121216"),
			Card:     ui.Hex("#1d1d22"),
			Popover:  ui.Hex("#232329"),
			Sunken:   ui.Hex("#17171c"),
			Hover:    ui.RGBA(255, 255, 255, 0.045),
			Chip:     ui.RGBA(255, 255, 255, 0.06),
			ChipDeep: ui.RGBA(255, 255, 255, 0.105),
			Selected: accent.Alpha(0.18),

			Text:   ui.Hex("#f0f0f4"),
			Muted:  ui.Hex("#a7a7b3"),
			Faint:  ui.Hex("#92929e"),
			Border: ui.RGBA(255, 255, 255, 0.075),

			Accent:     accent,
			AccentText: onAccent,
			AccentSoft: accent.Alpha(0.13),
			Danger:     ui.Hex("#ff6961"),
			Success:    ui.Hex("#4fd17a"),
			Warning:    ui.Hex("#f0a33a"),
			Star:       ui.Hex("#f0b429"),
			StarSoft:   ui.RGBA(240, 180, 41, 0.15),

			Shadow: ui.RGBA(0, 0, 0, 0.45),
		}
	}
	return palette{
		Window:   ui.Hex("#eef0f4"),
		Card:     ui.Hex("#ffffff"),
		Popover:  ui.Hex("#ffffff"),
		Sunken:   ui.Hex("#f4f5f8"),
		Hover:    ui.RGBA(15, 18, 35, 0.035),
		Chip:     ui.RGBA(15, 18, 35, 0.045),
		ChipDeep: ui.RGBA(15, 18, 35, 0.075),
		Selected: accent.Alpha(0.13),

		Text:   ui.Hex("#18181b"),
		Muted:  ui.Hex("#64646e"),
		Faint:  ui.Hex("#6b6b75"),
		Border: ui.RGBA(15, 18, 35, 0.07),

		Accent:     accent,
		AccentText: onAccent,
		AccentSoft: accent.Alpha(0.09),
		Danger:     ui.Hex("#d93a35"),
		Success:    ui.Hex("#22863a"),
		Warning:    ui.Hex("#b57c10"),
		Star:       ui.Hex("#a06c00"),
		StarSoft:   ui.RGBA(160, 108, 0, 0.11),

		Shadow: ui.RGBA(16, 18, 35, 0.16),
	}
}

// themed sets the widget theme from the app palette and returns both: the
// widgets (buttons, fields, lists) then match the surfaces painted here.
func themed(c *ui.Context, accentID string) (palette, *ui.Theme) {
	base := *c.Theme()
	p := paletteFor(base.Dark, accentID)
	p.Neutral = !hasAccent(accentID) || accentID == "ink"

	t := base
	t.Background = p.Window
	t.Surface = p.Card
	t.SurfaceHover = p.Sunken
	t.SurfacePressed = p.Chip
	t.Border = p.Border
	t.Text = p.Text
	t.TextMuted = p.Faint
	t.Accent = p.Accent
	t.AccentText = p.AccentText
	t.AccentHover = p.Accent.Mix(p.Text, 0.12)
	t.AccentPressed = p.Accent.Mix(p.Text, 0.2)
	t.Danger = p.Danger
	t.Warning = p.Warning
	t.Success = p.Success
	t.Selection = p.Selected
	t.Focus = p.Accent
	t.Scrollbar = p.Faint.Alpha(0.5)
	t.ScrollbarWidth = 10
	t.Radius = 8
	t.Spacing = 4
	if t.FontSize < 12 || t.FontSize > 20 {
		t.FontSize = 13
	}
	c.SetTheme(&t)
	return p, &t
}
