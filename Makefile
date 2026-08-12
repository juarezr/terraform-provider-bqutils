#! make -f

#region Constants and Variables --------------------------------------------------------

REGISTRY=registry.terraform.io
NAMESPACE=juarezr
NAME=bqutils
VERSION=0.7.0
BINARY=terraform-provider-${NAME}

REQUIRED_TOOLS=awk sed grep sort tail tr

OS_ARCH ?= $(shell go env GOOS)_$(shell go env GOARCH)
USER_HOME_DIR=$(shell realpath ~)
TERRAFORM_PLUGINS_DIR=$(shell realpath ~/.terraform.d/plugins)
PROVIDER_DIR=${TERRAFORM_PLUGINS_DIR}/${REGISTRY}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

#endregion -----------------------------------------------------------------------------

#region Defaults and Help --------------------------------------------------------------

default: all

.PHONY: help
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: help
query:
	@make -qp | awk -F':' '/^[a-zA-Z0-9][^$$#\t=]*:([^=]|$$)/ {split($$1,A,/ /);for(i in A)print A[i]}' | sort -u

.PHONY: requirements
requirements: ## Check the required tools are installed
	@echo "Starting batch processing..."
	@$(foreach item,$(REQUIRED_TOOLS), echo -n "$(item): "; command -v $(item) || echo "Missing";)

#endregion -----------------------------------------------------------------------------

#region Build and Test -----------------------------------------------------------------

SQLPARSE_SRCS := $(wildcard internal/sqlparse/*.go)
PROVIDER_SRCS := $(wildcard internal/provider/*.go) main.go
PACKAGES_SRCS := go.mod go.sum
BUILDING_SRCS := $(SQLPARSE_SRCS) $(PROVIDER_SRCS) $(PACKAGES_SRCS)

${BINARY}: $(BUILDING_SRCS)
	go build -o ${BINARY}

.PHONY: build
build: ${BINARY} ## Compile the project source code

.PHONY: test
test: ## Run the unit tests
	go test ./... -v -count=1

.PHONY: testacc
testacc: ## Run the acceptance tests
	TF_ACC=1 go test ./... -v -count=1 -timeout 120m

.PHONY: clean-build
clean-build: ## Clean the build artifacts
	rm -f ${BINARY}

#endregion -----------------------------------------------------------------------------

#region Documentation ------------------------------------------------------------------

MARKDOWN_OUTS := $(wildcard docs/*.md docs/**/*.md)
TEMPLATE_SRCS := $(wildcard templates/*.md.tmpl templates/**/*.md.tmpl)
EXAMPLES_SRCS := $(wildcard examples/**/*.tf examples/**/*.sql)
TFSCHEMA_SRCS := $(filter-out %_test.go,${PROVIDER_SRCS})

$(MARKDOWN_OUTS): $(TEMPLATE_SRCS) $(EXAMPLES_SRCS) $(TFSCHEMA_SRCS) $(PACKAGES_SRCS)
	go generate ./...
# 	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name bqutils

.PHONY: generate
generate: $(MARKDOWN_OUTS) ## Build the project documentation

.PHONY: validate
validate: generate ## Validate the project documentation
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs validate --provider-name bqutils

#endregion -----------------------------------------------------------------------------

#region Coverage -----------------------------------------------------------------------

coverage-sqlparse.cover: $(SQLPARSE_SRCS)
	go test ./internal/sqlparse/ -count=1 -coverprofile=coverage-sqlparse.cover -covermode=atomic

coverage-provider.cover: $(PROVIDER_SRCS) $(PACKAGES_SRCS)
	TF_ACC=1 go test ./internal/provider/ -count=1 -timeout 20m -coverprofile=coverage-provider.cover -covermode=atomic

coverage.out: coverage-sqlparse.cover coverage-provider.cover
	gocovmerge coverage-sqlparse.cover coverage-provider.cover > coverage.out

coverage.xml: coverage.out
	gocover-cobertura < coverage.out > coverage.xml

coverage.log: coverage.out
	go tool cover -func=coverage.out -o coverage.log

define COVERAGE_REPORT_TEXT
| File | Line | Function | Coverage |
|------|------|----------|----------|
endef
export COVERAGE_REPORT_TEXT

coverage.md: coverage.log
	echo "$${COVERAGE_REPORT_TEXT}" > coverage.md
	cat coverage.log | sed -E -e 's/^total/total:average/' -e 's/[:\t]+/ | /g' -e 's/.*/| & |/' >> coverage.md
	tail -n 1 coverage.log | tr -d '\t'

.PHONY: clean-coverage
clean-coverage: ## Clean the coverage artifacts
	rm -f *.cover coverage-*.* coverage.*

.PHONY: coverage
coverage: coverage.xml ## Generate the coverage from the source code

.PHONY: report
report: coverage.md ## Generate the coverage report

#endregion -----------------------------------------------------------------------------

#region Format and Lint ----------------------------------------------------------------

.PHONY: clean-trash
clean-trash:
	rm -f *.log
	rm -f *.tmp

.PHONY: clean
clean: clean-build clean-coverage clean-trash ## Clean the project

.PHONY: fmt
fmt: ## Format the project source code
	gofmt -w .

.PHONY: lint
lint: ## Lint the project source code ensuring it passes the CI checks
	gofmt -l .

.PHONY: check
check: build test testacc lint validate ## Run all the project sanity checks

.PHONY: all
all: build generate report ## Run all the project build steps

#endregion -----------------------------------------------------------------------------

#region Local Test Installation --------------------------------------------------------

.PHONY: install
install: build ## Install the provider in the local Terraform plugins directory
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
dev-override: ## Setup for using the provider in the local Terraform modules
	echo "$${TERRAFORMRC_TEXT}" > ~/.terraformrc

.PHONY: uninstall
uninstall: ## Uninstall the provider from Terraform plugins directory
	rm -rfv ${PROVIDER_DIR}
	rm -fv ~/.terraformrc
	rm -fv ${BINARY}

#endregion -----------------------------------------------------------------------------

#region Dependencies -------------------------------------------------------------------

.PHONY: tools
tools: ## Install the Golang tools used by the project
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/wadey/gocovmerge@latest
	go install github.com/boumenot/gocover-cobertura@latest

.PHONY: verify
verify: ## Verify the project dependencies for vulnerabilities
	govulncheck ./...

.PHONY: outdated
outdated: ## List the outdated dependencies in the project
	go list -m -u

.PHONY: upgrade
upgrade: ## Upgrade the dependencies in the project
	go get -u
	go mod tidy
	go mod verify

#endregion -----------------------------------------------------------------------------
