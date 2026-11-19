# Makefile for the OSTERA publications repository.
#
# Run these from the repository root. "make" alone lists the targets.
# Every target runs pubcheck (tools/pubcheck); see tools/pubcheck/README.md.

PUBCHECK := go run -C tools/pubcheck .

# The remote that holds the published repository on GitHub, and the branch
# that validate compares against, so it can check that nothing published or
# announced was changed. Set BASE= (empty) to skip that check.
REMOTE ?= origin
BASE ?= $(REMOTE)/main

.DEFAULT_GOAL := help
.PHONY: help generate checksums validate test

help: ## List the targets
	@echo "usage: make <target>"
	@grep -E '^[a-z]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN { FS = ":.*## " } { printf "  make %-10s %s\n", $$1, $$2 }'

generate: ## Write the generated pages, catalog.json, and generated sections
	@$(PUBCHECK) generate

checksums: ## Write SHA512SUMS in DIR, for example: make checksums DIR=docs/jcs/v1.0
	@test -n "$(DIR)" || { echo "usage: make checksums DIR=<folder>, for example DIR=docs/jcs/v1.0"; exit 2; }
	@$(PUBCHECK) checksums $(DIR)

validate: ## Check the repository, comparing against BASE (default origin/main)
ifneq ($(BASE),)
	@git fetch --quiet $(REMOTE)
	@$(PUBCHECK) validate -base $(BASE)
else
	@$(PUBCHECK) validate
endif

test: ## Run the pubcheck tests
	@cd tools/pubcheck && go test ./...
