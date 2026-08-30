package savegame

import (
	"os"
	"strings"
	"testing"
)

func load(t *testing.T) *Save {
	t.Helper()
	raw, err := os.ReadFile("testdata/sample_save.xml")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	s, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return s
}

// countOccurrences is how the byte-exactness assertions check that an edit
// touched exactly the region it claimed to touch.
func countOccurrences(s *Save, needle string) int {
	return strings.Count(string(s.Bytes()), needle)
}

func TestParseRejectsAnEmptyDocument(t *testing.T) {
	if _, err := Parse(nil); err == nil {
		t.Error("Parse accepted an empty document")
	}
}

func TestCreditsRoundTrip(t *testing.T) {
	s := load(t)

	if got, want := s.Credits(), 12500; got != want {
		t.Fatalf("Credits = %d, want %d", got, want)
	}
	if err := s.SetCredits(99999); err != nil {
		t.Fatalf("SetCredits: %v", err)
	}
	if got, want := s.Credits(), 99999; got != want {
		t.Errorf("Credits after write = %d, want %d", got, want)
	}
	if !s.Dirty() {
		t.Error("save should be marked dirty after an edit")
	}
}

func TestSetCreditsRejectsNegativeValues(t *testing.T) {
	s := load(t)

	if err := s.SetCredits(-1); err == nil {
		t.Error("SetCredits accepted a negative balance")
	}
	if got := s.Credits(); got != 12500 {
		t.Errorf("credits changed on a rejected edit: %d", got)
	}
}

func TestEditsPreserveEveryOtherByte(t *testing.T) {
	s := load(t)
	before := string(s.Bytes())

	if err := s.SetCredits(1); err != nil {
		t.Fatalf("SetCredits: %v", err)
	}
	after := string(s.Bytes())

	want := strings.Replace(before, `<playerBank ca="12500"/>`, `<playerBank ca="1"/>`, 1)
	if after != want {
		t.Error("an edit changed bytes outside the attribute it targeted")
	}
}

func TestPlayerShipIsFoundThroughTheSettlementFlag(t *testing.T) {
	s := load(t)

	ship, err := s.PlayerShip()
	if err != nil {
		t.Fatalf("PlayerShip: %v", err)
	}
	if ship.SID != "38" {
		t.Errorf("ship sid = %q, want %q", ship.SID, "38")
	}
	if ship.Name != "Nostromo" {
		t.Errorf("ship name = %q, want %q", ship.Name, "Nostromo")
	}

	block := string(s.Bytes()[ship.Start:ship.End])
	if !strings.HasPrefix(block, `<ship sid="38"`) {
		t.Errorf("ship block starts with %.30q", block)
	}
	if !strings.HasSuffix(block, "</ship>") {
		t.Errorf("ship block ends with %q", block[len(block)-20:])
	}
	// The nested shuttle must not be mistaken for the ship's closing tag.
	if !strings.Contains(block, `sid="77"`) {
		t.Error("ship block was cut short at the nested shuttle's closing tag")
	}
	if strings.Contains(block, `sid="12"`) {
		t.Error("ship block leaked into another ship")
	}
}

func TestCargoCountsOnlyContainersOnThePlayerShip(t *testing.T) {
	s := load(t)

	cargo, err := s.Cargo()
	if err != nil {
		t.Fatalf("Cargo: %v", err)
	}

	// 100 in the first container plus 50 in the third. The 7 units sitting in a
	// machine's production buffer are not cargo, and the 999 on the pirate ship
	// belongs to someone else.
	if got, want := cargo[157], 150; got != want {
		t.Errorf("base metals = %d, want %d", got, want)
	}
	if got, want := cargo[16], 40; got != want {
		t.Errorf("water = %d, want %d", got, want)
	}
	if got, want := cargo[178], 5; got != want {
		t.Errorf("hyperfuel on the docked shuttle = %d, want %d", got, want)
	}
}

