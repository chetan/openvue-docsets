OPENVUE_REV := 31626a6b0c7bc89421caa4869da46c3995727688
OPENICONS_REV := a5da14ac692c1903bdcc0164288cc8a5e7b3ae8c
OUTPUT ?= dist

.PHONY: setup fetch format test check build docsets package
setup:
	go mod download

fetch:
	sh scripts/fetch.sh openvue $(OPENVUE_REV)
	sh scripts/fetch.sh openicons $(OPENICONS_REV)

format:
	gofmt -w cmd internal

test:
	go test ./...

check: test
	go vet ./...
	test -z "$$(gofmt -l cmd internal)"

build:
	mkdir -p bin
	go build -trimpath -o bin/openvue-docsets ./cmd/openvue-docsets

docsets: fetch build
	bin/openvue-docsets build --output "$(OUTPUT)"

package:
	COPYFILE_DISABLE=1 tar --exclude=.DS_Store -czf "$(OUTPUT)/OpenVue.tgz" -C "$(OUTPUT)" OpenVue.docset
	COPYFILE_DISABLE=1 tar --exclude=.DS_Store -czf "$(OUTPUT)/OpenIcons.tgz" -C "$(OUTPUT)" OpenIcons.docset
