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
# comparison. FORCE=1 shows the history problems as warnings instead of
# errors, for a deliberate exception; it is refused in GitHub Actions, so
# such a change is merged by someone on the bypass list of the main branch
# ruleset.
REPO := github.com/OSTERA-org/publications
REMOTE ?= $(shell git remote -v 2>/dev/null | awk -v repo="$(REPO)" '$$3 == "(fetch)" { u = tolower($$2); sub(/\.git$$/, "", u); sub(/github\.com:/, "github.com/", u); if (u ~ ("[/:@]" tolower(repo) "$$") || u ~ ("^" tolower(repo) "$$")) { print $$1; exit } }')
BASE ?= $(REMOTE)/main

.DEFAULT_GOAL := help
.PHONY: help update validate generate checksums publish add-wg reserve assign test

help: ## List the targets
	@echo "usage: make <target>"
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN { FS = ":.*## " } { printf "  make %-10s %s\n", $$1, $$2 }'

update: ## Write any missing or outdated SHA512SUMS, then generate (use after every change)
	@$(PUBCHECK) update

generate: ## Write the generated pages, catalog.json, and generated sections
	@$(PUBCHECK) generate

checksums: ## Write SHA512SUMS in DIR, for example: make checksums DIR=docs/jcs/v1.0
	@test -n "$(DIR)" || { echo "usage: make checksums DIR=<folder>, for example DIR=docs/jcs/v1.0"; exit 2; }
	@$(PUBCHECK) checksums $(DIR)

validate: ## Check the repository, comparing against the main branch on GitHub (FORCE=1 for a deliberate exception)
ifneq ($(BASE),)
	@test -n "$(REMOTE)" || { echo "No git remote points to https://$(REPO)."; echo "Add one with: git remote add origin https://$(REPO).git"; exit 2; }
	@echo "Comparing against $(BASE)"
	@git fetch --quiet $(REMOTE)
	@$(PUBCHECK) validate -base $(BASE) $(if $(FORCE),-force)
else
	@test -z "$(FORCE)" || { echo "FORCE=1 only applies to the comparison with the main branch, so it cannot be used with BASE="; exit 2; }
	@$(PUBCHECK) validate
endif

publish: ## Publish an approved draft from its Working Group site (ID=jcs is optional)
	@$(PUBCHECK) publish $(or $(ID),$(URL))

add-wg: ## Add a Working Group to registry.yaml (ID=jss is optional)
	@$(PUBCHECK) add-wg $(ID)

reserve: ## Reserve the next document number for a work item (ID=jcs is optional)
	@$(PUBCHECK) reserve $(or $(ID),$(URL))

assign: ## Assign the next document number to a work item, or assign its reserved number (ID=jcs is optional)
	@$(PUBCHECK) assign $(or $(ID),$(URL))

test: ## Run the pubcheck tests
	@cd tools/pubcheck && go test ./...
