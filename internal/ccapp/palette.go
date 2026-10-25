package ccapp

import (
	"github.com/egoist/mygo/ui"
)

// palette holds the app's own colors, one set per appearance. Widgets keep
// drawing themselves from the theme; these are for the surfaces the design
// paints itself, such as the floating card and its hairline.
type palette struct {
	Dark bool

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

func paletteFor(dark bool) palette {
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
			Selected: ui.Hex("#7b9dff").Alpha(0.18),

			Text:   ui.Hex("#f0f0f4"),
			Muted:  ui.Hex("#a7a7b3"),
			Faint:  ui.Hex("#92929e"),
			Border: ui.RGBA(255, 255, 255, 0.075),

			Accent:     ui.Hex("#7b9dff"),
			AccentText: ui.Hex("#111114"),
			AccentSoft: ui.Hex("#7b9dff").Alpha(0.13),
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
		Selected: ui.Hex("#3b6cf6").Alpha(0.13),

		Text:   ui.Hex("#18181b"),
		Muted:  ui.Hex("#64646e"),
		Faint:  ui.Hex("#6b6b75"),
		Border: ui.RGBA(15, 18, 35, 0.07),

		Accent:     ui.Hex("#3b6cf6"),
		AccentText: ui.Hex("#ffffff"),
		AccentSoft: ui.Hex("#3b6cf6").Alpha(0.09),
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
func themed(c *ui.Context) (palette, *ui.Theme) {
	base := *c.Theme()
	p := paletteFor(base.Dark)

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
