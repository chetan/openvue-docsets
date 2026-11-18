package docset

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMarkdownIndex(t *testing.T) {
	source := []byte("# Button\n\n```vue\n<Button label=\"Save\" />\n## Fake\n```\n\n## Button\n\n### Props\n\n| Name | Type |\n| --- | --- |\n| label | string |\n\n### Events\n\n| Name | Type |\n| --- | --- |\n| click | Event |\n\n## Basic\n\n## Basic\n\n[Next](./dialog)\n")
	p, err := markdownPage("Button", "button.html", "https://openvue.dev/button", source, "Component")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []entry{{"Button.label", "Property", "button.html#props"}, {"Button.click", "Event", "button.html#events"}, {"Button / Basic", "Section", "button.html#basic-1"}} {
		found := false
		for _, e := range p.Entries {
			if e == want {
				found = true
			}
		}
		if !found {
			t.Errorf("missing %#v: %#v", want, p.Entries)
		}
	}
	for _, e := range p.Entries {
		if strings.Contains(e.Name, "Fake") {
			t.Fatal("indexed a code example")
		}
	}
	if !strings.Contains(p.Body, `&lt;Button`) || !strings.Contains(p.Body, `href="https://openvue.dev/dialog"`) {
		t.Fatal(p.Body)
	}
}

func TestBuild(t *testing.T) {
	root := t.TempDir()
	vue, icons := filepath.Join(root, "vue"), filepath.Join(root, "icons")
	write := func(base, path, body string) {
		t.Helper()
		full := filepath.Join(base, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write(vue, "LICENSE.md", "Vue MIT license")
	write(vue, "apps/showcase/server/assets/llms/llms.txt", "[Button](https://openvue.dev/button)")
	write(vue, "apps/showcase/server/assets/llms/components/button.md", "# Button\n\n```js\nimport Button from 'openvue/button';\n```\n\n## Button\n\n### Props\n\n| Name | Type |\n| --- | --- |\n| label | string |\n")
	write(icons, "LICENSE", "Icons MIT license")
	write(icons, "selection.json", `{"icons":[{"icon":{"tags":["save"]},"properties":{"name":"check","code":59657}}]}`)
	write(icons, "raw-svg/check.svg", `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M1 1"/></svg>`)
	write(icons, "docs/guide/vue.md", "# Vue components\n\n[Install](./installation)\n")
	write(icons, "docs/guide/installation.md", "# Installation\n")
	for _, dir := range []string{vue, icons} {
		for _, args := range [][]string{{"init", dir}, {"-C", dir, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-m", "fixture"}} {
			if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
				t.Fatalf("%v: %s", err, out)
			}
		}
	}
	cfg := Config{OpenVue: vue, OpenIcons: icons, Output: filepath.Join(root, "dist")}
	if err := Build(context.Background(), cfg, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"OpenVue", "OpenIcons"} {
		base := filepath.Join(cfg.Output, name+".docset", "Contents/Resources")
		db, err := sql.Open("sqlite", filepath.Join(base, "docSet.dsidx"))
		if err != nil {
			t.Fatal(err)
		}
		rows, err := db.Query("SELECT path FROM searchIndex")
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for rows.Next() {
			var path string
			if err := rows.Scan(&path); err != nil {
				t.Fatal(err)
			}
			file, anchor, _ := strings.Cut(path, "#")
			content, err := os.ReadFile(filepath.Join(base, "Documents", file))
			if err != nil {
				t.Fatal(err)
			}
			if anchor != "" && !strings.Contains(string(content), `id="`+anchor+`"`) {
				t.Errorf("missing anchor: %s", path)
			}
			count++
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		rows.Close()
		db.Close()
		if count == 0 {
			t.Fatal("empty index")
		}
	}
	svg, err := os.ReadFile(filepath.Join(cfg.Output, "OpenIcons.docset/Contents/Resources/Documents/check.svg"))
	if err != nil || !strings.Contains(string(svg), "<svg") {
		t.Fatal("missing SVG", err)
	}
	guide, err := os.ReadFile(filepath.Join(cfg.Output, "OpenIcons.docset/Contents/Resources/Documents/guide-vue.html"))
	if err != nil || !strings.Contains(string(guide), `href="guide-installation.html"`) {
		t.Fatal("missing local link", err)
	}
	if err := Build(context.Background(), cfg, io.Discard); err == nil {
		t.Fatal("overwrote existing bundle")
	}
	examples := filepath.Join(root, "examples")
	revision, err := exec.Command("git", "-C", vue, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	manifest := `{"formatVersion":2,"revision":"` + strings.TrimSpace(string(revision)) + `","pages":[{"slug":"button","features":[{"id":"basic","label":"Basic"}]}]}`
	write(examples, "manifest.json", manifest)
	write(examples, "examples.js", "/* compiled fixture */")
	write(examples, "examples.css", "/* fixture styles */")
	write(examples, "media/image.svg", "<svg/>")
	cfg.Examples = examples
	cfg.Output = filepath.Join(root, "interactive")
	if err := Build(context.Background(), cfg, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"OpenVue", "OpenIcons"} {
		plist, err := os.ReadFile(filepath.Join(cfg.Output, name+".docset/Contents/Info.plist"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(plist), "<key>isJavaScriptEnabled</key>") != (name == "OpenVue") {
			t.Fatal("incorrect JavaScript setting", name)
		}
	}
	body, err := os.ReadFile(filepath.Join(cfg.Output, "OpenVue.docset/Contents/Resources/Documents/components-button.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`id="example-basic"`, `data-component="button"`, `src="examples.js"`, `connect-src 'none'`, `id="props"`, `id="example-runtime"`, `id="demo-basic"`, `//apple_ref/cpp/Sample/Basic`, `//apple_ref/cpp/Section/Reference%20%2F%20Props`, `aria-current="page"`} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("missing %s", want)
		}
	}
	plist, err := os.ReadFile(filepath.Join(cfg.Output, "OpenVue.docset/Contents/Info.plist"))
	if err != nil || !strings.Contains(string(plist), `<key>DashDocSetFamily</key><string>dashtoc</string>`) {
		t.Fatal("missing native TOC metadata", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.Output, "OpenVue.docset/Contents/Resources/Documents/media/image.svg")); err != nil {
		t.Fatal(err)
	}
	write(examples, "manifest.json", strings.Replace(manifest, strings.TrimSpace(string(revision)), "wrong", 1))
	cfg.Output = filepath.Join(root, "mismatched")
	if err := Build(context.Background(), cfg, io.Discard); err == nil {
		t.Fatal("accepted mismatched source revision")
	}
	write(examples, "manifest.json", strings.Replace(manifest, `"formatVersion":2`, `"formatVersion":1`, 1))
	if err := Build(context.Background(), cfg, io.Discard); err == nil {
		t.Fatal("accepted stale renderer format")
	}
}

func TestCanceledAndMissingSources(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Build(ctx, Config{}, io.Discard); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation not reported: %v", err)
	}
	if err := Build(context.Background(), Config{OpenVue: t.TempDir()}, io.Discard); err == nil {
		t.Fatal("missing sources accepted")
	}
}
