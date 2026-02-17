.PHONY: vagrant-prep
vagrant-prep:
	./scripts/vagrant-prep.sh

.PHONY: fetch-dependencies
fetch-dependencies:
	./scripts/fetch-dependencies.sh

.PHONY: build-msi
build-msi: fetch-dependencies
	cd ..; $(MAKE) build-windows
	./scripts/build-msi.sh amd64
	./scripts/build-msi.sh arm64

.PHONY: test-install-msi
test-install-msi:
	./scripts/test-install-msi.sh

.PHONY: test-uninstall-msi
test-uninstall-msi:
	./scripts/test-uninstall-msi.sh

.PHONY: clean
clean:
	rm -rf go-msi.exe config.yaml cinc-auditor.msi observiq-otel-collector*.exe observiq-otel-collector*.msi \
		opentelemetry-java-contrib-jmx-metrics.jar wix-binaries.zip wix.dynamic.json \
		wix plugins LICENSE VERSION.txt
