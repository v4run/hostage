package tui

import "github.com/charmbracelet/lipgloss"

// Palette holds every color the TUI uses. Swap the active palette via
// SetTheme before constructing the program; the package-level style vars are
// rebuilt from it.
type Palette struct {
	Accent   lipgloss.TerminalColor
	Enabled  lipgloss.TerminalColor
	Disabled lipgloss.TerminalColor
	Muted    lipgloss.TerminalColor
	Fg       lipgloss.TerminalColor
	SelBg    lipgloss.TerminalColor
	Border   lipgloss.TerminalColor
	Warn     lipgloss.TerminalColor
	Danger   lipgloss.TerminalColor
}

// paletteDefault — cool zinc neutrals with sky / green / amber-red accents.
// True-color hex; lipgloss degrades on terminals that don't support truecolor.
var paletteDefault = Palette{
	Accent:   lipgloss.Color("#7DD3FC"),
	Enabled:  lipgloss.Color("#86EFAC"),
	Disabled: lipgloss.Color("#52525B"),
	Muted:    lipgloss.Color("#A1A1AA"),
	Fg:       lipgloss.Color("#E4E4E7"),
	SelBg:    lipgloss.Color("#3F3F46"), // zinc-700 — visibly distinct from typical dark terminal bg
	Border:   lipgloss.Color("#3F3F46"),
	Warn:     lipgloss.Color("#FBBF24"),
	Danger:   lipgloss.Color("#F87171"),
}

// paletteTerminal uses ANSI 16-color slots so each color is themed by the
// user's terminal palette. Fg is NoColor so text inherits the terminal default.
var paletteTerminal = Palette{
	Accent:   lipgloss.Color("14"), // bright cyan
	Enabled:  lipgloss.Color("10"), // bright green
	Disabled: lipgloss.Color("8"),  // bright black
	Muted:    lipgloss.Color("7"),  // white
	Fg:       lipgloss.NoColor{},
	SelBg:    lipgloss.Color("0"), // black — typically distinct from terminal default bg
	Border:   lipgloss.Color("8"),
	Warn:     lipgloss.Color("11"), // bright yellow
	Danger:   lipgloss.Color("9"),  // bright red
}

var (
	styleTitle               lipgloss.Style
	styleTitleStripe         lipgloss.Style
	styleSubtitle            lipgloss.Style
	styleEnabled             lipgloss.Style
	styleDisabled            lipgloss.Style
	styleEntryIP             lipgloss.Style
	styleEntryHost           lipgloss.Style
	styleEntryDim            lipgloss.Style
	styleSelBg               lipgloss.Style
	styleSelGutter           lipgloss.Style
	styleEnabledSel          lipgloss.Style
	styleDisabledSel         lipgloss.Style
	styleEntryIPSel          lipgloss.Style
	styleEntryHostSel        lipgloss.Style
	styleFilterLabel         lipgloss.Style
	styleFilterGlyph         lipgloss.Style
	styleRule                lipgloss.Style
	styleHelp                lipgloss.Style
	styleHelpKey             lipgloss.Style
	styleStatus              lipgloss.Style
	styleStatusDot           lipgloss.Style
	styleError               lipgloss.Style
	styleFormCard            lipgloss.Style
	styleFormTitle           lipgloss.Style
	styleFormLabel           lipgloss.Style
	styleFormCaret           lipgloss.Style
	styleFormCaretDim        lipgloss.Style
	styleDeleteBanner        lipgloss.Style
	styleDeleteStripe        lipgloss.Style
	styleDeleteEntry         lipgloss.Style
	styleScratchHeaderActive lipgloss.Style
	styleScratchHeaderDim    lipgloss.Style
	styleScratchDivider      lipgloss.Style
	styleScratchOnly         lipgloss.Style
)

func init() {
	applyPalette(paletteDefault)
}

// SetTheme selects a named palette. Unknown names fall back to the default
// theme. Call this before tea.NewProgram so styles are ready at first render.
func SetTheme(name string) {
	switch name {
	case "terminal":
		applyPalette(paletteTerminal)
	default:
		applyPalette(paletteDefault)
	}
}

