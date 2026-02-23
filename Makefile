# All source code and documents, used when checking for misspellings
ALLDOC := $(shell find . \( -name "*.md" -o -name "*.yaml" \) \
                                -type f | sort)
ALL_MODULES := $(shell find . -path ./builder -prune -o -type f -name "go.mod" -exec dirname {} \; | sort )
ALL_MDATAGEN_MODULES := $(shell find . -type f -name "metadata.yaml" -exec dirname {} \; | sort )

# All source code files
ALL_SRC := $(shell find . -path ./builder -prune -o -name '*.go' -o -name '*.sh' -o -name 'Dockerfile*' -type f | sort)

OUTDIR=./dist
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

INTEGRATION_TEST_ARGS?=-tags integration

TOOLS_MOD_DIR := ./internal/tools

ifeq ($(GOOS), windows)
EXT?=.exe
else
EXT?=
endif

SNAPSHOT := $(shell git rev-parse --short HEAD)
PREVIOUS_TAG := $(shell git tag --sort=v:refname --no-contains HEAD | grep -E "[0-9]+\.[0-9]+\.[0-9]+$$" | tail -n1)
CURRENT_TAG := $(shell git tag --sort=v:refname --points-at HEAD | grep -E "v[0-9]+\.[0-9]+\.[0-9]+$$" | tail -n1)
# Version will be the tag pointing to the current commit, or the previous version tag if there is no such tag
VERSION ?= $(if $(CURRENT_TAG),$(CURRENT_TAG),$(PREVIOUS_TAG)-SNAPSHOT-$(SNAPSHOT))

.PHONY: version
version:
	@printf $(VERSION)

# Build binaries for current GOOS/GOARCH by default
.DEFAULT_GOAL := collector

# Builds the collector for current GOOS/GOARCH pair
.PHONY: collector
collector:
	CGO_ENABLED=0 builder --config="./manifests/observIQ/manifest.yaml" --ldflags "-s -w -X github.com/observiq/bindplane-otel-collector/version.version=$(VERSION)"
	mkdir -p $(OUTDIR); cp ./builder/bindplane-otel-collector $(OUTDIR)/collector_$(GOOS)_$(GOARCH)$(EXT)

# Builds the agent for current GOOS/GOARCH pair (aix)
.PHONY: collector-aix
collector-aix:
	CGO_ENABLED=0 builder --config="./manifests/observIQ/manifest-aix.yaml" --ldflags "-s -w -X github.com/observiq/bindplane-otel-collector/version.version=$(VERSION)"
	mkdir -p $(OUTDIR); cp ./builder/bindplane-otel-collector $(OUTDIR)/collector_$(GOOS)_$(GOARCH)$(EXT)

# Builds a custom distro for the current GOOS/GOARCH pair using the manifest specified
# MANIFEST = path to the manifest file for the distro to be built
# Usage: make distro MANIFEST="./manifests/custom/my_distro_manifest.yaml"
.PHONY: distro
distro:
	builder --config="$(MANIFEST)"

# Runs the supervisor invoking the collector build in /dist
.PHONY: run-supervisor
run-supervisor:
	opampsupervisor --config ./local/supervisor.yaml

# Ensures the supervisor and collector are stopped
.PHONY: kill
kill:
	pkill -9 opampsupervisor || true
	pkill -9 collector_$(GOOS)_$(GOARCH) || true

# Stops processes and cleans up
.PHONY: reset
reset: kill
	rm -rf agent.log effective.yaml local/supervisor_storage/ builder/

.PHONY: build-all
build-all: build-linux build-unix build-windows

# Build all non-AIX platforms (used by main goreleaser before hook)
.PHONY: build-all-non-aix
build-all-non-aix: build-linux build-darwin build-windows

.PHONY: build-linux
build-linux: build-linux-amd64 build-linux-arm64 build-linux-ppc64 build-linux-ppc64le

.PHONY: build-unix
build-unix: build-darwin build-aix

