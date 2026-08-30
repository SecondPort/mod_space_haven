package tui

import (
	"errors"
	"strconv"
	"strings"

	"github.com/SecondPort/mod_space_haven/internal/savegame"
)

// Ranges the game accepts. Values are clamped rather than rejected: typing 999
// into a skill should give you the best skill the game allows, not an error.
const (
	maxHealth    = 200
	maxMood      = 100
	maxRest      = 200
	maxSkill     = 10
	minAttribute = 1
	maxAttribute = 8
)

// applyIntent turns a finished prompt into an edit on the save.
func (m *Model) applyIntent(act intent, value string) {
	if act.kind == intentNone {
		return
	}

	if act.kind == intentRename {
		m.applyRename(act, value)
		return
	}

	n, err := strconv.Atoi(value)
	if err != nil {
		m.status.fail(errors.New(value + ": " + m.txt.PromptNewValue))
		return
	}

	switch act.kind {
	case intentCredits:
		m.applyEdit(m.editor.save.SetCredits(clamp(n, 0, 1<<31-1)),
			m.txt.StatusUpdated, m.txt.Credits, humanInt(act.previous, m.lang), humanInt(n, m.lang))

	case intentCargo:
		inserted, err := m.editor.save.SetCargo(act.resourceID, clamp(n, 0, 1<<31-1))
		name := m.cat.ResourceName(act.resourceID, m.lang)
		if inserted {
			m.applyEdit(err, m.txt.StatusInserted, name, humanInt(n, m.lang))
		} else {
			m.applyEdit(err, m.txt.StatusUpdated, name, humanInt(act.previous, m.lang), humanInt(n, m.lang))
		}

	case intentStat:
		v := clamp(n, 0, statCeiling(act.stat))
		m.applyEdit(m.editor.save.SetCharacterStat(act.entID, act.stat, v),
			m.txt.StatusUpdated, act.stat, strconv.Itoa(act.previous), strconv.Itoa(v))

	case intentSkill:
		level := clamp(n, 0, maxSkill)
		skills, err := m.editor.save.CharacterSkills(act.entID)
		if err != nil {
			m.status.fail(err)
			return
		}
		current := skills[act.skillID]
		ceiling := current.Max
		if level > ceiling {
			ceiling = level
		}
		m.applyEdit(m.editor.save.SetCharacterSkill(act.entID, act.skillID, level, ceiling),
			m.txt.StatusUpdated, m.cat.Skill(act.skillID), strconv.Itoa(current.Level), strconv.Itoa(level))

	case intentAttribute:
		points := clamp(n, minAttribute, maxAttribute)
		m.applyEdit(m.editor.save.SetCharacterAttribute(act.entID, act.attrID, points),
			m.txt.StatusUpdated, m.cat.Attribute(act.attrID), strconv.Itoa(act.previous), strconv.Itoa(points))
	}
}

// applyRename walks the two-step rename: first name, then last name.
func (m *Model) applyRename(act intent, value string) {
	if !act.askingLast {
		first := strings.TrimSpace(value)
		if first == "" {
			return
		}
		character, err := m.editor.save.Character(act.entID)
		if err != nil {
			m.status.fail(err)
			return
		}
		m.overlay.openPrompt(m.txt.PromptLast, m.txt.KeysModal, character.LastName, false, intent{
			kind:       intentRename,
			entID:      act.entID,
			firstName:  first,
			askingLast: true,
		})
		return
	}

	err := m.editor.save.SetCharacterName(act.entID, act.firstName, strings.TrimSpace(value))
	name := strings.TrimSpace(act.firstName + " " + strings.TrimSpace(value))
	m.applyEdit(err, m.txt.StatusRenamed, name)
}

// applyEdit reports the outcome of a domain call and refreshes the panels.
func (m *Model) applyEdit(err error, format string, args ...any) {
	if err != nil {
		m.status.fail(err)
		return
	}
	if err := m.editor.reload(); err != nil {
		m.status.fail(err)
		return
	}
	if m.stage == stageDetail {
		m.refreshDetail()
	}
	m.status.ok(format, args...)
}

func statCeiling(stat string) int {
	switch stat {
	case savegame.StatMood:
		return maxMood
	case savegame.StatRest:
		return maxRest
	default:
		return maxHealth
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