func applyPalette(p Palette) {
	styleTitle = lipgloss.NewStyle().Foreground(p.Accent).Bold(true)
	styleTitleStripe = lipgloss.NewStyle().Foreground(p.Accent)
	styleSubtitle = lipgloss.NewStyle().Foreground(p.Muted)

	styleEnabled = lipgloss.NewStyle().Foreground(p.Enabled)
	styleDisabled = lipgloss.NewStyle().Foreground(p.Disabled)
	styleEntryIP = lipgloss.NewStyle().Foreground(p.Fg)
	styleEntryHost = lipgloss.NewStyle().Foreground(p.Muted)
	styleEntryDim = lipgloss.NewStyle().Foreground(p.Disabled).Italic(true)

	// Selection: accent gutter + IP/hostname recolored to accent. The
	// bullet keeps its enabled/disabled color so state is still readable
	// on the focused row; disabled rows stay dim and the gutter is the
	// sole focus signal for them.
	// Selection: every cell of the focused row carries Background(SelBg),
	// and the spaces between cells are rendered through styleSelBg so the
	// fill is unbroken across the whole row.
	styleSelBg = lipgloss.NewStyle().Background(p.SelBg)
	styleSelGutter = lipgloss.NewStyle().Foreground(p.Accent).Background(p.SelBg).Bold(true)
	styleEnabledSel = lipgloss.NewStyle().Foreground(p.Enabled).Background(p.SelBg)
	styleDisabledSel = lipgloss.NewStyle().Foreground(p.Disabled).Background(p.SelBg)
	styleEntryIPSel = lipgloss.NewStyle().Foreground(p.Accent).Background(p.SelBg).Bold(true)
	styleEntryHostSel = lipgloss.NewStyle().Foreground(p.Accent).Background(p.SelBg)

	styleFilterLabel = lipgloss.NewStyle().Foreground(p.Muted)
	styleFilterGlyph = lipgloss.NewStyle().Foreground(p.Accent)

	styleRule = lipgloss.NewStyle().Foreground(p.Border)

	styleHelp = lipgloss.NewStyle().Foreground(p.Muted)
	styleHelpKey = lipgloss.NewStyle().Foreground(p.Accent).Bold(true)

	styleStatus = lipgloss.NewStyle().Foreground(p.Warn)
	styleStatusDot = lipgloss.NewStyle().Foreground(p.Warn).Bold(true)

	styleError = lipgloss.NewStyle().Foreground(p.Danger)

	styleFormCard = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(p.Border).Padding(0, 1)
	styleFormTitle = lipgloss.NewStyle().Foreground(p.Accent).Bold(true)
	styleFormLabel = lipgloss.NewStyle().Foreground(p.Muted)
	styleFormCaret = lipgloss.NewStyle().Foreground(p.Accent).Bold(true)
	styleFormCaretDim = lipgloss.NewStyle().Foreground(p.Disabled)

	styleDeleteBanner = lipgloss.NewStyle().Foreground(p.Danger).Bold(true)
	styleDeleteStripe = lipgloss.NewStyle().Foreground(p.Danger)
	styleDeleteEntry = lipgloss.NewStyle().Foreground(p.Fg).Bold(true)

	styleScratchHeaderActive = lipgloss.NewStyle().Foreground(p.Accent).Bold(true)
	styleScratchHeaderDim = lipgloss.NewStyle().Foreground(p.Muted).Bold(true)
	styleScratchDivider = lipgloss.NewStyle().Foreground(p.Border)
	styleScratchOnly = lipgloss.NewStyle().Foreground(p.Disabled).Italic(true)
}

// helpItem renders a "[key] label" chip used in the bottom help bar.
func helpItem(key, label string) string {
	return styleHelpKey.Render("["+key+"]") + " " + styleHelp.Render(label)
}

// helpBar joins help items with a muted dot separator.
func helpBar(items ...string) string {
	sep := styleHelp.Render("  ·  ")
	out := ""
	for i, it := range items {
		if i > 0 {
			out += sep
		}
		out += it
	}
	return out
}
