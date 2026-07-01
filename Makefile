# All source code and documents, used when checking for misspellings
ALLDOC := $(shell find . \( -name "*.md" -o -name "*.yaml" \) \
                                -type f | sort)
# Exclude the ocb-generated ./build/ tree — it's a throwaway module, not
# part of the repo.
ALL_MODULES := $(shell find . -type f -name "go.mod" -not -path "./build/*" -exec dirname {} \; | sort )
ALL_MDATAGEN_MODULES := $(shell find . -type f -name "metadata.yaml" -exec dirname {} \; | sort )

# All source code files
ALL_SRC := $(shell find . -name '*.go' -o -name '*.sh' -o -name '*.ps1' -o -name 'Dockerfile*' -type f | sort)

OUTDIR=./dist
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

INTEGRATION_TEST_ARGS?=-tags integration

TOOLS_MOD_DIR := ./internal/tools

# Directories migrated to the contrib repo, excluded from testing.
# Keep in sync with the components filter in .github/workflows/checks.yml.
MIGRATED_MODULE_PATTERNS := $(shell cat migrated-modules.txt)

ifeq ($(GOOS), windows)
EXT?=.exe
else
EXT?=
endif

SNAPSHOT := $(shell git rev-parse --short HEAD)
PREVIOUS_TAG := $(shell git tag --sort=v:refname --no-contains HEAD | grep -E "[0-9]+\.[0-9]+\.[0-9]+$$" | grep -v 'version' | tail -n1)
CURRENT_TAG := $(shell git tag --sort=v:refname --points-at HEAD | grep -E "v[0-9]+\.[0-9]+\.[0-9]+$$" | grep -v 'version' | tail -n1)
# Version will be the tag pointing to the current commit, or the previous version tag if there is no such tag
VERSION ?= $(if $(CURRENT_TAG),$(CURRENT_TAG),$(PREVIOUS_TAG)-SNAPSHOT-$(SNAPSHOT))
# Tag passed to goreleaser in --snapshot mode. Must be the bare tag; goreleaser's snapshot
# template appends its own -SNAPSHOT-<sha> suffix, so passing VERSION here would double it.
SNAPSHOT_TAG := $(if $(CURRENT_TAG),$(CURRENT_TAG),$(PREVIOUS_TAG))

# Build-info stamps. These get linked into the binary via -ldflags so
# `collector --version` shows real values instead of "unknown".
GIT_HASH ?= $(shell git rev-parse HEAD)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# AGENT_LDFLAGS stamps version + git hash + build date into the v1 collector
# binaries (both consume github.com/observiq/bindplane-otel-contrib/pkg/version).
AGENT_LDFLAGS = -s -w \
	-X github.com/observiq/bindplane-otel-contrib/pkg/version.version=$(VERSION) \
	-X github.com/observiq/bindplane-otel-contrib/pkg/version.gitHash=$(GIT_HASH) \
	-X github.com/observiq/bindplane-otel-contrib/pkg/version.date=$(BUILD_DATE)

# AGENT_BUILD_TAGS are the build tags that should be used when building BDOT
# 'bindplane' builds with logic used by the v1 OpAMP implementation
# 'embed_library' used by the telemetry generator receiver to use blitz (PR#3525)
AGENT_BUILD_TAGS = bindplane embed_library

# UPDATER_LDFLAGS stamps the same values into the updater binary.
UPDATER_LDFLAGS = -s -w \
	-X github.com/observiq/bindplane-otel-collector/updater/internal/version.version=$(VERSION) \
	-X github.com/observiq/bindplane-otel-collector/updater/internal/version.gitHash=$(GIT_HASH) \
	-X github.com/observiq/bindplane-otel-collector/updater/internal/version.date=$(BUILD_DATE)

.PHONY: version
version:
	@printf $(VERSION)

# Build binaries for current GOOS/GOARCH by default
.DEFAULT_GOAL := build-binaries

