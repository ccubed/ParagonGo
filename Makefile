# --- Makefile Overview ---

# - Run "make help" to show a full list of commands.
# - Comments marked with double hash signs ("##") will appear in `make help` output.
# - Most command values are overridable: `make build BIN=gomud VERSION=v1.2.3`.

# --- Makefile Variables ---

.DEFAULT_GOAL := help

VERSION ?= $(shell git rev-parse HEAD)
BIN ?= go-mud-server
GO_VERSION ?= $(shell awk '/^toolchain go/ { sub(/^toolchain go/, ""); print; found=1; exit } /^go / && !gover { gover=$$2 } END { if (!found && gover) print gover }' go.mod)

DOCKER_COMPOSE := docker-compose -f compose.yml
GO_CONSOLE_IMAGE ?= golang:$(GO_VERSION)-bookworm
DOCKER_CMD ?= bash

CI_LOCAL_IMAGE ?= gomud-ci-local
CI_LOCAL_UID ?= $(shell id -u)
CI_LOCAL_GID ?= $(shell id -g)
CI_LOCAL_DOCKER_SOCK_GID ?= $(shell stat -c '%g' /var/run/docker.sock 2>/dev/null || id -g)
CI_LOCAL_HOME ?= /home/gomud
CI_LOCAL_ACT_CACHE_DIR ?= $(PWD)/.git/.cache/act
ACT_FLAGS ?= --pull=false -P ubuntu-24.04=catthehacker/ubuntu:act-latest
ACT_DRYRUN_SECRETS ?= -s DISCORD_WEBHOOK_URL=https://example.invalid/webhook

JSHINT_VERSION ?= 2.13.6
VENDORED_JS_LINT_PATHS := _datafiles/html/admin/static/js/monaco/%
JS_LINT_PATHS := $(filter-out $(VENDORED_JS_LINT_PATHS),$(shell find _datafiles -name '*.js' -print))
WEBCLIENT_WINDOW_JS := $(shell find _datafiles/html/public/static/js/windows -name '*.js' -print)
WEBCLIENT_BASE_JS := $(filter-out $(WEBCLIENT_WINDOW_JS),$(JS_LINT_PATHS))
JSHINT := npx --yes --loglevel=error jshint@$(JSHINT_VERSION)
JSHINT_BASE_CMD := $(JSHINT) $(WEBCLIENT_BASE_JS)
JSHINT_WINDOWS_CMD := $(JSHINT) --config .jshintrc.webclient-windows $(WEBCLIENT_WINDOW_JS)

LUACHECK_DOCKER_IMAGE ?= pipelinecomponents/luacheck:latest
LUA_LINT_PATHS := $(filter-out $(VENDORED_JS_LINT_PATHS),$(shell find _datafiles -name '*.lua' -print))
LUACHECK := luacheck
LUACHECK_CMD := $(LUACHECK) $(LUA_LINT_PATHS)

CROSS_BUILD_CMD = env $(strip GOOS=$(CROSS_GOOS) GOARCH=$(CROSS_GOARCH) $(if $(CROSS_GOARM),GOARM=$(CROSS_GOARM))) go build -o $(CROSS_OUTPUT)

CI_LOCAL_RUN := docker run --rm \
	--user "$(CI_LOCAL_UID):$(CI_LOCAL_GID)" \
	--group-add "$(CI_LOCAL_DOCKER_SOCK_GID)" \
	-e HOME="$(CI_LOCAL_HOME)" \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-v "$(PWD)":/work \
	-v "$(CI_LOCAL_ACT_CACHE_DIR)":"$(CI_LOCAL_HOME)/.cache/act" \
	-w /work \
	$(CI_LOCAL_IMAGE)

export GOFLAGS := -mod=mod

# --- Makefile Commands ---

## Help
help: ## List documented Makefile targets.
	@awk ' \
		BEGIN { FS = ":.*##"; printf "\nUsage:   make <target>\nExample: make build\n" } \
		/^## / { printf "\n\033[90;3m%s\033[0m\n", substr($$0, 4); next } \
		/^[[:alnum:]_.%/-]+:.*## / { printf "  \033[93m%-24s\033[0m %s\n", $$1, $$2 } \
	' $(MAKEFILE_LIST)
	@printf "\n"

