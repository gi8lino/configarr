.DEFAULT_GOAL := help

# renovate: datasource=github-releases depName=gi8lino/dev-tools
DEV_TOOLS_VERSION ?= v0.9.0

include bin/dev-tools.mk
include $(call dev-tools-module,tag)
include $(call dev-tools-module,help)

GO ?= go
TINYGO ?= tinygo
PACKAGE := ./cmd/configarr
BINARY ?= bin/configarr
GO_BINARY ?= bin/configarr-go
COVERAGE_FILE ?= coverage.out
FUZZ_TIME ?= 10s

# renovate: datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.15.0
GOLANGCI_LINT := bin/golangci-lint

##@ Development

.PHONY: deps
deps: ## Download Go dependencies.
	$(call run-tool,$(GO),mod download)

.PHONY: fmt
fmt: ## Format Go source files.
	$(call run-tool,$(GO),fmt ./...)

.PHONY: tidy
tidy: ## Update go.mod and go.sum.
	$(call run-tool,$(GO),mod tidy)

.PHONY: vet
vet: ## Check Go code for common mistakes.
	$(call run-tool,$(GO),vet ./...)

.PHONY: test
test: ## Run all tests with race detection.
	$(call run-tool,$(GO),test -race ./...)

.PHONY: coverage
coverage: ## Run tests and write a coverage report.
	$(call run-tool,$(GO),test -race -covermode=atomic -coverprofile "$(COVERAGE_FILE)" ./...)
	$(call run-tool,$(GO),tool cover -func "$(COVERAGE_FILE)")

.PHONY: fuzz
fuzz: ## Fuzz XML editing; override FUZZ_TIME to change the duration.
	$(call run-tool,$(GO),test ./internal/xmlconfig -run "^$$" -fuzz "^FuzzDocumentUpdates$$" -fuzztime "$(FUZZ_TIME)")

.PHONY: golangci-lint
golangci-lint: $(GO_INSTALL_TOOL)
	$(call run-tool,$(GO_INSTALL_TOOL),\
		--target "$(GOLANGCI_LINT)" \
		--package github.com/golangci/golangci-lint/v2/cmd/golangci-lint \
		--tool-version "$(GOLANGCI_LINT_VERSION)")

.PHONY: lint
lint: golangci-lint ## Run the pinned Go linter; installs it when needed.
	$(call run-tool,$(GOLANGCI_LINT),run --timeout=3m ./...)

.PHONY: check
check: vet test lint ## Run vet, race tests, and lint.

##@ Build

.PHONY: build
build: ## Build bin/configarr with TinyGo, using the container's size options.
	@mkdir -p "$(dir $(BINARY))"
	$(call run-tool,$(TINYGO),build -o "$(BINARY)" -opt=s -no-debug $(PACKAGE))

.PHONY: build-go
build-go: ## Build bin/configarr-go with the standard Go compiler.
	@mkdir -p "$(dir $(GO_BINARY))"
	$(call run-tool,$(GO),build -o "$(GO_BINARY)" $(PACKAGE))

.PHONY: run
run: ## Run with Go; pass command arguments through ARGS.
	$(call run-tool,$(GO),run $(PACKAGE) $(ARGS))

.PHONY: clean
clean: ## Remove application binaries and coverage output; keep cached tools.
	rm -f "$(BINARY)" "$(GO_BINARY)" "$(COVERAGE_FILE)"