func TestCargoSeesContainersBehindASelfClosingSibling(t *testing.T) {
	// feat id="3" holds <meta/> before its <inv>. A scan that only looks at the
	// nearest preceding tag misreads that container as a machine buffer and
	// silently drops 50 units.
	s := load(t)

	cargo, err := s.Cargo()
	if err != nil {
		t.Fatalf("Cargo: %v", err)
	}
	if cargo[157] < 150 {
		t.Errorf("base metals = %d; the container after <meta/> was skipped", cargo[157])
	}
}

func TestSetCargoSpreadsTheTotalAcrossContainers(t *testing.T) {
	s := load(t)

	inserted, err := s.SetCargo(157, 151)
	if err != nil {
		t.Fatalf("SetCargo: %v", err)
	}
	if inserted {
		t.Error("SetCargo reported an insertion for a resource already in cargo")
	}

	cargo, _ := s.Cargo()
	if got, want := cargo[157], 151; got != want {
		t.Errorf("total after write = %d, want %d", got, want)
	}

	out := string(s.Bytes())
	if !strings.Contains(out, `<s elementaryId="157" inStorage="76"`) ||
		!strings.Contains(out, `<s elementaryId="157" inStorage="75"`) {
		t.Error("the remainder was not distributed across the two containers")
	}
	if !strings.Contains(out, `<s elementaryId="157" inStorage="7"`) {
		t.Error("SetCargo overwrote a machine production buffer")
	}
	if !strings.Contains(out, `<s elementaryId="157" inStorage="999"`) {
		t.Error("SetCargo reached into another ship's cargo")
	}
}

func TestSetCargoToZero(t *testing.T) {
	s := load(t)

	if _, err := s.SetCargo(16, 0); err != nil {
		t.Fatalf("SetCargo: %v", err)
	}
	cargo, _ := s.Cargo()
	if got := cargo[16]; got != 0 {
		t.Errorf("water = %d, want 0", got)
	}
}

func TestSetCargoInsertsAResourceThatIsNotThereYet(t *testing.T) {
	s := load(t)

	inserted, err := s.SetCargo(2475, 250)
	if err != nil {
		t.Fatalf("SetCargo: %v", err)
	}
	if !inserted {
		t.Error("SetCargo did not report the insertion")
	}

	cargo, _ := s.Cargo()
	if got, want := cargo[2475], 250; got != want {
		t.Errorf("fertilizer = %d, want %d", got, want)
	}
	if got := countOccurrences(s, `elementaryId="2475"`); got != 1 {
		t.Errorf("resource written %d times, want exactly 1", got)
	}
	// It belongs in the fullest cargo container, not in a machine buffer.
	out := string(s.Bytes())
	idx := strings.Index(out, `elementaryId="2475"`)
	if before := out[:idx]; strings.LastIndex(before, "<prod>") > strings.LastIndex(before, "</prod>") {
		t.Error("the new resource landed inside a machine production buffer")
	}
}

func TestSetCargoRejectsNegativeAmounts(t *testing.T) {
	s := load(t)

	if _, err := s.SetCargo(157, -5); err == nil {
		t.Error("SetCargo accepted a negative amount")
	}
}

func TestCharactersReturnsThePlayerCrewOnly(t *testing.T) {
	s := load(t)

	crew, err := s.Characters()
	if err != nil {
		t.Fatalf("Characters: %v", err)
	}
	if len(crew) != 2 {
		t.Fatalf("crew size = %d, want 2", len(crew))
	}
	if got, want := crew[0].FullName(), "Ellen Ripley"; got != want {
		t.Errorf("first crew member = %q, want %q", got, want)
	}
	if got, want := crew[1].EntID, "1002"; got != want {
		t.Errorf("second entId = %q, want %q", got, want)
	}
	for _, c := range crew {
		if c.EntID == "2001" {
			t.Error("an enemy character was listed as crew")
		}
	}
}

func TestCharacterBlockStopsAtItsOwnClosingTag(t *testing.T) {
	s := load(t)

	c, err := s.Character("1001")
	if err != nil {
		t.Fatalf("Character: %v", err)
	}
	block := string(s.Bytes()[c.Start:c.End])

	if !strings.Contains(block, "Ripley") {
		t.Error("character block does not contain its own name")
	}
	if strings.Contains(block, "Dallas") {
		t.Error("character block leaked into the next crew member")
	}
}

