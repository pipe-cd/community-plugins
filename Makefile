.PHONY: sync
sync: sync/plugins-list sync/codeowners

.PHONY: sync/plugins-list
sync/plugins-list:
	bash hack/sync-plugins-list.sh

.PHONY: sync/codeowners
sync/codeowners:
	bash hack/sync-codeowners.sh

.PHONY: init/plugin
init/plugin:
	bash hack/init-plugin.sh $(PLUGIN_DIR_NAME) $(CODEOWNERS)

.PHONY: lint/go
lint/go: FIX ?= false
lint/go: VERSION ?= sha256:991246ff030683f8c6fbbc101c5fb477b54f6562b60d41e8a585eb8bac42225b # golangci/golangci-lint:v2.1.6 linux/arm64
lint/go: FLAGS ?= --rm -e GOCACHE=/repo/.cache/go-build -e GOLANGCI_LINT_CACHE=/repo/.cache/golangci-lint -v ${PWD}:/repo -it
lint/go: MODULES ?= $(shell find . -name go.mod | while read -r dir; do dirname "$$dir"; done | paste -sd, -) # comma separated list of modules. e.g. MODULES=.,plugins/xxx-plugin
lint/go:
	@echo "Linting go modules..."
	@for module in $(shell echo $(MODULES) | tr ',' ' '); do \
		echo "Linting module: $$module"; \
		docker run ${FLAGS} -w /repo/$$module golangci/golangci-lint@${VERSION} golangci-lint run -v --config /repo/.golangci.yml --fix=$(FIX); \
	done

.PHONY: build/go
build/go: PLUGINS_BIN_DIR ?= ~/.piped/plugins
build/go: PLUGINS_SRC_DIR ?= ./plugins
build/go: PLUGINS_OUT_DIR ?= ${PWD}/.artifacts/plugins
build/go: PLUGINS ?= $(shell find $(PLUGINS_SRC_DIR) -mindepth 1 -maxdepth 1 -type d | while read -r dir; do basename "$$dir"; done | paste -sd, -) # comma separated list of plugins. eg: PLUGINS=kubernetes,ecs,lambda
build/go: BUILD_OPTS ?= -ldflags "-s -w" -trimpath
build/go: BUILD_OS ?= $(shell go version | cut -d ' ' -f4 | cut -d/ -f1)
build/go: BUILD_ARCH ?= $(shell go version | cut -d ' ' -f4 | cut -d/ -f2)
build/go: BUILD_ENV ?= GOOS=$(BUILD_OS) GOARCH=$(BUILD_ARCH) CGO_ENABLED=0
build/go: BIN_SUFFIX ?=
build/go:
	mkdir -p $(PLUGINS_BIN_DIR)
	@echo "Building plugins..."
	@for plugin in $(shell echo $(PLUGINS) | tr ',' ' '); do \
		if [ ! -f $(PLUGINS_SRC_DIR)/$$plugin/go.mod ]; then \
			echo "⏭️ Skipped plugin: $$plugin (no go.mod found)"; \
			continue; \
		fi; \
		echo "🔨 Building plugin: $$plugin"; \
		$(BUILD_ENV) go -C $(PLUGINS_SRC_DIR)/$$plugin build $(BUILD_OPTS) -o $(PLUGINS_OUT_DIR)/$${plugin}$(BIN_SUFFIX) . \
			&& cp $(PLUGINS_OUT_DIR)/$${plugin}$(BIN_SUFFIX) $(PLUGINS_BIN_DIR)/$$plugin; \
		if [ $$? -ne 0 ]; then \
			echo "❌ Failed to build plugin: $$plugin"; \
			exit 1; \
		fi; \
	done
	@echo "✅ Plugins are built to $(PLUGINS_OUT_DIR) and copied to $(PLUGINS_BIN_DIR)"

# test/go can be overridden by each plugin.
.PHONY: test/go
# test/go: COVERAGE ?= false
# test/go: COVERAGE_OUTPUT ?= ${PWD}/coverage.out
# test/go: COVERAGE_OPTS ?= -covermode=atomic -coverprofile=${COVERAGE_OUTPUT}.tmp
test/go: PLUGINS ?= $(shell find ./plugins -name go.mod | while read -r dir; do basename $$(dirname "$$dir"); done | paste -sd, -) # comma separated list of plugins. e.g.: PLUGINS=example-stage,yyy
test/go:
# TODO: Use the COVERAGE flag and report in CI.
	@echo "PLUGINS: $(PLUGINS)"
	@echo "Testing plugins..."
	@for plugin in $(shell echo $(PLUGINS) | tr ',' ' '); do \
		echo "🧪 Testing plugin: $$plugin"; \
		make common/test/go -C ./plugins/$$plugin; \
		if [ $$? -ne 0 ]; then \
			echo "❌ Failed to test plugin: $$plugin"; \
		fi; \
	done

# TODO
# .PHONY: push/plugins

.PHONY: precommit
precommit: lint/go build/go test/go sync
	bash hack/ensure-dco.sh