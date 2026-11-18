package docset

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type exampleManifest struct {
	FormatVersion int
	Revision      string
	Pages         []struct {
		Slug     string
		Features []struct{ ID, Label string }
	}
}

var featureID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

func attachExamples(root, source string, pages []page) error {
	for _, file := range []string{"examples.js", "examples.css", "manifest.json"} {
		info, err := os.Stat(filepath.Join(root, file))
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("missing compiled examples (%s); run bun run examples", file)
		}
	}
	raw, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		return err
	}
	var manifest exampleManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	if manifest.FormatVersion != 2 {
		return fmt.Errorf("compiled example format is stale; run bun run examples")
	}
	revision, err := exec.Command("git", "-C", source, "rev-parse", "HEAD").Output()
	if err != nil {
		return err
	}
	if manifest.Revision != strings.TrimSpace(string(revision)) {
		return fmt.Errorf("example source revision differs from documentation; rebuild examples")
	}
	bySlug := map[string]int{}
	for i, p := range pages {
		slug := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(p.File, "components-"), "pages-"), ".html")
		bySlug[slug] = i
	}
	count := 0
	for _, examples := range manifest.Pages {
		i, ok := bySlug[examples.Slug]
		if !ok {
			continue
		}
		if !iconName.MatchString(examples.Slug) {
			return fmt.Errorf("invalid example slug %q", examples.Slug)
		}
		p := &pages[i]
		var rendered strings.Builder
		fmt.Fprintf(&rendered, `<link rel="stylesheet" href="examples.css"><h1>%s</h1><p>Interactive examples from the OpenVue website, bundled locally. Network actions, such as uploads, require a server and are disabled offline.</p><p><a href="#reference-documentation">Skip to reference documentation</a></p><div id="rendered-examples" data-component="%s">`, html.EscapeString(p.Name), examples.Slug)
		for _, feature := range examples.Features {
			if !featureID.MatchString(feature.ID) {
				return fmt.Errorf("invalid feature id %q", feature.ID)
			}
			fmt.Fprintf(&rendered, `<section class="live-section" id="example-%s" data-example="%s">%s<h2>%s</h2><div id="demo-%s"><p class="demo-loading">Loading interactive example…</p></div></section>`, feature.ID, feature.ID, dashAnchor("Sample", feature.Label), html.EscapeString(feature.Label), feature.ID)
			p.Entries = append(p.Entries, entry{p.Name + " / " + feature.Label + " example", "Sample", p.File + "#example-" + feature.ID})
		}
		rendered.WriteString(`</div><div id="example-runtime"></div><noscript>Enable JavaScript in Dash to display the interactive examples.</noscript><hr><section id="reference-documentation">`)
		rendered.WriteString(p.Body)
		rendered.WriteString(`</section><script defer src="examples.js"></script>`)
		p.Body = rendered.String()
		count++
	}
	if count == 0 {
		return fmt.Errorf("no compiled example pages match documentation")
	}
	return nil
}

// Generated assets cannot replace the reference pages, index, or bundled license.
func copyExamples(ctx context.Context, source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if path == source {
			return nil
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular example asset %s", path)
		}
		if _, err := os.Lstat(target); !os.IsNotExist(err) {
			return fmt.Errorf("example asset would replace existing file: %s", relative)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}