# ocb-driven build:
#   1. Run the OTel collector builder against manifests/observIQ/manifest.yaml
#      to generate ./build/ (components.go, go.mod, etc).
#   2. Overwrite the generated main.go with the v1 entry point shipped by
#      the opampconnectionextension at cmd/main/main.go. That entry point
#      wires ocb's factories into the managed/standalone runtime via
#      runtime.Run.
#   3. go mod tidy + go build inside ./build/.
#
# Resolve ocb to the first of:
#   - the OCB env var
#   - `builder` on PATH
#   - $(GOBIN)/builder (else $$HOME/go/bin/builder)
#
# Install with: make install-ocb
OCB_VERSION ?= v0.155.0
OCB ?= $(shell command -v $${OCB:-builder} 2>/dev/null || echo $${GOBIN:-$$HOME/go/bin}/builder)

# Installs the ocb builder at the pinned version. The single source of truth
# for the ocb version — CI workflows call this instead of pinning their own.
.PHONY: install-ocb
install-ocb:
	go install go.opentelemetry.io/collector/cmd/builder@$(OCB_VERSION)
MANIFEST ?= manifests/observIQ/manifest.yaml
BUILD_DIR ?= ./build
AGENT_MAIN ?= internal/extension/opampconnectionextension/cmd/main/main.go

# verify-manifest is the CI gate: regenerate sources from the manifest and
# compile them. Fails on any unresolvable component, missing replace, or
# version drift between sibling deps. Runs against every PR that touches
# the manifest or any sub-module.
.PHONY: verify-manifest
verify-manifest:
	@if [ ! -x "$(OCB)" ]; then \
		echo "ocb not found at $(OCB). Install with: make install-ocb"; \
		exit 1; \
	fi
	rm -rf $(BUILD_DIR)
	$(OCB) --config $(MANIFEST) --skip-compilation
	cp $(AGENT_MAIN) $(BUILD_DIR)/main.go
	rm -f $(BUILD_DIR)/main_others.go $(BUILD_DIR)/main_windows.go
	cd $(BUILD_DIR) && go mod tidy
	# Compile with the same build tags as the release build ($(AGENT_BUILD_TAGS))
	# so the gate exercises exactly what `make agent` ships.
	cd $(BUILD_DIR) && CGO_ENABLED=0 go build -tags "$(AGENT_BUILD_TAGS)" -o /dev/null .

# Builds the collector for the current GOOS/GOARCH pair using ocb.
.PHONY: agent
agent:
	@if [ ! -x "$(OCB)" ]; then \
		echo "ocb not found at $(OCB). Install with: make install-ocb"; \
		exit 1; \
	fi
	$(OCB) --config $(MANIFEST) --skip-compilation
	cp $(AGENT_MAIN) $(BUILD_DIR)/main.go
	# Drop ocb's run/runInteractive helpers — our main.go owns startup.
	rm -f $(BUILD_DIR)/main_others.go $(BUILD_DIR)/main_windows.go
	cd $(BUILD_DIR) && go mod tidy
	cd $(BUILD_DIR) && CGO_ENABLED=0 go build -tags "$(AGENT_BUILD_TAGS)" -ldflags "$(AGENT_LDFLAGS)" -o ../$(OUTDIR)/collector_$(GOOS)_$(GOARCH)$(EXT) .

# Builds just the updater for current GOOS/GOARCH pair
.PHONY: updater
updater:
	cd ./updater/; CGO_ENABLED=0 go build -ldflags "$(UPDATER_LDFLAGS)" -o ../$(OUTDIR)/updater_$(GOOS)_$(GOARCH)$(EXT) ./cmd/updater

# Builds the updater + agent for current GOOS/GOARCH pair
.PHONY: build-binaries
build-binaries: agent updater

.PHONY: build-all
build-all: build-linux build-darwin build-windows

# v1 doesn't ship AIX binaries today, so build-all and build-all-non-aix are equivalent.
# Goreleaser's before-hook uses this target name to handle the release flow.
.PHONY: build-all-non-aix
build-all-non-aix: build-all

