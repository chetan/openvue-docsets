OPENVUE_REV := 31626a6b0c7bc89421caa4869da46c3995727688
OPENICONS_REV := a5da14ac692c1903bdcc0164288cc8a5e7b3ae8c
OUTPUT ?= dist

.PHONY: setup fetch format test check build examples browser-setup test-browser docsets package
setup:
	go mod download
	bun install --frozen-lockfile

fetch:
	sh scripts/fetch.sh openvue $(OPENVUE_REV)
	sh scripts/fetch.sh openicons $(OPENICONS_REV)

format:
	gofmt -w cmd internal
	bun run format

test:
	go test ./...
	bun run test:renderer

check: test
	go vet ./...
	test -z "$$(gofmt -l cmd internal)"
	bun run format:check

build:
	mkdir -p bin
	go build -trimpath -o bin/openvue-docsets ./cmd/openvue-docsets

examples: fetch
	bun run examples

browser-setup:
	python3 -m venv .cache/test-venv
	.cache/test-venv/bin/pip install playwright==1.65.0
	.cache/test-venv/bin/playwright install chromium webkit

test-browser:
	.cache/test-venv/bin/python renderer/smoke.py --docset "$(OUTPUT)/OpenVue.docset" --all

docsets: examples build
	bin/openvue-docsets build --output "$(OUTPUT)"

package:
	COPYFILE_DISABLE=1 tar --exclude=.DS_Store -czf "$(OUTPUT)/OpenVue.tgz" -C "$(OUTPUT)" OpenVue.docset
	COPYFILE_DISABLE=1 tar --exclude=.DS_Store -czf "$(OUTPUT)/OpenIcons.tgz" -C "$(OUTPUT)" OpenIcons.docset
