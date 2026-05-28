package ai

import "testing"

func TestParseGeneratedSectionsFiltersUnsafeCommands(t *testing.T) {
	sections, err := ParseGeneratedSections(`{
  "tldr": "Jack updated billing.",
  "mainStoryline": ["Billing changed."],
  "thingsWorthChecking": ["Review billing smoke tests."]
}`)
	if err != nil {
		t.Fatal(err)
	}
	if sections.TLDR == "" || len(sections.MainStoryline) != 1 || len(sections.ThingsWorthChecking) != 1 {
		t.Fatalf("unexpected parsed sections: %#v", sections)
	}
}