.PHONY: build-linux
build-linux: build-linux-amd64 build-linux-arm64 build-linux-arm build-linux-ppc64 build-linux-ppc64le

.PHONY: build-darwin
build-darwin: build-darwin-amd64 build-darwin-arm64

.PHONY: build-windows
build-windows: build-windows-amd64 build-windows-arm64

.PHONY: build-linux-ppc64
build-linux-ppc64:
	CGO_ENABLED=0 GOOS=linux GOARCH=ppc64 $(MAKE) build-binaries -j2

.PHONY: build-linux-ppc64le
build-linux-ppc64le:
	CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le $(MAKE) build-binaries -j2

.PHONY: build-linux-amd64
build-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(MAKE) build-binaries -j2

.PHONY: build-linux-arm64
build-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(MAKE) build-binaries -j2

.PHONY: build-linux-arm
build-linux-arm:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm $(MAKE) build-binaries -j2

.PHONY: build-darwin-amd64
build-darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(MAKE) build-binaries -j2

.PHONY: build-darwin-arm64
build-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(MAKE) build-binaries -j2

.PHONY: build-windows-amd64
build-windows-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(MAKE) build-binaries -j2

.PHONY: build-windows-arm64
build-windows-arm64:
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 $(MAKE) build-binaries -j2

# tool-related commands
.PHONY: install-tools
install-tools:
	cd $(TOOLS_MOD_DIR) && go install github.com/client9/misspell/cmd/misspell
	cd $(TOOLS_MOD_DIR) && go install github.com/google/addlicense
	cd $(TOOLS_MOD_DIR) && go install github.com/mgechev/revive
	cd $(TOOLS_MOD_DIR) && go install go.opentelemetry.io/collector/cmd/mdatagen
	cd $(TOOLS_MOD_DIR) && go install github.com/securego/gosec/v2/cmd/gosec
# update cosign in release.yml when updating this version
# update cosign in docs/verify-signature.md when updating this version
	go install github.com/sigstore/cosign/cmd/cosign@v1.13.1
	cd $(TOOLS_MOD_DIR) && go install github.com/uw-labs/lichen
	cd $(TOOLS_MOD_DIR) && go install github.com/vektra/mockery/v2
	cd $(TOOLS_MOD_DIR) && go install golang.org/x/tools/cmd/goimports
	cd $(TOOLS_MOD_DIR) && go install gotest.tools/gotestsum

# Fast install that checks if tools exist (for CI with cache)
.PHONY: install-tools-ci
install-tools-ci:
	@command -v misspell > /dev/null 2>&1 || (cd $(TOOLS_MOD_DIR) && go install github.com/client9/misspell/cmd/misspell)
	@command -v addlicense > /dev/null 2>&1 || (cd $(TOOLS_MOD_DIR) && go install github.com/google/addlicense)
	@command -v revive > /dev/null 2>&1 || (cd $(TOOLS_MOD_DIR) && go install github.com/mgechev/revive)
	@command -v mdatagen > /dev/null 2>&1 || (cd $(TOOLS_MOD_DIR) && go install go.opentelemetry.io/collector/cmd/mdatagen)
	@command -v gosec > /dev/null 2>&1 || (cd $(TOOLS_MOD_DIR) && go install github.com/securego/gosec/v2/cmd/gosec)
	@command -v cosign > /dev/null 2>&1 || go install github.com/sigstore/cosign/cmd/cosign@v1.13.1
	@command -v lichen > /dev/null 2>&1 || (cd $(TOOLS_MOD_DIR) && go install github.com/uw-labs/lichen)
	@command -v mockery > /dev/null 2>&1 || (cd $(TOOLS_MOD_DIR) && go install github.com/vektra/mockery/v2)
	@command -v goimports > /dev/null 2>&1 || (cd $(TOOLS_MOD_DIR) && go install golang.org/x/tools/cmd/goimports)
	@command -v gotestsum > /dev/null 2>&1 || (cd $(TOOLS_MOD_DIR) && go install gotest.tools/gotestsum)

