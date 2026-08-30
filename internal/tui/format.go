package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/SecondPort/mod_space_haven/internal/catalog"
)

// humanInt groups thousands the way the interface language does: a dot in
// Spanish, a comma in English.
func humanInt(n int, lang catalog.Language) string {
	separator := "."
	if lang == catalog.English {
		separator = ","
	}

	digits := strconv.Itoa(n)
	sign := ""
	if strings.HasPrefix(digits, "-") {
		sign, digits = "-", digits[1:]
	}
	if len(digits) <= 3 {
		return sign + digits
	}

	var b strings.Builder
	lead := len(digits) % 3
	if lead > 0 {
		b.WriteString(digits[:lead])
	}
	for i := lead; i < len(digits); i += 3 {
		if b.Len() > 0 {
			b.WriteString(separator)
		}
		b.WriteString(digits[i : i+3])
	}
	return sign + b.String()
}

// truncate shortens a line to fit a width, marking the cut with an ellipsis.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	if width == 1 {
		return "…"
	}
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

// headingRow marks a grouping row inside a table. Rows carrying this id are
// labels, not selectable entries.
const headingID = -1

// heading formats a group label to stand out among the data rows.
func heading(label string) string { return "── " + label }

// amountCell renders a stock number. Table cells stay unstyled on purpose: the
// table clips them by rune width, which would cut an ANSI escape in half and
// leak the raw sequence into the view.
func amountCell(n int, lang catalog.Language) string {
	if n == 0 {
		return "·"
	}
	return humanInt(n, lang)
}
