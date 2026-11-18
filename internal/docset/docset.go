package docset

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	_ "modernc.org/sqlite"
)

type Config struct{ OpenVue, OpenIcons, Output, Examples string }
type entry struct{ Name, Type, Path string }
type page struct {
	Name, File, Online, Body string
	Entries                  []entry
	Assets                   map[string][]byte
}

// Build publishes complete bundles only. Existing bundles are never removed.
func Build(ctx context.Context, cfg Config, log io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	vue, err := componentPages(cfg.OpenVue)
	if err != nil {
		return err
	}
	icons, err := iconPages(cfg.OpenIcons)
	if err != nil {
		return err
	}
	if len(vue) == 0 || len(icons) == 0 {
		return fmt.Errorf("source checkouts contain no documentation")
	}
	if cfg.Examples != "" {
		if err := attachExamples(cfg.Examples, cfg.OpenVue, vue); err != nil {
			return err
		}
	}
	for _, name := range []string{"OpenVue", "OpenIcons"} {
		if _, err := os.Lstat(filepath.Join(cfg.Output, name+".docset")); !os.IsNotExist(err) {
			return fmt.Errorf("%s.docset already exists or cannot be inspected; choose a new --output directory", name)
		}
	}
	if err := os.MkdirAll(cfg.Output, 0755); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(cfg.Output, ".build-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	for _, set := range []struct {
		Name, Source, License string
		Pages                 []page
	}{
		{"OpenVue", cfg.OpenVue, "LICENSE.md", vue},
		{"OpenIcons", cfg.OpenIcons, "LICENSE", icons},
	} {
		if err := ctx.Err(); err != nil {
			return err
		}
		license, err := os.ReadFile(filepath.Join(set.Source, set.License))
		if err != nil {
			return fmt.Errorf("read %s license: %w", set.Name, err)
		}
		sha, err := exec.CommandContext(ctx, "git", "-C", set.Source, "rev-parse", "HEAD").Output()
		if err != nil {
			return fmt.Errorf("read source revision: %w", err)
		}
		if err := writeBundle(ctx, stage, set.Name, set.Pages, string(license), strings.TrimSpace(string(sha))); err != nil {
			return err
		}
		if set.Name == "OpenVue" && cfg.Examples != "" {
			docs := filepath.Join(stage, "OpenVue.docset/Contents/Resources/Documents")
			if err := copyExamples(ctx, cfg.Examples, docs); err != nil {
				return err
			}
			plistPath := filepath.Join(stage, "OpenVue.docset/Contents/Info.plist")
			plist, err := os.ReadFile(plistPath)
			if err != nil {
				return err
			}
			plist = bytes.Replace(plist, []byte("</dict>"), []byte("<key>isJavaScriptEnabled</key><true/>\n</dict>"), 1)
			if err := os.WriteFile(plistPath, plist, 0644); err != nil {
				return err
			}
		}
	}
	for _, name := range []string{"OpenVue", "OpenIcons"} {
		if err := os.Rename(filepath.Join(stage, name+".docset"), filepath.Join(cfg.Output, name+".docset")); err != nil {
			return err
		}
	}
	fmt.Fprintf(log, "Built OpenVue (%d pages) and OpenIcons (%d pages) in %s\n", len(vue), len(icons), cfg.Output)
	return nil
}

var md = goldmark.New(goldmark.WithExtensions(extension.GFM), goldmark.WithParserOptions(parser.WithAutoHeadingID()))
var importName = regexp.MustCompile(`import ([A-Z][A-Za-z0-9]*) from ['"]openvue/`)
var iconName = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var documentationURL = regexp.MustCompile(`https://openvue\.dev/[a-zA-Z0-9/-]+`)

func componentPages(root string) ([]page, error) {
	base := filepath.Join(root, "apps/showcase/server/assets/llms")
	manifest, err := os.ReadFile(filepath.Join(base, "llms.txt"))
	if err != nil {
		return nil, fmt.Errorf("read OpenVue page manifest: %w", err)
	}
	urls := map[string]string{}
	for _, online := range documentationURL.FindAllString(string(manifest), -1) {
		urls[filepath.Base(online)] = online
	}
	var pages []page
	for _, group := range []string{"components", "pages"} {
		files, err := filepath.Glob(filepath.Join(base, group, "*.md"))
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			source, err := os.ReadFile(file)
			if err != nil {
				return nil, err
			}
			slug := strings.TrimSuffix(filepath.Base(file), ".md")
			name := strings.TrimSpace(strings.TrimPrefix(strings.SplitN(string(source), "\n", 2)[0], "#"))
			kind := "Guide"
			if group == "components" {
				if m := importName.FindSubmatch(source); m != nil {
					name = string(m[1])
					kind = "Component"
				}
			}
			online := urls[slug]
			if online == "" {
				online = "https://openvue.dev/" + slug
			}
			p, err := markdownPage(name, group+"-"+slug+".html", online, source, kind)
			if err != nil {
				return nil, err
			}
			pages = append(pages, p)
		}
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no OpenVue Markdown found under %s", base)
	}
	return pages, nil
}