## Developer Workflow
.PHONY: build build_local generate module validate test coverage fmt fmtcheck vet mod js-lint lua-lint

build: validate build_local ## Validate the code and build ./$(BIN).

build_local: generate ## Generate module imports and compile the local server binary.
	@go mod tidy
	CGO_ENABLED=0 go build -trimpath -a -o $(BIN)

generate: ## Refresh generated module import wiring.
	go generate

# Pass module-manager arguments after the target:
#   make module list
#   make module install all-official
MODULE_ARGS := $(filter-out module,$(MAKECMDGOALS))
module: ## Run the community module manager.
	@go run . module $(MODULE_ARGS)

ifneq ($(filter module,$(MAKECMDGOALS)),)
.PHONY: $(MODULE_ARGS)
$(MODULE_ARGS):
	@:
endif

validate: fmtcheck vet ## Run the standard Go formatting and vet checks.

test: generate js-lint lua-lint ## Run code generation, JavaScript/Lua linting, and Go tests.
	@go test -race ./...

coverage: ## Generate and open an HTML Go coverage report.
	@mkdir -p bin/covdatafiles && \
	go test ./... -coverprofile=bin/covdatafiles/cover.out && \
	go tool cover -html=bin/covdatafiles/cover.out && \
	rm -rf bin

fmt: ## Format all Go files.
	@go fmt ./...

fmtcheck: ## Fail if any Go file is not gofmt-formatted.
	@set -e; \
	unformatted=$$(gofmt -l $$(git ls-files '*.go')); \
	if [ -n "$$unformatted" ]; then \
		echo "Go files need formatting:"; \
		printf '%s\n' "$$unformatted"; \
		exit 1; \
	fi

vet: ## Run go vet with repo-specific composite literal settings.
	@go vet -composites=false ./...

mod: ## Refresh vendored modules, tidy go.mod, and verify dependencies.
	@go mod vendor
	@go mod tidy
	@go mod verify

js-lint: ## Run JSHint using npx when available, otherwise Docker.
	@if command -v npx >/dev/null 2>&1; then \
		$(JSHINT_BASE_CMD) && \
		$(JSHINT_WINDOWS_CMD); \
	elif command -v docker >/dev/null 2>&1; then \
		docker run --rm -v "$(PWD)":/app -w /app node:22 sh -lc "\
			$(JSHINT_BASE_CMD) && \
			$(JSHINT_WINDOWS_CMD)"; \
	else \
		echo "js-lint requires npx or docker" >&2; \
		exit 127; \
	fi

lua-lint: ## Run Luacheck using a local install when available, otherwise Docker.
	@if [ -z "$(strip $(LUA_LINT_PATHS))" ]; then \
		echo "lua-lint: no .lua files to check"; \
	elif command -v luacheck >/dev/null 2>&1; then \
		$(LUACHECK_CMD); \
	elif command -v docker >/dev/null 2>&1; then \
		docker run --rm -v "$(PWD)":/code -w /code $(LUACHECK_DOCKER_IMAGE) $(LUACHECK_CMD); \
	else \
		echo "lua-lint requires luacheck or docker" >&2; \
		exit 127; \
	fi

## Running Locally
.PHONY: run run-new clean-instances https-setup reset-admin-pw client

run: generate ## Start the server with `go run .`.
	@go run .

run-new: clean-instances generate run ## Delete room instance data and start a fresh world.

clean-instances: ## Delete generated room instance data for bundled worlds.
	rm -Rf _datafiles/world/default/rooms.instances
	rm -Rf _datafiles/world/empty/rooms.instances

https-setup: ## Run the interactive HTTPS certificate setup helper.
	@sh ./scripts/https-setup.sh

reset-admin-pw: ## Interactively reset the admin user's password.
	@go run ./cmd/reset-admin-pw

client: ## Open a telnet client connected to the Docker server.
	$(DOCKER_COMPOSE) run --rm terminal telnet go-mud-server 33333

## Docker And CI
.PHONY: docker_build run-docker console ci-local-image ci-local ci-local-inner clean

docker_build: ## Build the server image with compose.yml.
	GO_VERSION=$(GO_VERSION) TAG=$(VERSION) $(DOCKER_COMPOSE) build server

