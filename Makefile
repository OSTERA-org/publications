# Makefile for the OSTERA publications repository.
#
# Run these from the repository root. "make" alone lists the targets.
# Every target runs pubcheck (tools/pubcheck). README-Makefile.md explains
# what each target does, and when and why to use it.

PUBCHECK := go run -C tools/pubcheck .

# REPO is the published repository on GitHub. REMOTE is the name your git
# copy uses for it, found by looking for REPO in "git remote -v", whatever
# the remote is called. BASE is the branch that validate compares against,
# so it can check that nothing published or announced was changed. Set
# REMOTE=<name> to choose the remote yourself, or BASE= (empty) to skip the
# comparison.
REPO := github.com/OSTERA-org/publications
REMOTE ?= $(shell git remote -v 2>/dev/null | awk -v repo="$(REPO)" '$$3 == "(fetch)" { u = tolower($$2); sub(/\.git$$/, "", u); sub(/github\.com:/, "github.com/", u); if (u ~ ("[/:@]" tolower(repo) "$$") || u ~ ("^" tolower(repo) "$$")) { print $$1; exit } }')
BASE ?= $(REMOTE)/main

.DEFAULT_GOAL := help
.PHONY: help update validate generate checksums test

help: ## List the targets
	@echo "usage: make <target>"
	@grep -E '^[a-z]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN { FS = ":.*## " } { printf "  make %-10s %s\n", $$1, $$2 }'

update: ## Write any missing or outdated SHA512SUMS, then generate (use after every change)
	@$(PUBCHECK) update

generate: ## Write the generated pages, catalog.json, and generated sections
	@$(PUBCHECK) generate

checksums: ## Write SHA512SUMS in DIR, for example: make checksums DIR=docs/jcs/v1.0
	@test -n "$(DIR)" || { echo "usage: make checksums DIR=<folder>, for example DIR=docs/jcs/v1.0"; exit 2; }
	@$(PUBCHECK) checksums $(DIR)

validate: ## Check the repository, comparing against the main branch on GitHub
ifneq ($(BASE),)
	@test -n "$(REMOTE)" || { echo "No git remote points to https://$(REPO)."; echo "Add one with: git remote add origin https://$(REPO).git"; exit 2; }
	@echo "Comparing against $(BASE)"
	@git fetch --quiet $(REMOTE)
	@$(PUBCHECK) validate -base $(BASE)
else
	@$(PUBCHECK) validate
endif

test: ## Run the pubcheck tests
	@cd tools/pubcheck && go test ./...
