NAME=libvirt
BINARY=packer-plugin-${NAME}
PLUGIN_FQN="$(shell grep -E '^module' <go.mod | sed -E 's/module *//')"

COUNT?=1
TEST?=$(shell go list ./...)

.PHONY: dev

build:
	@go build -o ${BINARY}

dev:
	@go build -ldflags="-X '${PLUGIN_FQN}/version.VersionPrerelease=dev'" -o '${BINARY}'
	packer plugins install --path ${BINARY} "$(shell echo "${PLUGIN_FQN}" | sed 's/packer-plugin-//')"

test:
	@go test -race -count $(COUNT) $(TEST) -timeout=3m

ci-release-docs:
	@go run github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc renderdocs -src docs -partials docs-partials/ -dst docs/
	@/bin/sh -c "[ -d docs ] && zip -r docs.zip docs/"

plugin-check: build
	@go run github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc plugin-check ${BINARY}

testacc: dev
	@PACKER_ACC=1 go test -count $(COUNT) -v $(TEST) -timeout=120m

generate:
	@go generate ./...
	go run github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc renderdocs -src ./docs -dst ./.docs -partials ./docs-partials
	# checkout the .docs folder for a preview of the docs