// markdownPage indexes real AST headings and API table rows, never code examples.
func markdownPage(name, file, online string, source []byte, kind string) (page, error) {
	p := page{Name: name, File: file, Online: online, Entries: []entry{{name, kind, file}}}
	doc := md.Parser().Parse(text.NewReader(source))
	base, err := url.Parse(online)
	if err != nil {
		return p, err
	}
	section, anchor, owner := "", "", name
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if link, ok := n.(*ast.Link); ok && !strings.HasPrefix(string(link.Destination), "#") {
			if dest, err := url.Parse(string(link.Destination)); err == nil {
				link.Destination = []byte(base.ResolveReference(dest).String())
			}
		}
		if image, ok := n.(*ast.Image); ok {
			if dest, err := url.Parse(string(image.Destination)); err == nil {
				image.Destination = []byte(base.ResolveReference(dest).String())
			}
		}
		if h, ok := n.(*ast.Heading); ok {
			label := string(h.Text(source))
			if h.Level == 2 {
				owner = label
			}
			section = label
			id, ok := h.AttributeString("id")
			if ok {
				anchor = string(id.([]byte))
				p.Entries = append(p.Entries, entry{name + " / " + label, "Section", file + "#" + anchor})
			}
		}
		if row, ok := n.(*extast.TableRow); ok && row.FirstChild() != nil {
			types := map[string]string{"Props": "Property", "Events": "Event", "Emits": "Event", "Slots": "Method", "Design Tokens": "Variable", "Pass Through Options": "Property"}
			if entryType := types[section]; entryType != "" {
				member := strings.TrimSpace(string(row.FirstChild().Text(source)))
				if member != "" {
					memberOwner := name
					if kind == "Component" && (section == "Props" || section == "Events" || section == "Emits" || section == "Slots") {
						memberOwner = owner
					}
					p.Entries = append(p.Entries, entry{memberOwner + "." + member, entryType, file + "#" + anchor})
				}
			}
		}
		return ast.WalkContinue, nil
	})
	var out bytes.Buffer
	if err := md.Renderer().Render(&out, source, doc); err != nil {
		return p, err
	}
	p.Body = out.String()
	return p, nil
}