.PHONY: build-aix
build-aix: build-aix-ppc64

.PHONY: build-darwin
build-darwin: build-darwin-amd64 build-darwin-arm64

.PHONY: build-windows
build-windows: build-windows-amd64

.PHONY: build-linux-ppc64
build-linux-ppc64:
	GOOS=linux GOARCH=ppc64 $(MAKE) collector

.PHONY: build-linux-ppc64le
build-linux-ppc64le:
	GOOS=linux GOARCH=ppc64le $(MAKE) collector

.PHONY: build-linux-amd64
build-linux-amd64:
	GOOS=linux GOARCH=amd64 $(MAKE) collector

.PHONY: build-linux-arm64
build-linux-arm64:
	GOOS=linux GOARCH=arm64 $(MAKE) collector

.PHONY: build-linux-arm
build-linux-arm:
	GOOS=linux GOARCH=arm $(MAKE) collector

.PHONY: build-aix-ppc64
build-aix-ppc64:
	GOOS=aix GOARCH=ppc64 $(MAKE) collector-aix

.PHONY: build-darwin-amd64
build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 $(MAKE) collector

.PHONY: build-darwin-arm64
build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 $(MAKE) collector

.PHONY: build-windows-amd64
build-windows-amd64:
	GOOS=windows GOARCH=amd64 $(MAKE) collector

# tool-related commands
.PHONY: install-tools
install-tools:
	cd $(TOOLS_MOD_DIR) && go install github.com/client9/misspell/cmd/misspell
	cd $(TOOLS_MOD_DIR) && go install github.com/google/addlicense
	cd $(TOOLS_MOD_DIR) && go install github.com/mgechev/revive
	cd $(TOOLS_MOD_DIR) && go install go.opentelemetry.io/collector/cmd/mdatagen
	cd $(TOOLS_MOD_DIR) && go install go.opentelemetry.io/collector/cmd/builder
	cd $(TOOLS_MOD_DIR) && go install github.com/open-telemetry/opentelemetry-collector-contrib/cmd/opampsupervisor
	cd $(TOOLS_MOD_DIR) && go install github.com/securego/gosec/v2/cmd/gosec
	cd $(TOOLS_MOD_DIR) && go install github.com/uw-labs/lichen
	cd $(TOOLS_MOD_DIR) && go install github.com/vektra/mockery/v2
	cd $(TOOLS_MOD_DIR) && go install golang.org/x/tools/cmd/goimports
# update cosign in release.yml when updating this version
# update cosign in docs/verify-signature.md when updating this version
	go install github.com/sigstore/cosign/cmd/cosign@v1.13.1
	cd $(TOOLS_MOD_DIR) && go install gotest.tools/gotestsum


# install builder cmd for better CI
.PHONY: install-builder
install-builder:
	cd $(TOOLS_MOD_DIR) && go install go.opentelemetry.io/collector/cmd/builder

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
	$(MAKE) for-all CMD="gotestsum --rerun-fails --packages="./..." -- -race"

.PHONY: test-receivers
test-receivers:
	@set -e; for dir in $(ALL_MODULES); do \
		if echo "$${dir}" | grep -qE "^\.?/?receiver/"; then \
			(cd "$${dir}" && \
				echo "running tests in $${dir}" && \
				gotestsum --rerun-fails --packages="./..." -- -race) || exit 1; \
		fi; \
	done

.PHONY: test-processors
test-processors:
	@set -e; for dir in $(ALL_MODULES); do \
		if echo "$${dir}" | grep -qE "^\.?/?processor/"; then \
			(cd "$${dir}" && \
				echo "running tests in $${dir}" && \
				gotestsum --rerun-fails --packages="./..." -- -race) || exit 1; \
		fi; \
	done