.PHONY: lint
lint:
	revive -config revive/config.toml -formatter friendly ./...

.PHONY: misspell
misspell:
	misspell -error $(ALLDOC)

.PHONY: misspell-fix
misspell-fix:
	misspell -w $(ALLDOC)

.PHONY: test
test:
	@if [ -n "$$(go list ./... 2>/dev/null)" ]; then \
		echo "running tests in root"; \
		gotestsum --rerun-fails --packages="./..." -- -race; \
	fi
	@set -e; for dir in $(ALL_MODULES); do \
		if [ "$${dir}" = "." ]; then continue; fi; \
		SKIP=false; \
		for pattern in $(MIGRATED_MODULE_PATTERNS); do \
			case "$${dir}" in "./$${pattern}"*) SKIP=true; break;; esac; \
		done; \
		if [ "$${SKIP}" = "true" ]; then \
			echo "skipping migrated module $${dir}"; \
			continue; \
		fi; \
		(cd "$${dir}" && \
			echo "running tests in $${dir}" && \
			gotestsum --rerun-fails --packages="./..." -- -race) || exit 1; \
	done

.PHONY: test-no-race
test-no-race:
	$(MAKE) for-all CMD="gotestsum --rerun-fails --packages="./..." "

.PHONY: test-with-cover
test-with-cover:
	$(MAKE) for-all CMD="go test -coverprofile=cover.out ./..."
	$(MAKE) for-all CMD="go tool cover -html=cover.out -o cover.html"

.PHONY: test-updater-integration
test-updater-integration:
	cd updater; gotestsum --rerun-fails --packages="./..." -- \
		$(INTEGRATION_TEST_ARGS) -race

.PHONY: bench
bench:
	$(MAKE) for-all CMD="go test -benchmem -run=^$$ -bench ^* ./..."

.PHONY: check-fmt
check-fmt:
	goimports -d ./ | diff -u /dev/null -

.PHONY: fmt
fmt:
	goimports -w .

.PHONY: tidy
tidy:
	$(MAKE) for-all-modules CMD="go mod tidy -compat=1.26.4"

.PHONY: gosec
gosec:
	@set -e; for dir in $(ALL_MODULES); do \
		case "$${dir}" in \
			"."|"./updater"|"./internal/tools"|"./cmd/plugindocgen") continue;; \
		esac; \
		SKIP=false; \
		for pattern in $(MIGRATED_MODULE_PATTERNS); do \
			case "$${dir}" in "./$${pattern}"*) SKIP=true; break;; esac; \
		done; \
		if [ "$${SKIP}" = "true" ]; then \
			echo "skipping migrated module $${dir}"; \
			continue; \
		fi; \
		(cd "$${dir}" && \
			echo "running gosec in $${dir}" && \
			gosec ./...) || exit 1; \
	done
# exclude the testdata dir; it contains a go program for testing.
	cd updater; gosec -exclude-dir internal/service/testdata ./...

# This target performs all checks that CI will do (excluding the build itself)
.PHONY: ci-checks
ci-checks: check-fmt check-license check-mod-paths check-dependabot misspell lint gosec test

