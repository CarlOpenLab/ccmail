package ccapp

import (
	"github.com/egoist/mygo/ui"
)

// palette holds the app's own colors, one set per appearance. Widgets keep
// drawing themselves from the theme; these are for the surfaces the design
// paints itself, such as the floating card and its hairline.
type palette struct {
	Dark bool

	Window      ui.Color
	Card        ui.Color
	Popover     ui.Color
	Hover       ui.Color
	HoverStrong ui.Color
	Btn         ui.Color
	BtnHover    ui.Color
	Chip        ui.Color
	ChipDeep    ui.Color

	Text   ui.Color
	Muted  ui.Color
	Faint  ui.Color
	Border ui.Color

	// Accent drives carets, switches and the focused source-list item.
	Accent     ui.Color
	AccentText ui.Color
	AccentSoft ui.Color
	// CTA fills the primary action: near-black in light, near-white in dark.
	CTA, CTAText, CTAHover ui.Color
	// SelectedRow paints the chosen message, quieter than the CTA.
	SelectedRow, SelectedText, SelectedMuted ui.Color
	// FocusRing outlines keyboard focus.
	FocusRing ui.Color

	Danger  ui.Color
	Success ui.Color
	Warning ui.Color
	Star    ui.Color

	Shadow ui.Color
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

// badgeColors keeps counts readable without turning the sidebar into a
// column of colored pills.
func (p palette) badgeColors() (ui.Color, ui.Color) {
	return p.ChipDeep, p.Text
}

func paletteFor(dark bool) palette {
	if dark {
		return palette{
			Dark:        true,
			Window:      ui.Hex("#0f1013"),
			Card:        ui.Hex("#17181c"),
			Popover:     ui.Hex("#1e1f24"),
			Hover:       ui.RGBA(255, 255, 255, 0.055),
			HoverStrong: ui.RGBA(255, 255, 255, 0.085),
			Btn:         ui.Hex("#23242a"),
			BtnHover:    ui.Hex("#2c2d34"),
			Chip:        ui.RGBA(255, 255, 255, 0.06),
			ChipDeep:    ui.RGBA(255, 255, 255, 0.10),

			Text:   ui.Hex("#ececf0"),
			Muted:  ui.Hex("#a6a6b1"),
			Faint:  ui.Hex("#91919d"),
			Border: ui.RGBA(255, 255, 255, 0.085),

			Accent:     ui.Hex("#8b94a4"),
			AccentText: ui.Hex("#141414"),
			AccentSoft: ui.RGBA(255, 255, 255, 0.07),
			CTA:        ui.Hex("#e8e9ed"),
			CTAText:    ui.Hex("#14151a"),
			CTAHover:   ui.Hex("#f4f5f7"),

			SelectedRow:   ui.Hex("#2f3441"),
			SelectedText:  ui.Hex("#f3f4f7"),
			SelectedMuted: ui.RGBA(243, 244, 247, 0.62),
			FocusRing:     ui.Hex("#7f8795"),

			Danger:  ui.Hex("#ff6961"),
			Success: ui.Hex("#4fd17a"),
			Warning: ui.Hex("#f0a33a"),
			Star:    ui.Hex("#f0b429"),

			Shadow: ui.RGBA(0, 0, 0, 0.5),
		}
	}
	return palette{
		Window:      ui.Hex("#eef0f4"),
		Card:        ui.Hex("#ffffff"),
		Popover:     ui.Hex("#ffffff"),
		Hover:       ui.RGBA(15, 18, 35, 0.055),
		HoverStrong: ui.RGBA(15, 18, 35, 0.085),
		Btn:         ui.Hex("#f2f3f5"),
		BtnHover:    ui.Hex("#e8eaee"),
		Chip:        ui.RGBA(15, 18, 35, 0.045),
		ChipDeep:    ui.RGBA(15, 18, 35, 0.075),

		Text:   ui.Hex("#18181b"),
		Muted:  ui.Hex("#64646e"),
		Faint:  ui.Hex("#6b6b75"),
		Border: ui.RGBA(15, 18, 35, 0.07),

		Accent:     ui.Hex("#2b2e34"),
		AccentText: ui.Hex("#ffffff"),
		AccentSoft: ui.Hex("#2b2e34").Alpha(0.09),
		CTA:        ui.Hex("#23262b"),
		CTAText:    ui.Hex("#ffffff"),
		CTAHover:   ui.Hex("#15171b"),

		// the chosen message is a quiet gray with deep text, matching the
		// site's spec (#e4e7ec): near-black with white text read as a heavy bar
		SelectedRow:   ui.Hex("#e4e7ec"),
		SelectedText:  ui.Hex("#18181b"),
		SelectedMuted: ui.RGBA(24, 24, 27, 0.62),
		FocusRing:     ui.Hex("#2b2e34"),

		Danger:  ui.Hex("#d93a35"),
		Success: ui.Hex("#22863a"),
		Warning: ui.Hex("#b57c10"),
		Star:    ui.Hex("#a06c00"),

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
	t.SurfaceHover = p.HoverStrong
	t.SurfacePressed = p.BtnHover
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
	t.Selection = p.Accent.Alpha(0.25)
	t.Focus = p.FocusRing
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