.PHONY: test-exporters
test-exporters:
	@set -e; for dir in $(ALL_MODULES); do \
		if echo "$${dir}" | grep -qE "^\.?/?exporter/"; then \
			(cd "$${dir}" && \
				echo "running tests in $${dir}" && \
				gotestsum --rerun-fails --packages="./..." -- -race) || exit 1; \
		fi; \
	done

.PHONY: test-extensions
test-extensions:
	@set -e; for dir in $(ALL_MODULES); do \
		if echo "$${dir}" | grep -qE "^\.?/?extension/"; then \
			(cd "$${dir}" && \
				echo "running tests in $${dir}" && \
				gotestsum --rerun-fails --packages="./..." -- -race) || exit 1; \
		fi; \
	done

.PHONY: test-other
test-other:
	@PACKAGES=$$(go list ./... | grep -v "/receiver" | grep -v "/processor" | grep -v "/exporter" | grep -v "/extension" | tr '\n' ' '); \
	if [ -n "$$PACKAGES" ]; then \
		gotestsum --rerun-fails --packages="$$PACKAGES" -- -race; \
	fi
	@set -e; for dir in $(ALL_MODULES); do \
		if [ "$${dir}" = "." ]; then \
			continue; \
		elif echo "$${dir}" | grep -qE "^(receiver|processor|exporter|extension)/"; then \
			continue; \
		else \
			(cd "$${dir}" && \
				echo "running tests in $${dir}" && \
				gotestsum --rerun-fails --packages="./..." -- -race) || exit 1; \
		fi; \
	done

.PHONY: test-no-race
test-no-race:
	$(MAKE) for-all CMD="gotestsum --rerun-fails --packages="./..." "

.PHONY: test-with-cover
test-with-cover:
	$(MAKE) for-all CMD="go test -coverprofile=cover.out ./..."
	$(MAKE) for-all CMD="go tool cover -html=cover.out -o cover.html"

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
	$(MAKE) for-all CMD="go mod tidy -compat=1.24"

.PHONY: gosec
gosec:
	cd exporter; $(MAKE) -f "../Makefile" for-all CMD="gosec --exclude-dir=internal/metadata --exclude-dir=protos/api ./..."
	cd processor; $(MAKE) -f "../Makefile" for-all CMD="gosec ./..."
	cd internal; $(MAKE) -f "../Makefile" for-all CMD="gosec ./..."
	cd extension; $(MAKE) -f "../Makefile" for-all CMD="gosec ./..."
	cd receiver; $(MAKE) -f "../Makefile" for-all CMD="gosec --exclude-dir=internal/metadata ./..."

# This target performs all checks that CI will do (excluding the build itself)
.PHONY: ci-checks
ci-checks: check-fmt check-license check-mod-paths misspell lint gosec test

# This target checks that every go.mod has the correct module path.
# Root must be github.com/observiq/bindplane-otel-collector.
# Subdirectories must be github.com/observiq/bindplane-otel-collector/<relative-path>.
# Modules with legacy paths that cannot be renamed are excluded.
MOD_PATH_EXCLUDES := ./cmd/plugindocgen
.PHONY: check-mod-paths
check-mod-paths:
	@FAILED=0; \
	for dir in $(ALL_MODULES); do \
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
# Usage: make update-otel OTEL_VERSION=vx.x.x CONTRIB_VERSION=vx.x.x PDATA_VERSION=vx.x.x-rcx
.PHONY: update-otel
update-otel:
	./scripts/update-otel.sh "$(OTEL_VERSION)" "$(CONTRIB_VERSION)" "$(PDATA_VERSION)"
	./scripts/update-docs.sh "$(OTEL_VERSION)" "$(CONTRIB_VERSION)"
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

