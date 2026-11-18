package docset

import (
	_ "embed"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
)

//go:embed navigation.css
var navigationCSS string

//go:embed navigation.js
var navigationJS string

func dashAnchor(kind, label string) string {
	encoded := strings.ReplaceAll(url.QueryEscape(label), "+", "%20")
	return `<a class="dashAnchor" name="//apple_ref/cpp/` + kind + `/` + encoded + `"></a>`
}

// Native TOC markers must exist in the static file and survive Vue mounting.
func referenceTOC(p page) string {
	body := p.Body
	seen := map[string]int{}
	for _, e := range p.Entries {
		if e.Type != "Section" {
			continue
		}
		_, id, found := strings.Cut(e.Path, "#")
		if !found {
			continue
		}
		label := strings.TrimPrefix(e.Name, p.Name+" / ")
		if strings.Contains(p.Body, `id="rendered-examples"`) {
			label = "Reference / " + label
		}
		seen[label]++
		if seen[label] > 1 {
			label = fmt.Sprintf("%s (%d)", label, seen[label])
		}
		heading := regexp.MustCompile(`<h[1-6] id="` + regexp.QuoteMeta(html.EscapeString(id)) + `"[^>]*>`)
		body = heading.ReplaceAllStringFunc(body, func(tag string) string { return dashAnchor("Section", label) + tag })
	}
	return body
}

// The component browser is outside the Vue host and uses ordinary file links.
func componentBrowser(current string, pages []page) string {
	var nav strings.Builder
	nav.WriteString(`<aside class="component-browser"><details open><summary>Browse OpenVue</summary><nav class="component-list" aria-label="OpenVue documentation">`)
	for _, group := range []struct{ Label, Type string }{{"Components", "Component"}, {"Guides", "Guide"}} {
		fmt.Fprintf(&nav, `<h2>%s</h2><ul>`, group.Label)
		for _, p := range pages {
			if len(p.Entries) == 0 || p.Entries[0].Type != group.Type {
				continue
			}
			active := ""
			if p.File == current {
				active = ` aria-current="page"`
			}
			fmt.Fprintf(&nav, `<li><a href="%s"%s>%s</a></li>`, html.EscapeString(p.File), active, html.EscapeString(p.Name))
		}
		nav.WriteString(`</ul>`)
	}
	nav.WriteString(`</nav></details></aside>`)
	return nav.String()
}

func withNavigation(current string, pages []page, body string) string {
	return `<link rel="stylesheet" href="navigation.css"><a class="skip-content" href="#documentation">Skip to documentation</a><div class="documentation-layout">` +
		componentBrowser(current, pages) + `<div id="documentation" class="documentation-content" tabindex="-1">` + body +
		`</div></div><script defer src="navigation.js"></script>`
}
