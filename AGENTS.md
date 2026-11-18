# OpenVue docsets

## Purpose and stack

This standalone Go CLI builds separate OpenVue and OpenIcons Dash docsets.
Use Cobra for commands, Goldmark for Markdown, and modernc SQLite for indexes.
Do not add a web server, frontend runtime, or Tradecraft app dependencies.

## Layout

- `cmd/openvue-docsets/`: CLI, version, flags, process exit.
- `internal/docset/`: Markdown conversion, icon pages, bundles, search indexes.
- `scripts/fetch.sh`: pinned, read-only-after-clone upstream source setup.
- `Makefile`: source revisions and root commands.
- `.cache/`, `bin/`, `dist/`: ignored source cache and generated output.

## Commands

Run from the repository root:

- Setup: `make setup`
- Fetch sources: `make fetch`
- Format: `make format`
- Tests and checks: `make check`
- Build CLI: `make build`
- Generate docsets: `make docsets`
- Package bundles: `make package`
- New output directory: `make docsets OUTPUT=dist/update-1`

Tests use local fixtures and need no network. Fetch needs GitHub access.
No local services, credentials, environment prefix, or configuration files
are required. Flags select input checkouts and output paths.

## Ownership and behavior

Upstream documentation and icons are external inputs, not project-owned code.
Consult current upstream docs before changing generated API examples.
Preserve upstream license files and revision provenance in both bundles.
Do not commit downloaded sources, generated bundles, local config, or secrets.
Never remove existing output or reset source checkouts automatically.

Keep `README.md` current for command, source, output, and behavior changes.
Add tests for indexing, rendering, bundle structure, and error handling.
Validate that indexed files and fragment IDs exist. Keep help and version
independent of sources, network, and runtime services.