# This target checks that every go.mod has the correct module path.
# Root must be github.com/observiq/bindplane-otel-collector.
# Subdirectories must be github.com/observiq/bindplane-otel-collector/<relative-path>.
# Modules with legacy paths that cannot be renamed are excluded.
MOD_PATH_EXCLUDES := ./cmd/plugindocgen
.PHONY: check-mod-paths
check-mod-paths:
	@FAILED=0; \
	for dir in $(ALL_MODULES); do \
		SKIP=false; \
		for pattern in $(MIGRATED_MODULE_PATTERNS); do \
			case "$${dir}" in "./$${pattern}"*) SKIP=true; break;; esac; \
		done; \
		if [ "$${SKIP}" = "true" ]; then continue; fi; \
		case " $(MOD_PATH_EXCLUDES) " in *" $${dir} "*) continue ;; esac; \
		MOD=$$(head -1 "$${dir}/go.mod" | sed 's/^module //'); \
		if [ "$${dir}" = "." ]; then \
			EXPECTED="github.com/observiq/bindplane-otel-collector"; \
		else \
			RELPATH=$$(echo "$${dir}" | sed 's|^\./||'); \
			EXPECTED="github.com/observiq/bindplane-otel-collector/$${RELPATH}"; \
		fi; \
		if [ "$${MOD}" != "$${EXPECTED}" ]; then \
			echo "MISMATCH: $${dir}/go.mod"; \
			echo "  got:      $${MOD}"; \
			echo "  expected: $${EXPECTED}"; \
			FAILED=1; \
		fi; \
	done; \
	if [ "$${FAILED}" -eq 1 ]; then \
		echo ""; \
		echo "check-mod-paths FAILED: module paths must match directory structure."; \
		exit 1; \
	else \
		echo "Check module paths finished successfully"; \
	fi

# This target checks that every directory with a go.mod has an entry in dependabot.yml
.PHONY: check-dependabot
check-dependabot:
	@FAILED=0; \
	DEPENDABOT_DIRS=$$(grep 'directory:' .github/dependabot.yml | sed 's/.*directory: *"\(.*\)"/\1/'); \
	for dir in $(ALL_MODULES); do \
		SKIP=false; \
		for pattern in $(MIGRATED_MODULE_PATTERNS); do \
			case "$${dir}" in "./$${pattern}"*) SKIP=true; break;; esac; \
		done; \
		if [ "$${SKIP}" = "true" ]; then continue; fi; \
		if [ "$${dir}" = "." ]; then \
			EXPECTED="/"; \
		else \
			EXPECTED=$$(echo "$${dir}" | sed 's|^\./|/|'); \
		fi; \
		if ! echo "$${DEPENDABOT_DIRS}" | grep -qx "$${EXPECTED}"; then \
			echo "MISSING: $${dir}/go.mod has no entry in .github/dependabot.yml (expected directory: \"$${EXPECTED}\")"; \
			FAILED=1; \
		fi; \
	done; \
	if [ "$${FAILED}" -eq 1 ]; then \
		echo ""; \
		echo "check-dependabot FAILED: add missing entries to .github/dependabot.yml"; \
		exit 1; \
	else \
		echo "Check dependabot finished successfully"; \
	fi

# This target checks that license copyright header is on every source file
.PHONY: check-license
check-license:
	@ADDLICENSEOUT=`addlicense -check $(ALL_SRC) 2>&1`; \
		if [ "$$ADDLICENSEOUT" ]; then \
			echo "addlicense FAILED => add License errors:\n"; \
			echo "$$ADDLICENSEOUT\n"; \
			echo "Use 'make add-license' to fix this."; \
			exit 1; \
		else \
			echo "Check License finished successfully"; \
		fi

# This target adds a license copyright header is on every source file that is missing one
.PHONY: add-license
add-license:
	@ADDLICENSEOUT=`addlicense -y "" -c "observIQ, Inc." $(ALL_SRC) 2>&1`; \
		if [ "$$ADDLICENSEOUT" ]; then \
			echo "addlicense FAILED => add License errors:\n"; \
			echo "$$ADDLICENSEOUT\n"; \
			exit 1; \
		else \
			echo "Add License finished successfully"; \
		fi

# update-otel attempts to update otel dependencies in go.mods,
# and update the otel versions in the docs.
# Usage: make update-otel OTEL_VERSION=vx.x.x CONTRIB_VERSION=vx.x.x PDATA_VERSION=vx.x.x-rcx BDOT_CONTRIB_VERSION=vx.x.x
.PHONY: update-otel
update-otel:
	./scripts/update-otel.sh "$(OTEL_VERSION)" "$(CONTRIB_VERSION)" "$(PDATA_VERSION)"
	./scripts/update-docs.sh "$(OTEL_VERSION)" "$(CONTRIB_VERSION)" "$(BDOT_CONTRIB_VERSION)"
	$(MAKE) tidy
