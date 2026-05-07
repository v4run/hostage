package tui

import "github.com/charmbracelet/lipgloss"

// Palette — cool zinc neutrals with sky / green / amber-red accents.
// All values are true-color hex; lipgloss degrades to the nearest 256-color
// match on terminals that don't support truecolor.
var (
	colAccent   = lipgloss.Color("#7DD3FC") // sky-300
	colEnabled  = lipgloss.Color("#86EFAC") // green-300
	colDisabled = lipgloss.Color("#52525B") // zinc-600
	colMuted    = lipgloss.Color("#A1A1AA") // zinc-400
	colFg       = lipgloss.Color("#E4E4E7") // zinc-200
	colSelBg    = lipgloss.Color("#27272A") // zinc-800
	colBorder   = lipgloss.Color("#3F3F46") // zinc-700
	colWarn     = lipgloss.Color("#FBBF24") // amber-400
	colDanger   = lipgloss.Color("#F87171") // red-400
)

var (
	// Title: accent stripe + bold label.
	styleTitle = lipgloss.NewStyle().
			Foreground(colAccent).
			Bold(true)

	styleTitleStripe = lipgloss.NewStyle().
				Foreground(colAccent)

	styleSubtitle = lipgloss.NewStyle().
			Foreground(colMuted)

	// List rows.
	styleEnabled   = lipgloss.NewStyle().Foreground(colEnabled)
	styleDisabled  = lipgloss.NewStyle().Foreground(colDisabled)
	styleEntryIP   = lipgloss.NewStyle().Foreground(colFg)
	styleEntryHost = lipgloss.NewStyle().Foreground(colMuted)
	styleEntryDim  = lipgloss.NewStyle().Foreground(colDisabled).Italic(true)

	// Selection: left bar + subtle bg. The bar is colored by the caller
	// (accent for enabled rows, muted for disabled rows) so it stays legible.
	styleSelBg = lipgloss.NewStyle().
			Background(colSelBg)

	styleSelBar = lipgloss.NewStyle().
			Foreground(colAccent).
			Background(colSelBg).
			Bold(true)

	// Filter input chrome.
	styleFilterLabel = lipgloss.NewStyle().
				Foreground(colMuted)

	styleFilterGlyph = lipgloss.NewStyle().
				Foreground(colAccent)

	// Rules / borders.
	styleRule = lipgloss.NewStyle().Foreground(colBorder)

	// Help and status.
	styleHelp = lipgloss.NewStyle().Foreground(colMuted)

	styleHelpKey = lipgloss.NewStyle().
			Foreground(colAccent).
			Bold(true)

	styleStatus = lipgloss.NewStyle().
			Foreground(colWarn)

	styleStatusDot = lipgloss.NewStyle().
			Foreground(colWarn).
			Bold(true)

	styleError = lipgloss.NewStyle().Foreground(colDanger)

	// Add form card.
	styleFormCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colBorder).
			Padding(0, 1)

	styleFormTitle = lipgloss.NewStyle().
			Foreground(colAccent).
			Bold(true)

	styleFormLabel = lipgloss.NewStyle().
			Foreground(colMuted)

	styleFormCaret = lipgloss.NewStyle().
			Foreground(colAccent).
			Bold(true)

	styleFormCaretDim = lipgloss.NewStyle().
				Foreground(colDisabled)

	// Delete confirm banner.
	styleDeleteBanner = lipgloss.NewStyle().
				Foreground(colDanger).
				Bold(true)

	styleDeleteStripe = lipgloss.NewStyle().
				Foreground(colDanger)

	styleDeleteEntry = lipgloss.NewStyle().
				Foreground(colFg).
				Bold(true)

	// Scratch / split pane.
	styleScratchHeaderActive = lipgloss.NewStyle().
					Foreground(colAccent).
					Bold(true)

	styleScratchHeaderDim = lipgloss.NewStyle().
				Foreground(colMuted).
				Bold(true)

	styleScratchDivider = lipgloss.NewStyle().Foreground(colBorder)

	styleScratchOnly = lipgloss.NewStyle().
				Foreground(colDisabled).
				Italic(true)
)

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
