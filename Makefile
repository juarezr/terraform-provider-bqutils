HOSTNAME=registry.terraform.io
NAMESPACE=juarezr
NAME=bqutils
BINARY=terraform-provider-${NAME}
VERSION=0.1.0
OS_ARCH ?= $(shell go env GOOS)_$(shell go env GOARCH)

default: build

.PHONY: build
build:
	go build -o ${BINARY}

.PHONY: install
install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

.PHONY: test
test:
	go test ./... -v -count=1

.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v -count=1 -timeout 120m

.PHONY: generate
generate:
	cd internal/sqlparse && goyacc -o y.go -p yy parser.y
	go generate ./...
	gofmt -w ./internal/sqlparse

.PHONY: fmt
fmt:
	gofmt -w .

.PHONY: lint
lint:
	gofmt -l .

.PHONY: check
check: build test testacc docs

.PHONY: tools
tools:
	go install github.com/vitessio/goyacc@latest
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

.PHONY: verify
verify:
	govulncheck ./...

.PHONY: docs
docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name bqutils
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs validate --provider-name bqutils

.PHONY: clean
clean:
	rm -f ${BINARY}
	rm -f y.go
	rm -f *.log
	rm -f *.tmp

.PHONY: outdated
outdated:
	go list -m -u all

.PHONY: upgrade
update:
	go get -u all
	go mod tidy
	go mod verify