# Double make tidy - this unfortunately is needed due to the order in which modules are tidied.
# The modules this seems to effect are plugindocgen and bindplaneextension
	$(MAKE) tidy

# update-modules updates all submodules to be the new version.
# Usage: make update-modules NEW_VERSION=vx.x.x
.PHONY: update-modules
update-modules:
	./scripts/update-module-version.sh "$(NEW_VERSION)"
	$(MAKE) tidy

# update-contrib updates all bindplane-otel-contrib dependencies to the new version.
# Usage: make update-contrib BDOT_CONTRIB_VERSION=vx.x.x
.PHONY: update-contrib
update-contrib:
	./scripts/update-bindplane-contrib.sh "$(BDOT_CONTRIB_VERSION)"
	$(MAKE) tidy

# Downloads and setups dependencies that are packaged with binary
.PHONY: release-prep
release-prep:
	@rm -rf release_deps
	@mkdir release_deps
	@echo 'v$(CURR_VERSION)' > release_deps/VERSION.txt
	./buildscripts/download-dependencies.sh release_deps
	@cp -r ./plugins release_deps/
	@cp config/example.yaml release_deps/config.yaml
	@cp config/logging.yaml release_deps/logging.yaml
	@cp service/com.observiq.collector.plist release_deps/com.observiq.collector.plist
	@jq ".files[] | select(.service != null)" windows/wix.json >> release_deps/windows_service.json

.PHONY: release-prep-gpg
release-prep-gpg:
	$(MAKE) release-prep
	@cp -r ./signature/gpg release_deps/gpg
	@rm release_deps/gpg/revocations.md
	@rm release_deps/gpg/deb-revocations/.keep
	@cd release_deps/gpg && tar -czf ../gpg-keys.tar.gz .

# Build and sign, skip release and ignore dirty git tree
.PHONY: release-test
release-test:
# If there are no MSIs in the root dir, we'll create dummy ones so that goreleaser can complete successfully
	if [ ! -e "./observiq-otel-collector.msi" ]; then touch ./observiq-otel-collector.msi; fi
	if [ ! -e "./observiq-otel-collector-arm64.msi" ]; then touch ./observiq-otel-collector-arm64.msi; fi
	SIGNING_KEY_FILE="fake-file" GORELEASER_CURRENT_TAG=$(SNAPSHOT_TAG) goreleaser release --parallelism 4 --skip=publish --skip=validate --skip=sign --clean --snapshot

.PHONY: release-containers-test
release-containers-test:
	$(MAKE) -j3 agent-linux-amd64 agent-linux-arm64 agent-linux-ppc64le
	mkdir -p tmp
	mv ./dist/collector_linux_amd64 ./tmp/collector_linux_amd64
	mv ./dist/collector_linux_arm64 ./tmp/collector_linux_arm64
	mv ./dist/collector_linux_ppc64le ./tmp/collector_linux_ppc64le
	GORELEASER_CURRENT_TAG=$(SNAPSHOT_TAG) goreleaser release --parallelism 4 --skip=publish --skip=validate --skip=sign --clean --snapshot --config .goreleaser-docker.yml

.PHONY: agent-linux-amd64 agent-linux-arm64 agent-linux-ppc64le
agent-linux-amd64:
	GOARCH=amd64 GOOS=linux $(MAKE) agent
agent-linux-arm64:
	GOARCH=arm64 GOOS=linux $(MAKE) agent
agent-linux-ppc64le:
	GOARCH=ppc64le GOOS=linux $(MAKE) agent