# Downloads and setups dependencies that are packaged with binary
.PHONY: release-prep
release-prep:
	@rm -rf release_deps
	@mkdir release_deps
	@echo '$(CURR_VERSION)' > release_deps/VERSION.txt
	bash ./buildscripts/download-dependencies.sh release_deps
	@cp -r ./plugins release_deps/
	@cp -r ./signature/gpg release_deps/gpg
	@rm release_deps/gpg/revocations.md
	@rm release_deps/gpg/deb-revocations/.keep
	@cp service/com.bindplane.otel.collector.plist release_deps/com.bindplane.otel.collector.plist
	@jq ".files[] | select(.service != null)" windows/wix.json >> release_deps/windows_service.json
	@cp service/bindplane-otel-collector release_deps/bindplane-otel-collector
	@cp service/bindplane-otel-collector.aix.env release_deps/bindplane-otel-collector.aix.env

.PHONY: release-prep-aix
release-prep-aix:
	@rm -rf release_deps
	@mkdir release_deps
	@echo '$(CURR_VERSION)' > release_deps/VERSION.txt
	bash ./buildscripts/download-dependencies.sh release_deps --aix
	@cp -r ./plugins release_deps/
	@cp -r ./signature/gpg release_deps/gpg
	@rm release_deps/gpg/revocations.md
	@rm release_deps/gpg/deb-revocations/.keep
	@cp service/bindplane-otel-collector.aix.env release_deps/bindplane-otel-collector.aix.env
	@cd release_deps/gpg && tar -czf ../gpg-keys.tar.gz .

.PHONY: release-prep-gpg
release-prep-gpg:
	$(MAKE) release-prep
	@cd release_deps/gpg && tar -czf ../gpg-keys.tar.gz .

# Build and sign, skip release and ignore dirty git tree
.PHONY: release-test
release-test:
# If there is no MSI in the root dir, we'll create a dummy one so that goreleaser can complete successfully
	if [ ! -e "./bindplane-otel-collector.msi" ]; then touch ./bindplane-otel-collector.msi; fi
	SIGNING_KEY_FILE="fake-file" GORELEASER_CURRENT_TAG=$(VERSION) goreleaser release --parallelism 4 --skip=publish --skip=validate --skip=sign --clean --snapshot
	$(MAKE) release-test-aix

# Build and sign AIX, skip release and ignore dirty git tree
.PHONY: release-test-aix
release-test-aix:
	SIGNING_KEY_FILE="fake-file" GORELEASER_CURRENT_TAG=$(VERSION) goreleaser release --config .goreleaser-aix.yml --parallelism 4 --skip=publish --skip=validate --skip=sign --clean --snapshot

build-single:
	SIGNING_KEY_FILE="fake-file" GORELEASER_CURRENT_TAG=$(VERSION) goreleaser release --skip=publish --skip=sign --clean --skip=validate --snapshot --single-target


.PHONY: for-all
for-all:
	@set -e; for dir in $(ALL_MODULES); do \
	  (cd "$${dir}" && \
	  	echo "running $${CMD} in $${dir}" && \
	 	$${CMD} ); \
	done

# Release a new version of the collector. This will also tag all submodules
.PHONY: release
release:
	@if [ -z "$(version)" ]; then \
		echo "version was not set"; \
		exit 1; \
	fi

	@if ! [[ "$(version)" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-.*)? ]]; then \
		echo "version $(version) is invalid semver"; \
		exit 1; \
	fi

	@git tag $(version)
	@git push --tags

	@set -e; for dir in $(ALL_MODULES); do \
	  if [ $${dir} == \. ]; then \
	  	continue; \
	  else \
	    echo "$${dir}" | sed -e "s+^./++" -e 's+$$+/$(version)+' | awk '{print $1}' | git tag $$(cat)  ; \
	  fi; \
	done

	@git push --tags

.PHONY: clean
clean:
	rm -rf $(OUTDIR) builder/

.PHONY: scan-licenses
scan-licenses:
	lichen --config=./license.yaml $$(find dist/collector_*)

.PHONY: generate
generate:
	$(MAKE) for-all CMD="go generate ./..."
	$(MAKE) fmt

.PHONY: create-plugin-docs
create-plugin-docs:
	cd cmd/plugindocgen; go run .