func TestUnknownCharacterIsReported(t *testing.T) {
	s := load(t)

	if _, err := s.Character("does-not-exist"); err == nil {
		t.Error("Character accepted an unknown entId")
	}
	if err := s.SetCharacterStat("does-not-exist", StatHealth, 10); err == nil {
		t.Error("SetCharacterStat accepted an unknown entId")
	}
}

func TestCharacterStatsRoundTrip(t *testing.T) {
	s := load(t)

	stats, err := s.CharacterStats("1001")
	if err != nil {
		t.Fatalf("CharacterStats: %v", err)
	}
	if stats.Health != 80 || stats.Mood != 55 || stats.Rest != 70 {
		t.Fatalf("stats = %+v, want {80 55 70}", stats)
	}

	if err := s.SetCharacterStat("1001", StatMood, 100); err != nil {
		t.Fatalf("SetCharacterStat: %v", err)
	}
	stats, _ = s.CharacterStats("1001")
	if stats.Mood != 100 {
		t.Errorf("mood = %d, want 100", stats.Mood)
	}
	if stats.Health != 80 {
		t.Errorf("writing mood changed health to %d", stats.Health)
	}

	// The enemy's identical stat block must be untouched.
	if !strings.Contains(string(s.Bytes()), `<Mood v="0"/>`) {
		t.Error("the edit reached another character's stats")
	}
}

func TestCharacterSkillsRoundTrip(t *testing.T) {
	s := load(t)

	skills, err := s.CharacterSkills("1001")
	if err != nil {
		t.Fatalf("CharacterSkills: %v", err)
	}
	if got, want := skills[2].Level, 3; got != want {
		t.Errorf("mining level = %d, want %d", got, want)
	}
	if got, want := skills[4].Max, 9; got != want {
		t.Errorf("construction max = %d, want %d", got, want)
	}

	if err := s.SetCharacterSkill("1001", 2, 10, 10); err != nil {
		t.Fatalf("SetCharacterSkill: %v", err)
	}
	skills, _ = s.CharacterSkills("1001")
	if skills[2] != (SkillLevel{Level: 10, Max: 10}) {
		t.Errorf("mining = %+v, want {10 10}", skills[2])
	}
	if skills[4].Level != 7 {
		t.Errorf("writing one skill changed another: %+v", skills[4])
	}
}

func TestSetCharacterSkillRejectsAnUnknownSkill(t *testing.T) {
	s := load(t)

	if err := s.SetCharacterSkill("1001", 999, 5, 5); err == nil {
		t.Error("SetCharacterSkill accepted a skill the character does not have")
	}
}

func TestCharacterAttributesRoundTrip(t *testing.T) {
	s := load(t)

	attrs, err := s.CharacterAttributes("1001")
	if err != nil {
		t.Fatalf("CharacterAttributes: %v", err)
	}
	if got, want := attrs[210], 5; got != want {
		t.Errorf("bravery = %d, want %d", got, want)
	}

	if err := s.SetCharacterAttribute("1001", 210, 8); err != nil {
		t.Fatalf("SetCharacterAttribute: %v", err)
	}
	attrs, _ = s.CharacterAttributes("1001")
	if attrs[210] != 8 {
		t.Errorf("bravery = %d, want 8", attrs[210])
	}
	if attrs[213] != 3 {
		t.Errorf("writing one attribute changed another: %d", attrs[213])
	}
}

