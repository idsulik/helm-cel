PLUGIN_NAME := helm-cel
VERSION := 0.1.0
DIST_DIR := dist

.PHONY: build
build:
	go build -o bin/$(PLUGIN_NAME) ./cmd/helm-cel

.PHONY: install
install: build
	mkdir -p $(HELM_PLUGIN_DIR)/bin
	cp bin/$(PLUGIN_NAME) $(HELM_PLUGIN_DIR)/bin/
	cp plugin.yaml $(HELM_PLUGIN_DIR)/

.PHONY: test
test:
	go test ./...

.PHONY: release
release:
	goreleaser release --snapshot --rm-dist

.PHONY: clean
clean:
	rm -rf $(DIST_DIR)
	rm -rf bin