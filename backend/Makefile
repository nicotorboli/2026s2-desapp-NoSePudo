-include app.env
export

SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c

MODULE := $(shell go list -m)
TESTED := $(shell go list -test -f '{{if not .ForTest}}{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}{{end}}' ./...)
INTEGRATION := $(shell go list -test -f '{{if not .ForTest}}{{.ImportPath}}|{{join .TestImports " "}}|{{join .XTestImports " "}}{{end}}' ./... | awk -F'|' '$$2 ~ /\/testsupport|testcontainers/ || $$3 ~ /\/testsupport|testcontainers/ {print $$1}')
UNIT := $(filter-out $(INTEGRATION),$(TESTED))

RACE ?= 1
COUNT ?= 1
SLOWEST ?= 5
COVER_MIN ?= 0
COVERDIR := coverage
REPORT := scripts/test-report.sh
COVER_REPORT := scripts/coverage-report.sh

ifeq ($(RACE),1)
RACE_FLAG := -race
else
RACE_FLAG :=
endif

COUNT_FLAG := $(if $(COUNT),-count=$(COUNT),)
COVER_FLAGS := -covermode=atomic -coverpkg=./...

.PHONY: help run require-jq require-docker test test-integration test-e2e test-all test-packages coverage coverage-unit coverage-ci clean-coverage

help:
	@printf '%s\n' \
	  'NoSePudo backend' \
	  '' \
	  '  make run                start the server' \
	  '  make test               unit tests only, no Docker' \
	  '  make test-integration   tests that start Postgres with testcontainers (Docker required)' \
	  '  make test-e2e           end-to-end tests only (Docker required)' \
	  '  make test-all           unit and integration tests (Docker required)' \
	  '  make test-packages      show how packages are classified' \
	  '  make coverage-unit      unit coverage, merged table and HTML' \
	  '  make coverage           unit and integration coverage, merged table and HTML' \
	  '  make coverage-ci        exact CI command, writes coverage.out' \
	  '  make clean-coverage     remove generated coverage artifacts' \
	  '' \
	  'Variables: RACE=1 COUNT=1 SLOWEST=5 COVER_MIN=0' \
	  'Requires: go and jq; Docker for integration targets'

run:
	go run cmd/server/main.go

require-jq:
	@command -v jq >/dev/null 2>&1 || { \
	  echo '[ERROR] jq is required to format test output.' >&2; \
	  echo '        Install it with your package manager (for example: apt install jq).' >&2; \
	  exit 2; \
	}

require-docker:
	@docker info >/dev/null 2>&1 || { \
	  echo '[ERROR] Docker daemon is not reachable.' >&2; \
	  echo '        This target runs tests that start Postgres with testcontainers.' >&2; \
	  exit 2; \
	}

test: require-jq
	@test -n "$(UNIT)" || { echo '[ERROR] no unit packages detected' >&2; exit 2; }
	@go test -json $(RACE_FLAG) $(COUNT_FLAG) $(UNIT) | LABEL=unit MODULE="$(MODULE)" RACE="$(RACE)" SLOWEST="$(SLOWEST)" $(REPORT)

test-integration: require-jq require-docker
	@test -n "$(INTEGRATION)" || { echo '[ERROR] no integration packages detected' >&2; exit 2; }
	@go test -json $(RACE_FLAG) $(COUNT_FLAG) $(INTEGRATION) | LABEL=integration MODULE="$(MODULE)" RACE="$(RACE)" SLOWEST="$(SLOWEST)" $(REPORT)

test-e2e: require-jq require-docker
	@go test -json $(RACE_FLAG) $(COUNT_FLAG) ./test/e2e | LABEL=e2e MODULE="$(MODULE)" RACE="$(RACE)" SLOWEST="$(SLOWEST)" $(REPORT)

test-all: require-jq require-docker
	@test -n "$(TESTED)" || { echo '[ERROR] no test packages detected' >&2; exit 2; }
	@go test -json $(RACE_FLAG) $(COUNT_FLAG) $(TESTED) | LABEL=all MODULE="$(MODULE)" RACE="$(RACE)" SLOWEST="$(SLOWEST)" $(REPORT)

test-packages:
	@printf 'UNIT (%s packages, no Docker)\n' "$(words $(UNIT))"
	@for pkg in $(UNIT); do printf '  %s\n' "$${pkg#$(MODULE)/}"; done
	@printf 'INTEGRATION (%s packages, Docker required)\n' "$(words $(INTEGRATION))"
	@for pkg in $(INTEGRATION); do printf '  %s\n' "$${pkg#$(MODULE)/}"; done

coverage-unit: require-jq
	@test -n "$(UNIT)" || { echo '[ERROR] no unit packages detected' >&2; exit 2; }
	@mkdir -p $(COVERDIR)
	@go test -json $(RACE_FLAG) $(COUNT_FLAG) $(COVER_FLAGS) -coverprofile=$(COVERDIR)/unit.out $(UNIT) | LABEL=unit MODULE="$(MODULE)" RACE="$(RACE)" SLOWEST="$(SLOWEST)" $(REPORT)
	@MODULE="$(MODULE)" $(COVER_REPORT) --out $(COVERDIR)/merged.out --html $(COVERDIR)/index.html --min $(COVER_MIN) $(COVERDIR)/unit.out

coverage: require-jq require-docker
	@test -n "$(UNIT)" || { echo '[ERROR] no unit packages detected' >&2; exit 2; }
	@test -n "$(INTEGRATION)" || { echo '[ERROR] no integration packages detected' >&2; exit 2; }
	@mkdir -p $(COVERDIR)
	@go test -json $(RACE_FLAG) $(COUNT_FLAG) $(COVER_FLAGS) -coverprofile=$(COVERDIR)/unit.out $(UNIT) | LABEL=unit MODULE="$(MODULE)" RACE="$(RACE)" SLOWEST="$(SLOWEST)" $(REPORT)
	@go test -json $(RACE_FLAG) $(COUNT_FLAG) $(COVER_FLAGS) -coverprofile=$(COVERDIR)/integration.out $(INTEGRATION) | LABEL=integration MODULE="$(MODULE)" RACE="$(RACE)" SLOWEST="$(SLOWEST)" $(REPORT)
	@MODULE="$(MODULE)" $(COVER_REPORT) --out $(COVERDIR)/merged.out --html $(COVERDIR)/index.html --min $(COVER_MIN) $(COVERDIR)/unit.out $(COVERDIR)/integration.out

coverage-ci: require-docker
	go test -v -race -covermode=atomic -coverpkg=./... -coverprofile=coverage.out ./...
	@MODULE="$(MODULE)" $(COVER_REPORT) --min $(COVER_MIN) coverage.out

clean-coverage:
	@rm -rf $(COVERDIR) coverage.out
