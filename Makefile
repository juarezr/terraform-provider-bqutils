REGISTRY=registry.terraform.io
NAMESPACE=juarezr
NAME=bqutils
VERSION=0.1.0
BINARY=terraform-provider-${NAME}
OS_ARCH ?= $(shell go env GOOS)_$(shell go env GOARCH)
PLUGINS_DIR=$(shell realpath ~/.terraform.d/plugins)
PROVIDER_DIR=${PLUGINS_DIR}/${REGISTRY}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

default: build

.PHONY: build
build:
	go build -o ${BINARY}

.PHONY: test
test:
	go test ./... -v -count=1

.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v -count=1 -timeout 120m

.PHONY: fmt
fmt:
	gofmt -w .

.PHONY: lint
lint:
	gofmt -l .

.PHONY: check
check: build test testacc
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs validate --provider-name bqutils

.PHONY: install
install: build
	mkdir -p ${PROVIDER_DIR}
	mv ${BINARY} ${PROVIDER_DIR}

define TERRAFORMRC_TEXT
provider_installation {
	dev_overrides {
		"juarezr/bqutils" = "${PROVIDER_DIR}"
	}
	direct {}
}
endef

export TERRAFORMRC_TEXT

.PHONY: dev-override
dev-override:
	echo "$${TERRAFORMRC_TEXT}" > ~/.terraformrc

.PHONY: uninstall
uninstall:
	rm -rfv ${PROVIDER_DIR}
	rm -fv ~/.terraformrc
	rm -fv ${BINARY}

.PHONY: clean
clean:
	rm -f ${BINARY}
	rm -f *.log
	rm -f *.tmp

.PHONY: tools
tools:
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

.PHONY: generate
generate:
	go generate ./...

.PHONY: docs
docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name bqutils

.PHONY: verify
verify:
	govulncheck ./...

.PHONY: outdated
outdated:
	go list -m -u all

.PHONY: upgrade
update:
	go get -u all
	go mod tidy
	go mod verify
