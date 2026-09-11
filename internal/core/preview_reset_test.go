package core

import "testing"

func TestPreviewRemovingAnOverrideShowsTheInheritedValue(t *testing.T) {
	fx, _ := previewFixture(t)
	e := fx.openEngine()
	before := e.State().Current
	_, err := e.Apply(Change{Revision: e.State().Revision, Values: Values{
		"background-opacity": {"0.95"}, "font-family": {"", "Maple Mono"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.Preview(Change{Revision: e.State().Revision, Values: Values{
		"background-opacity": nil, "font-family": nil,
	}})
	if err != nil {
		t.Fatal(err)
	}
	graph, err := resolveGraph([]string{harnessPathFromOpen(t, fx)})
	if err != nil {
		t.Fatal(err)
	}
	preview := evaluate(e.defaults, graph.files, themeIndex(e.themes))
	for _, key := range []string{"background-opacity", "font-family"} {
		if !equalStrings(preview[key], before[key]) {
			t.Errorf("preview reset %s = %v, want inherited %v", key, preview[key], before[key])
		}
	}
	if got := e.State().Current.Last("background-opacity"); got != "0.95" {
		t.Fatalf("preview changed the saved configuration: %s", got)
	}
}