func iconPages(root string) ([]page, error) {
	var selection struct {
		Icons []struct {
			Icon       struct{ Tags []string }
			Properties struct {
				Name string
				Code int
			}
		}
	}
	raw, err := os.ReadFile(filepath.Join(root, "selection.json"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &selection); err != nil {
		return nil, err
	}
	var pages []page
	for _, icon := range selection.Icons {
		name := icon.Properties.Name
		if !iconName.MatchString(name) {
			return nil, fmt.Errorf("invalid icon name %q", name)
		}
		svg, err := os.ReadFile(filepath.Join(root, "raw-svg", name+".svg"))
		if err != nil {
			return nil, err
		}
		component := "Oi"
		for _, part := range strings.Split(name, "-") {
			component += strings.ToUpper(part[:1]) + part[1:]
		}
		file := "icon-" + name + ".html"
		// SVG is an image resource, not executable inline markup.
		body := fmt.Sprintf(`<h1>%s</h1><img width="96" height="96" alt="%s" src="%s.svg"><p>Codepoint: U+%04X</p><p>Keywords: %s</p><h2>Webfont</h2><pre><code>%s</code></pre><h2>Vue</h2><pre><code>%s</code></pre>`,
			html.EscapeString(name), html.EscapeString(name), name, icon.Properties.Code, html.EscapeString(strings.Join(icon.Icon.Tags, ", ")),
			html.EscapeString(`<i class="oi oi-`+name+`" aria-hidden="true"></i>`),
			html.EscapeString("import "+component+" from '@openvue/openicons-vue/icons/"+component+"';\n\n<"+component+` :size="24" title="`+name+`" />`))
		p := page{Name: name, File: file, Online: "https://openicons.openvue.dev/icons/" + name, Body: body, Entries: []entry{{name, "Constant", file}, {"oi-" + name, "Constant", file}, {component, "Component", file}}}
		p.Assets = map[string][]byte{name + ".svg": svg}
		pages = append(pages, p)
	}
	files, err := filepath.Glob(filepath.Join(root, "docs/guide", "*.md"))
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		source, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		slug := strings.TrimSuffix(filepath.Base(file), ".md")
		name := strings.TrimSpace(strings.TrimPrefix(strings.SplitN(string(source), "\n", 2)[0], "#"))
		p, err := markdownPage(name, "guide-"+slug+".html", "https://openicons.openvue.dev/guide/"+slug, source, "Guide")
		if err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, nil
}

func writeBundle(ctx context.Context, output, name string, pages []page, license, revision string) error {
	root := filepath.Join(output, name+".docset")
	docs := filepath.Join(root, "Contents/Resources/Documents")
	if err := os.MkdirAll(docs, 0755); err != nil {
		return err
	}
	write := func(path, content string) error { return os.WriteFile(path, []byte(content), 0644) }
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>io.bixby.%s</string>
<key>CFBundleName</key><string>%s</string>
<key>DocSetPlatformFamily</key><string>%s</string>
<key>isDashDocset</key><true/>
<key>dashIndexFilePath</key><string>index.html</string>
<key>DashDocSetDefaultFTSEnabled</key><true/>
</dict></plist>`, strings.ToLower(name), name, strings.ToLower(name))
	if err := write(filepath.Join(root, "Contents/Info.plist"), plist); err != nil {
		return err
	}
	if err := write(filepath.Join(docs, "style.css"), stylesheet); err != nil {
		return err
	}
	if err := write(filepath.Join(docs, "LICENSE.txt"), license); err != nil {
		return err
	}
	if err := write(filepath.Join(docs, "revision.txt"), revision+"\n"); err != nil {
		return err
	}
	db, err := sql.Open("sqlite", filepath.Join(root, "Contents/Resources/docSet.dsidx"))
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `CREATE TABLE searchIndex(id INTEGER PRIMARY KEY, name TEXT, type TEXT, path TEXT); CREATE UNIQUE INDEX anchor ON searchIndex(name,type,path)`); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, `INSERT OR IGNORE INTO searchIndex(name,type,path) VALUES(?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	sort.Slice(pages, func(i, j int) bool { return pages[i].Name < pages[j].Name })
	localLinks := map[string]string{}
	for _, p := range pages {
		localLinks[p.Online] = p.File
	}
	var index strings.Builder
	fmt.Fprintf(&index, "<h1>%s</h1><p>Offline reference. Source revision: <code>%s</code>.</p><ul>", name, html.EscapeString(revision))
	for _, p := range pages {
		if err := ctx.Err(); err != nil {
			return err
		}
		for file, content := range p.Assets {
			if err := os.WriteFile(filepath.Join(docs, file), content, 0644); err != nil {
				return err
			}
		}
		body := p.Body
		for online, local := range localLinks {
			body = strings.ReplaceAll(body, `href="`+html.EscapeString(online)+`"`, `href="`+local+`"`)
			body = strings.ReplaceAll(body, `href="`+html.EscapeString(online)+`#`, `href="`+local+`#`)
		}
		if err := write(filepath.Join(docs, p.File), document(p.Name, p.Online, body)); err != nil {
			return err
		}
		fmt.Fprintf(&index, `<li><a href="%s">%s</a></li>`, p.File, html.EscapeString(p.Name))
		for _, e := range p.Entries {
			if _, err := stmt.ExecContext(ctx, e.Name, e.Type, e.Path); err != nil {
				return err
			}
		}
	}
	index.WriteString(`</ul><p><a href="LICENSE.txt">Upstream license</a>. Unofficial docsets.</p>`)
	if err := write(filepath.Join(docs, "index.html"), document(name, "", index.String())); err != nil {
		return err
	}
	return tx.Commit()
}

func document(title, online, body string) string {
	link := ""
	policy := ""
	if strings.Contains(body, `id="rendered-examples"`) {
		policy = `<meta http-equiv="Content-Security-Policy" content="connect-src 'none'; frame-src 'none'; form-action 'none'">`
	}
	if online != "" {
		link = ` · <a href="` + html.EscapeString(online) + `">Online documentation</a>`
	}
	return `<!doctype html><html lang="en"><!-- Online page at ` + html.EscapeString(online) + ` --><head><meta charset="utf-8"><meta name="viewport" content="width=device-width">` + policy + `<title>` + html.EscapeString(title) + `</title><link rel="stylesheet" href="style.css"></head><body><nav><a href="index.html">Index</a>` + link + `</nav><main>` + body + `</main></body></html>`
}

const stylesheet = `:root{color-scheme:light dark}body{font:15px/1.6 -apple-system,BlinkMacSystemFont,sans-serif;margin:24px auto;padding:0 24px;max-width:1100px}a{color:#16846b}h1,h2,h3{line-height:1.25}h2{margin-top:2em;border-bottom:1px solid #8885;padding-bottom:.4em}pre{padding:16px;overflow:auto;background:#8881;border:1px solid #8884;border-radius:6px}code{font:13px/1.5 ui-monospace,monospace}table{border-collapse:collapse;display:block;overflow:auto}th,td{text-align:left;padding:8px 12px;border:1px solid #8884;min-width:80px}td{vertical-align:top}nav{font-size:13px;margin-bottom:24px}img{max-width:100%}@media(prefers-color-scheme:dark){a{color:#62cfb1}img[src$=".svg"]{background:#eee;border-radius:8px}}`
