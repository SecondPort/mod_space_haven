package tui

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// grid is a table plus the identity of each row, because bubbles' table
// addresses rows by index only. Rows whose id is headingID are group labels:
// they are drawn but never selected.
type grid struct {
	table table.Model
	ids   []int
}

func newGrid(cols []table.Column, height int) grid {
	return grid{table: newTable(cols, height)}
}

func (g *grid) setRows(rows []table.Row, ids []int) {
	cursor := g.table.Cursor()
	g.table.SetRows(rows)
	g.ids = ids
	if cursor >= len(ids) {
		cursor = len(ids) - 1
	}
	if cursor < 0 {
		cursor = 0
	}
	g.table.SetCursor(cursor)
	g.normalize(1)
}

// selected returns the id under the cursor, or false on a heading or an empty
// table.
func (g *grid) selected() (int, bool) {
	i := g.table.Cursor()
	if i < 0 || i >= len(g.ids) || g.ids[i] == headingID {
		return 0, false
	}
	return g.ids[i], true
}

// normalize walks off a heading row in the given direction, then the other way
// if it runs out of table.
func (g *grid) normalize(direction int) {
	n := len(g.ids)
	if n == 0 {
		return
	}
	i := g.table.Cursor()
	if i < 0 {
		i = 0
	}
	for i >= 0 && i < n && g.ids[i] == headingID {
		i += direction
	}
	if i < 0 || i >= n {
		i = g.table.Cursor()
		for i >= 0 && i < n && g.ids[i] == headingID {
			i -= direction
		}
	}
	if i >= 0 && i < n {
		g.table.SetCursor(i)
	}
}

// move steps one selectable row up or down.
func (g *grid) move(delta int) {
	n := len(g.ids)
	if n == 0 {
		return
	}
	next := g.table.Cursor() + delta
	for next >= 0 && next < n && g.ids[next] == headingID {
		next += delta
	}
	if next < 0 || next >= n {
		return
	}
	g.table.SetCursor(next)
}

// update forwards a key to the table and keeps the cursor off heading rows.
func (g *grid) update(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		g.move(-1)
		return nil
	case "down", "j":
		g.move(1)
		return nil
	}

	var cmd tea.Cmd
	g.table, cmd = g.table.Update(msg)
	g.normalize(1)
	return cmd
}

func (g *grid) setHeight(h int)             { g.table.SetHeight(h) }
func (g *grid) view() string                { return g.table.View() }
func (g *grid) setColumns(c []table.Column) { g.table.SetColumns(c) }
