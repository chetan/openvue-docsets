package docset

import (
	"strings"
	"testing"
)

func TestDashTOCAnchors(t *testing.T) {
	p := page{
		Name:    "Button",
		Body:    `<h2 id="basic">Basic</h2><pre>&lt;h2 id="fake"&gt;</pre><h3 id="props">Props</h3>`,
		Entries: []entry{{"Button / Basic", "Section", "button.html#basic"}, {"Button / Props", "Section", "button.html#props"}, {"Button.label", "Property", "button.html#props"}},
	}
	got := referenceTOC(p)
	if strings.Count(got, `class="dashAnchor"`) != 2 {
		t.Fatal(got)
	}
	if !strings.Contains(got, `name="//apple_ref/cpp/Section/Basic"></a><h2 id="basic"`) {
		t.Fatal(got)
	}
	if got := dashAnchor("Section", "API / Props & slots"); !strings.Contains(got, "API%20%2F%20Props%20%26%20slots") {
		t.Fatal(got)
	}
}

func TestComponentBrowser(t *testing.T) {
	pages := []page{
		{Name: "Button", File: "components-button.html", Entries: []entry{{"Button", "Component", "components-button.html"}}},
		{Name: "DatePicker", File: "components-datepicker.html", Entries: []entry{{"DatePicker", "Component", "components-datepicker.html"}}},
		{Name: "Introduction", File: "pages-introduction.html", Entries: []entry{{"Introduction", "Guide", "pages-introduction.html"}}},
	}
	got := componentBrowser("components-datepicker.html", pages)
	if strings.Count(got, `aria-current="page"`) != 1 {
		t.Fatal(got)
	}
	if !strings.Contains(got, `href="components-datepicker.html" aria-current="page"`) {
		t.Fatal(got)
	}
	for _, want := range []string{`aria-label="OpenVue documentation"`, `<h2>Components</h2>`, `<h2>Guides</h2>`, `href="components-button.html"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q: %s", want, got)
		}
	}
}
