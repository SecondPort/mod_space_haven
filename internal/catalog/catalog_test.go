package catalog

import (
	"strings"
	"testing"
)

func testCatalog(t *testing.T) *Catalog {
	t.Helper()
	c, err := Embedded()
	if err != nil {
		t.Fatalf("Embedded: %v", err)
	}
	return c
}

func TestEmbeddedTablesLoad(t *testing.T) {
	c := testCatalog(t)

	if got := len(c.Resources()); got < 90 {
		t.Errorf("resources = %d, want at least 90", got)
	}
	if got := len(c.Traits()); got < 20 {
		t.Errorf("traits = %d, want at least 20", got)
	}
	if _, ok := c.Tech(2532); !ok {
		t.Error("tech 2532 (Scanner) missing from the embedded table")
	}
}

func TestResourceNamesFollowLanguage(t *testing.T) {
	c := testCatalog(t)

	if got, want := c.ResourceName(16, Spanish), "Agua"; got != want {
		t.Errorf("Spanish name = %q, want %q", got, want)
	}
	if got, want := c.ResourceName(16, English), "Water"; got != want {
		t.Errorf("English name = %q, want %q", got, want)
	}
}

func TestUnknownResourceFallsBackToItsID(t *testing.T) {
	c := testCatalog(t)

	if got, want := c.ResourceName(999999, Spanish), "Elemento #999999"; got != want {
		t.Errorf("Spanish fallback = %q, want %q", got, want)
	}
	if got, want := c.ResourceName(999999, English), "Item #999999"; got != want {
		t.Errorf("English fallback = %q, want %q", got, want)
	}
}

func TestResourceLabelShowsBothLanguages(t *testing.T) {
	c := testCatalog(t)

	got := c.ResourceLabel(16, Spanish)
	if !strings.Contains(got, "Agua") || !strings.Contains(got, "Water") {
		t.Errorf("label = %q, want it to carry both names", got)
	}
}

func TestSearchMatchesEitherLanguageAndID(t *testing.T) {
	c := testCatalog(t)

	for _, query := range []string{"agua", "Water", "16"} {
		found := false
		for _, r := range c.SearchResources(query) {
			if r.ID == 16 {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("SearchResources(%q) did not return resource 16", query)
		}
	}

	if got := len(c.SearchResources("")); got != len(c.Resources()) {
		t.Errorf("empty query returned %d resources, want all %d", got, len(c.Resources()))
	}
	if got := c.SearchResources("zzzz-no-such-thing"); len(got) != 0 {
		t.Errorf("nonsense query returned %d resources, want 0", len(got))
	}
}

func TestGroupsPartitionWeaponsFromAttachments(t *testing.T) {
	c := testCatalog(t)

	// 3068..3975: scopes and grips are attachments, not weapons.
	if !c.InGroup(GroupAttachments, 3968) {
		t.Error("3968 (Basic Scope) should be an attachment")
	}
	if c.InGroup(GroupWeapons, 3968) {
		t.Error("3968 (Basic Scope) should not be listed as a weapon")
	}
	if !c.InGroup(GroupWeapons, 725) {
		t.Error("725 (Rifle) should be a weapon")
	}
}

func TestGroupOrdersByDisplayName(t *testing.T) {
	c := testCatalog(t)

	ids := c.Group(GroupMedical, Spanish)
	if len(ids) < 2 {
		t.Fatalf("medical group has %d entries, want several", len(ids))
	}
	for i := 1; i < len(ids); i++ {
		prev := strings.ToLower(c.ResourceName(ids[i-1], Spanish))
		cur := strings.ToLower(c.ResourceName(ids[i], Spanish))
		if prev > cur {
			t.Fatalf("group not sorted: %q came before %q", prev, cur)
		}
	}
}

func TestIsClassifiedSeparatesMaterials(t *testing.T) {
	c := testCatalog(t)

	if !c.IsClassified(16) {
		t.Error("water should be classified (food group)")
	}
	if c.IsClassified(157) {
		t.Error("base metals should be unclassified, so they land under materials")
	}
}

func TestTechLabPointsFallBackToMinimalCompletion(t *testing.T) {
	c := testCatalog(t)

	if got, want := c.TechLabPoints(2532), [3]int{150, 40, 10}; got != want {
		t.Errorf("lab points for Scanner = %v, want %v", got, want)
	}
	if got, want := c.TechLabPoints(999999), [3]int{1, 0, 0}; got != want {
		t.Errorf("lab points for an unknown tech = %v, want %v", got, want)
	}
}

func TestNameLookupsFallBackWithoutPanicking(t *testing.T) {
	c := testCatalog(t)

	if got, want := c.Skill(2), "Minería"; got != want {
		t.Errorf("Skill(2) = %q, want %q", got, want)
	}
	if got, want := c.Skill(999), "sk=999"; got != want {
		t.Errorf("Skill(999) = %q, want %q", got, want)
	}
	if got, want := c.Attribute(210), "Bravery"; got != want {
		t.Errorf("Attribute(210) = %q, want %q", got, want)
	}
	if got, want := c.Trait(191), "Hero"; got != want {
		t.Errorf("Trait(191) = %q, want %q", got, want)
	}
	if got, want := c.TechName(999999), "Tech #999999"; got != want {
		t.Errorf("TechName(999999) = %q, want %q", got, want)
	}
}

func TestParseLanguage(t *testing.T) {
	cases := map[string]Language{
		"en": English, "EN": English, " en ": English,
		"es": Spanish, "": Spanish, "fr": Spanish,
	}
	for in, want := range cases {
		if got := ParseLanguage(in); got != want {
			t.Errorf("ParseLanguage(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResourcesCopyIsIndependent(t *testing.T) {
	c := testCatalog(t)

	first := c.Resources()
	first[0].ES = "mutated"

	if c.Resources()[0].ES == "mutated" {
		t.Error("Resources handed out a slice aliasing the catalog's own data")
	}
}
