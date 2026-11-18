# OpenVue docsets

## Purpose and stack

This standalone Go CLI builds separate OpenVue and OpenIcons Dash docsets.
Use Cobra for commands, Goldmark for Markdown, and modernc SQLite for indexes.
The offline examples use Vue and OpenVue bundled as a classic IIFE with Vite.
Do not add a web server, Nuxt runtime, or Tradecraft app dependencies.

## Layout

- `cmd/openvue-docsets/`: CLI, version, flags, process exit.
- `internal/docset/`: Markdown conversion, icon pages, bundles, search indexes.
- `internal/docset/navigation.{go,css,js}`: native TOC markers and persistent
  component browser. The CSS and JavaScript are embedded in the Go binary.
- `scripts/fetch.sh`: pinned, read-only-after-clone upstream source setup.
- `renderer/`: build-time extraction of original website demos, offline assets,
  Vue docset host, renderer tests, and file-URL browser tests.
- `package.json`, `bun.lock`: pinned renderer runtime and build dependencies.
- `Makefile`: source revisions and root commands.
- `.cache/`, `bin/`, `dist/`: ignored source cache and generated output.

## Commands

Run from the repository root:

- Setup: `make setup`
- Fetch sources: `make fetch`
- Format: `make format`
- Tests and checks: `make check`
- Build CLI: `make build`
- Bundle interactive examples: `make examples`
- Generate docsets: `make docsets`
- Package bundles: `make package`
- New output directory: `make docsets OUTPUT=dist/update-1`
- Install browser test tools: `make browser-setup`
- Offline Chromium/WebKit tests: `make test-browser OUTPUT=dist/update-1`

Unit tests use local fixtures and need no network. Fetch needs GitHub access.
Example builds download and cache demo media in `.cache/media/`. Later builds
can use the cached media offline. Browser tests use file URLs and disable
network access. They need Python and the tools from `make browser-setup`.
No local services, credentials, environment prefix, or configuration files
are required. Flags select input checkouts and output paths.

## Ownership and behavior

Upstream documentation and icons are external inputs, not project-owned code.
Consult current upstream docs before changing generated API examples.
Compile the original feature SFCs. Do not execute Markdown snippets or replace
real examples with fabricated placeholders. Keep site-specific adapters small.
Do not use ESM imports, dynamic chunks or runtime fetches: Dash opens file URLs.
Preserve example load events and component services. Do not hide demo failures.
Keep Dash TOC markers and section wrappers outside Vue's mount root. Use
Teleport into static demo slots so anchor identities survive rendering.
Maintain the renderer format version when changing the host/slot contract;
the builder must reject stale compiled assets. Test marker preservation,
component navigation, active-page state, and narrow-window disclosure.
Set Dash's JavaScript flag only for OpenVue, not for the static OpenIcons set.
Preserve upstream license files and revision provenance in both bundles.
Do not commit downloaded sources, generated bundles, local config, or secrets.
Never remove existing output or reset source checkouts automatically.

Keep `README.md` current for command, source, output, and behavior changes.
Add tests for indexing, rendering, bundle structure, and error handling.
Validate that indexed files and fragment IDs exist. Keep help and version
independent of sources, network, and runtime services.
