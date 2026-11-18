# OpenVue Dash docsets

Build two separate, offline docsets for [Dash](https://kapeli.com/dash):

- **OpenVue:** rendered, interactive examples plus component documentation,
  API tables, accessibility,
  pass-through options, design tokens, and setup guides.
- **OpenIcons:** one page per icon with an SVG preview, webfont class,
  Vue import, codepoint, and keywords. Includes the upstream icon guides.

Search for `Button`, `Button.label`, `DataTable`, `oi-check`, or `OiCheck`.
API member results open the relevant table. Section results open their heading.

These are unofficial docsets. Content and SVGs come from the MIT-licensed
[OpenVue](https://github.com/openvi-foundation/openvue) and
[OpenIcons](https://github.com/openvi-foundation/openicons) projects.
Each docset contains its upstream license and source revision.

## Build

Requirements: Go 1.26.4 or later, Node.js 22.12 or later, Bun, Git, Make, and
internet access for setup and the first media download.
No web server, credentials, or globally installed Vue packages are required.

```sh
make setup
make check
make docsets
make package
```

Output:

```text
dist/OpenVue.docset
dist/OpenIcons.docset
dist/OpenVue.tgz
dist/OpenIcons.tgz
```

Open both `.docset` bundles in Dash:

```sh
open -a Dash dist/OpenVue.docset dist/OpenIcons.docset
```

Alternatively, use **Dash → Settings → Docsets → +** and select each bundle.
Use the `openvue` and `openicons` docset keywords to narrow searches.
Dash may copy the bundles into its own library.

## Rebuild and update

The Makefile pins upstream Git revisions. `make fetch` downloads those sources
to `.cache/`. Later builds work offline. It does not modify existing checkouts.
To update, change the revisions in `Makefile`, move the old cache checkouts
aside, and run `make fetch`.

The builder refuses to overwrite existing bundles. Select a new output
directory for each rebuild:

```sh
make docsets OUTPUT=dist/update-1
make package OUTPUT=dist/update-1
```

You can also build from other local upstream checkouts:

```sh
make build
bin/openvue-docsets build \
  --openvue /path/to/openvue \
  --icons /path/to/openicons \
  --output dist/custom \
  --examples .cache/examples
```

The examples and documentation must use the same upstream Git revision.
Set `OPENVUE_SOURCE=/path/to/openvue bun run examples` before building from a
custom checkout. Use `--examples ""` for a reference-only build.

## Interactive examples

The builder compiles the original Vue single-file components from
`apps/showcase/doc/`, using the feature lists from `apps/showcase/pages/`.
It does not execute or approximate the examples in the generated Markdown.
OpenVue and Vue are included locally. The theme, styles, icon font, fixture
data, and demo media are included in the docset.

Each supported page starts with the website's interactive feature examples.
Expand **Example source** to see the upstream code. The searchable Markdown
reference and API tables remain below the examples.

Examples use a classic JavaScript bundle, not ESM modules or a local server.
The OpenVue docset sets Dash's `isJavaScriptEnabled` flag. If Dash has an older
copy registered, remove **OpenVue** from **Settings → Docsets**, then open the
new bundle to reload its metadata. Do not remove OpenIcons.

Media provenance is in `media-manifest.json`. Bundled package licenses are in
`THIRD_PARTY_LICENSES.txt`. The example manifest also records the source revision.

## Scope and limits

- The docset includes the website's feature demos, not its Nuxt navigation,
  theme builder, StackBlitz integration, or separate playground tab.
- Examples that upload files or contact a backend cannot complete those actions
  offline. The docset blocks network connections, embedded frames and form
  submissions. Local input, selection, overlays and fixture-data demos run.
- The searchable reference uses upstream generated Markdown. Raw HTML in that
  reference is disabled; actual Vue demos are compiled separately.
- Links between included pages are local when their URLs match. Other links
  open online. The renderer bundles the media used by its website demos.
- Icon previews are local SVG files. No font download is required.
- Source revisions are recorded in `Contents/Resources/Documents/revision.txt`.
  Custom source checkouts should be clean; the builder records their HEAD.
- The project does not publish a feed, schedule updates, or install into Dash
  automatically.

## Development

```sh
make format
make check
make build
bin/openvue-docsets --help
bin/openvue-docsets version
make browser-setup
make test-browser OUTPUT=dist
```

The CLI uses Cobra, Markdown rendering uses Goldmark, and the Dash index uses
pure-Go SQLite. Dependencies are locked in `go.mod`, `go.sum`, and `bun.lock`.
The browser tests exercise popup date selection, buttons, checkboxes, dialogs,
and data tables, then load every interactive page with networking disabled.
They run in Chromium and WebKit, directly from `file://` URLs. Tradecraft's
managed app runtime is not used because this is a standalone file generator.
