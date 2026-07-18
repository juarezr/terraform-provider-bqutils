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

.PHONY: fmt
fmt:
	gofmt -w .

.PHONY: tools
tools:
	go install github.com/vitessio/goyacc@latest
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

.PHONY: docs
docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name bqutils

.PHONY: clean
clean:
	rm -f ${BINARY}
	rm -f y.go
	rm -f *.log
	rm -f *.tmp