run-docker: ## Build and start the server container from compose.yml.
	GO_VERSION=$(GO_VERSION) $(DOCKER_COMPOSE) up --build --remove-orphans server

console: ## Open a shell in a Go toolchain container mounted on this repo.
	@docker run --rm -it --init \
		-v "$(PWD)":/src \
		-w /src \
		$(GO_CONSOLE_IMAGE) \
		$(DOCKER_CMD)

docker-%: ## Run a make target inside the Go toolchain container, for example `make docker-test`.
	@$(MAKE) console DOCKER_CMD="make $(patsubst docker-%,%,$@)"

ci-local-image: ## Build the local CI tool image used by `make ci-local`.
	docker build \
		--build-arg GO_VERSION=$(GO_VERSION) \
		-f .github/Dockerfile.act \
		-t $(CI_LOCAL_IMAGE) .

ci-local: ci-local-image ## Run local CI validation in the CI tool container.
	mkdir -p "$(CI_LOCAL_ACT_CACHE_DIR)"
	$(CI_LOCAL_RUN) make ci-local-inner

ci-local-inner: ## Run CI checks from inside the local CI tool container.
	actionlint .github/workflows/*.yml
	yamllint .github
	shellcheck .github/scripts/*.sh
	$(MAKE) validate
	$(MAKE) js-lint
	$(MAKE) lua-lint
	ACT_FLAGS="$(ACT_FLAGS)" \
		ACT_DRYRUN_SECRETS="$(ACT_DRYRUN_SECRETS)" \
		.github/scripts/ci-local-act.sh

clean: ## Stop compose services, remove their volumes, and prune Docker images.
	$(DOCKER_COMPOSE) down --volumes --remove-orphans
	docker system prune -a

## Cross Builds
.PHONY: build_rpi_zero2w build_win64 build_linux64

# For supported GOOS/GOARCH values, run: go tool dist list
build_rpi_zero2w: CROSS_GOOS := linux
build_rpi_zero2w: CROSS_GOARCH := arm64
build_rpi_zero2w: CROSS_OUTPUT := $(BIN)-rpi
build_rpi_zero2w: generate ## Build a Raspberry Pi Zero 2 W binary.

build_win64: CROSS_GOOS := windows
build_win64: CROSS_GOARCH := amd64
build_win64: CROSS_OUTPUT := $(BIN)-win64.exe
build_win64: generate ## Build a 64-bit Windows binary.

build_linux64: CROSS_GOOS := linux
build_linux64: CROSS_GOARCH := amd64
build_linux64: CROSS_OUTPUT := $(BIN)-linux64
build_linux64: generate ## Build a 64-bit Linux binary.

build_rpi_zero2w build_win64 build_linux64:
	$(CROSS_BUILD_CMD)

## Utility
.PHONY: go-version image_tag port shell cert-clean cert set_gopath view_pprof_mem help

go-version: ## Print the Go version pinned in go.mod.
	@printf '%s\n' "$(GO_VERSION)"

image_tag: ## Print the current Docker image tag value.
	@echo $(VERSION)

port: ## Print the host port mapped to server port 8080.
	@$(eval PORT := $(shell $(DOCKER_COMPOSE) port server 8080))
	@echo $(PORT)

shell: ## Open /bin/sh inside the running server container.
	@$(eval CONTAINER_NAME := $(shell docker ps --filter="name=mud" --format '{{.Names}}' ))
	docker exec -it $(CONTAINER_NAME) /bin/sh

cert-clean: ## Remove local development TLS certificate files.
	rm -f server.crt server.key

cert: server.crt server.key ## Generate local self-signed TLS certificate files.

server.crt server.key:
	openssl req -x509 -nodes -newkey rsa:4096 \
		-keyout server.key -out server.crt \
		-days 365 -subj "/CN=localhost"

set_gopath: ## Print a command for adding this repo to GOPATH in legacy shells.
ifeq ($(OS),Windows_NT)
	@echo 'PowerShell: $$env:GOPATH = "$$env:GOPATH;$(CURDIR)"'
else
	@printf 'export GOPATH="$${GOPATH:+$$GOPATH:}%s"\n' "$(CURDIR)"
endif

view_pprof_mem: ## Open the saved memory profile in the Go pprof web UI.
	go tool pprof -http=:8989 source/_datafiles/profiles/mem.pprof
