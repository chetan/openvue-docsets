# OpenVue Dash docsets

Build two separate, offline docsets for [Dash](https://kapeli.com/dash):

- **OpenVue:** component documentation, examples, API tables, accessibility,
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

Requirements: Go 1.26.4 or later, Git, Make, and internet access for setup.
No Node.js, web server, credentials, or installed Vue packages are required.

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
  --output dist/custom
```

## Scope and limits

- Documentation and code samples are static. Live Vue demos do not run.
- The builder uses OpenVue's generated Markdown, not a mirror of its website.
  It can only include the content present in that source.
- Raw HTML in upstream Markdown is disabled. Example code remains visible.
- Links between included pages are local when their URLs match. Other links
  open online. Images referenced by upstream guides are not mirrored.
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
```

The CLI uses Cobra, Markdown rendering uses Goldmark, and the Dash index uses
pure-Go SQLite. Dependencies are locked in `go.mod` and `go.sum`. Tradecraft's
managed app runtime is not used because this is a standalone file generator.