# agent-clean wipes the ocb-generated output trees. The ocb step is
# platform-agnostic Go-source generation, so subsequent platform builds
# reuse the generated tree until you clean it.
.PHONY: agent-clean
agent-clean:
	rm -rf $(BUILD_DIR) ./builder

.PHONY: agent-windows-amd64 agent-windows-arm64
agent-windows-amd64:
	GOARCH=amd64 GOOS=windows $(MAKE) agent
agent-windows-arm64:
	GOARCH=arm64 GOOS=windows $(MAKE) agent


build-single:
	$(MAKE)
	SIGNING_KEY_FILE="fake-file" GORELEASER_CURRENT_TAG=$(SNAPSHOT_TAG) goreleaser release --skip=publish --skip=sign --clean --skip=validate --snapshot --single-target

.PHONY: release-test-single
release-test-single:
	GORELEASER_CURRENT_TAG=$(SNAPSHOT_TAG) goreleaser release -f .goreleaser.arm64.yml --skip=publish --clean --skip=validate --snapshot

.PHONY: for-all
for-all:
	@if [ -n "$$(go list ./... 2>/dev/null)" ]; then \
		echo "running $${CMD} in root"; \
		$${CMD}; \
	fi
	@set -e; for dir in $(ALL_MODULES); do \
	  if [ "$${dir}" = "." ]; then continue; fi; \
	  SKIP=false; \
	  for pattern in $(MIGRATED_MODULE_PATTERNS); do \
	    case "$${dir}" in "./$${pattern}"*) SKIP=true; break;; esac; \
	  done; \
	  if [ "$${SKIP}" = "true" ]; then \
	    echo "skipping migrated module $${dir}"; \
	    continue; \
	  fi; \
	  (cd "$${dir}" && \
	    echo "running $${CMD} in $${dir}" && \
	    $${CMD} ); \
	done

.PHONY: for-all-modules
for-all-modules:
	@set -e; for dir in $(ALL_MODULES); do \
	  if [ "$${dir}" = "." ]; then continue; fi; \
	  SKIP=false; \
	  for pattern in $(MIGRATED_MODULE_PATTERNS); do \
	    case "$${dir}" in "./$${pattern}"*) SKIP=true; break;; esac; \
	  done; \
	  if [ "$${SKIP}" = "true" ]; then \
	    echo "skipping migrated module $${dir}"; \
	    continue; \
	  fi; \
	  (cd "$${dir}" && \
	    echo "running $${CMD} in $${dir}" && \
	    $${CMD} ); \
	done

# Release a new version of the agent. This will also tag all submodules
.PHONY: release
release:
	@if [ -z "$(version)" ]; then \
		echo "version was not set"; \
		exit 1; \
	fi

	@if ! [[ "$(version)" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$$ ]]; then \
		echo "version $(version) is invalid semver"; \
		exit 1; \
	fi

	@git tag $(version)
	@git push --tags

	@set -e; for dir in $(ALL_MODULES); do \
	  if [ $${dir} == \. ]; then \
	  	continue; \
	  fi; \
	  SKIP=false; \
	  for pattern in $(MIGRATED_MODULE_PATTERNS); do \
	    case "$${dir}" in "./$${pattern}"*) SKIP=true; break;; esac; \
	  done; \
	  if [ "$${SKIP}" = "true" ]; then \
	    echo "skipping migrated module $${dir}"; \
	    continue; \
	  fi; \
	  echo "$${dir}" | sed -e "s+^./++" -e 's+$$+/$(version)+' | awk '{print $$1}' | git tag $$(cat); \
	done

	@git push --tags

.PHONY: clean
clean:
	rm -rf $(OUTDIR)

.PHONY: scan-licenses
scan-licenses:
	lichen --config=./license.yaml $$(find dist/collector_* dist/updater_*)

.PHONY: generate
generate:
	$(MAKE) for-all CMD="go generate ./..."
	$(MAKE) fmt

.PHONY: create-plugin-docs
create-plugin-docs:
	cd cmd/plugindocgen; go run .