func TestCharacterTraitsAddAndRemove(t *testing.T) {
	s := load(t)

	traits, err := s.CharacterTraits("1001")
	if err != nil {
		t.Fatalf("CharacterTraits: %v", err)
	}
	if len(traits) != 2 || traits[0] != 191 || traits[1] != 1035 {
		t.Fatalf("traits = %v, want [191 1035]", traits)
	}

	if err := s.AddCharacterTrait("1001", 1039); err != nil {
		t.Fatalf("AddCharacterTrait: %v", err)
	}
	traits, _ = s.CharacterTraits("1001")
	if len(traits) != 3 {
		t.Fatalf("traits after add = %v, want three entries", traits)
	}

	if err := s.RemoveCharacterTrait("1001", 191); err != nil {
		t.Fatalf("RemoveCharacterTrait: %v", err)
	}
	traits, _ = s.CharacterTraits("1001")
	if len(traits) != 2 || traits[0] != 1035 || traits[1] != 1039 {
		t.Errorf("traits after remove = %v, want [1035 1039]", traits)
	}

	// The other crew member keeps their own trait list.
	other, _ := s.CharacterTraits("1002")
	if len(other) != 1 || other[0] != 1041 {
		t.Errorf("another character's traits changed: %v", other)
	}
}

func TestAddCharacterTraitIsIdempotent(t *testing.T) {
	s := load(t)

	if err := s.AddCharacterTrait("1001", 191); err == nil {
		t.Error("AddCharacterTrait accepted a trait the character already has")
	}
	traits, _ := s.CharacterTraits("1001")
	if len(traits) != 2 {
		t.Errorf("traits = %v, want the original two", traits)
	}
}

func TestRemoveCharacterTraitReportsAMissingTrait(t *testing.T) {
	s := load(t)

	if err := s.RemoveCharacterTrait("1001", 4242); err == nil {
		t.Error("RemoveCharacterTrait accepted a trait the character does not have")
	}
}

func TestAddedTraitKeepsSurroundingIndentation(t *testing.T) {
	s := load(t)

	if err := s.AddCharacterTrait("1001", 1039); err != nil {
		t.Fatalf("AddCharacterTrait: %v", err)
	}
	out := string(s.Bytes())
	if !strings.Contains(out, "\t\t\t\t\t<t id=\"1039\"/>\n\t\t\t\t</traits>") {
		idx := strings.Index(out, `<t id="1039"/>`)
		t.Errorf("new trait not indented like its siblings: %q", out[idx-12:idx+22])
	}
}

func TestSetCharacterName(t *testing.T) {
	s := load(t)

	if err := s.SetCharacterName("1001", "Ripley", "Amanda"); err != nil {
		t.Fatalf("SetCharacterName: %v", err)
	}

	c, _ := s.Character("1001")
	if got, want := c.FullName(), "Ripley Amanda"; got != want {
		t.Errorf("name = %q, want %q", got, want)
	}
	if !strings.Contains(string(s.Bytes()), `name="Arthur" lname="Dallas"`) {
		t.Error("renaming one character changed another")
	}
}

func TestSetCharacterNameEscapesMarkup(t *testing.T) {
	s := load(t)

	if err := s.SetCharacterName("1001", `A&B`, `<Q>`); err != nil {
		t.Fatalf("SetCharacterName: %v", err)
	}
	out := string(s.Bytes())
	if !strings.Contains(out, `name="A&amp;B" lname="&lt;Q&gt;"`) {
		t.Error("a name containing markup was written unescaped")
	}

	c, _ := s.Character("1001")
	if got, want := c.FullName(), "A&B <Q>"; got != want {
		t.Errorf("round-tripped name = %q, want %q", got, want)
	}
}

func TestSetCharacterNameRejectsAnEmptyFirstName(t *testing.T) {
	s := load(t)

	if err := s.SetCharacterName("1001", "  ", "Ripley"); err == nil {
		t.Error("SetCharacterName accepted a blank first name")
	}
}

func TestCharacterLoadout(t *testing.T) {
	s := load(t)

	lo, err := s.CharacterLoadout("1001")
	if err != nil {
		t.Fatalf("CharacterLoadout: %v", err)
	}
	if lo.Primary != 725 || lo.Armor != 3383 || lo.Headgear != 481 || lo.Secondary != 760 {
		t.Errorf("loadout = %+v", lo)
	}

	empty, err := s.CharacterLoadout("1002")
	if err != nil {
		t.Fatalf("CharacterLoadout: %v", err)
	}
	if empty.Primary != 0 {
		t.Errorf("empty loadout reported %d in the primary slot", empty.Primary)
	}
}

func TestCharacterWithoutALoadoutIsNotAnError(t *testing.T) {
	s := load(t)

	// The enemy has no <loadout>; asking for one yields empty slots, not a failure.
	if _, err := s.CharacterLoadout("2001"); err != nil {
		t.Errorf("CharacterLoadout on a character without a loadout: %v", err)
	}
}

func TestResearchReadsEveryEntry(t *testing.T) {
	s := load(t)

	research, err := s.Research()
	if err != nil {
		t.Fatalf("Research: %v", err)
	}
	if len(research) != 3 {
		t.Fatalf("research entries = %d, want 3", len(research))
	}
	if research[2532] {
		t.Error("2532 should be pending")
	}
	if !research[2533] {
		t.Error("2533 should be done")
	}
}

func TestSetTechDoneWritesStateAndBlocks(t *testing.T) {
	s := load(t)

	if err := s.SetTechDone(2532, true, [3]int{150, 40, 10}); err != nil {
		t.Fatalf("SetTechDone: %v", err)
	}

	research, _ := s.Research()
	if !research[2532] {
		t.Error("2532 is still pending after completion")
	}

	out := string(s.Bytes())
	if !strings.Contains(out, `<blocksDone level1="150" level2="40" level3="10"/>`) {
		t.Error("lab point blocks were not written")
	}
	// Neighbouring entries keep their own state.
	if !strings.Contains(out, `<blocksDone level1="180" level2="40" level3="12"/>`) {
		t.Error("completing one technology changed another's blocks")
	}
	if strings.Count(out, `done="false"`) != 1 {
		t.Errorf("expected exactly one pending technology left, got %d", strings.Count(out, `done="false"`))
	}
}

func TestSetTechDoneResets(t *testing.T) {
	s := load(t)

	if err := s.SetTechDone(2533, false, [3]int{180, 40, 12}); err != nil {
		t.Fatalf("SetTechDone: %v", err)
	}

	research, _ := s.Research()
	if research[2533] {
		t.Error("2533 is still done after a reset")
	}
	if !strings.Contains(string(s.Bytes()), `<blocksDone level1="0" level2="0" level3="0"/>`) {
		t.Error("a reset should zero the lab point blocks")
	}
}

func TestSetTechDoneReportsAnUnknownTechnology(t *testing.T) {
	s := load(t)

	if err := s.SetTechDone(999999, true, [3]int{1, 0, 0}); err == nil {
		t.Error("SetTechDone accepted a technology that is not in the save")
	}
}

func TestRewritingTheSameValueLeavesTheSaveClean(t *testing.T) {
	s := load(t)
	before := string(s.Bytes())

	if err := s.SetCredits(12500); err != nil {
		t.Fatalf("SetCredits: %v", err)
	}
	if _, err := s.SetCargo(16, 40); err != nil {
		t.Fatalf("SetCargo: %v", err)
	}
	if err := s.SetCharacterStat("1001", StatHealth, 80); err != nil {
		t.Fatalf("SetCharacterStat: %v", err)
	}

	if s.Dirty() {
		t.Error("writing back the values already in the save should not dirty it")
	}
	if string(s.Bytes()) != before {
		t.Error("a no-op write changed the document")
	}
}

func TestDirtyTracksUnsavedWork(t *testing.T) {
	s := load(t)

	if s.Dirty() {
		t.Error("a freshly parsed save should be clean")
	}
	if err := s.SetCredits(1); err != nil {
		t.Fatalf("SetCredits: %v", err)
	}
	if !s.Dirty() {
		t.Error("save should be dirty after an edit")
	}

	s.MarkSaved()
	if s.Dirty() {
		t.Error("save should be clean after MarkSaved")
	}
}

func TestBytesHandsOutACopy(t *testing.T) {
	s := load(t)

	b := s.Bytes()
	b[0] = 'X'

	if s.Bytes()[0] == 'X' {
		t.Error("Bytes exposed the save's own buffer")
	}
}
