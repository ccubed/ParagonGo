

.DEFAULT_GOAL := build

VERSION ?= $(shell git rev-parse HEAD)
BIN ?= go-mud-server
DOCKER_COMPOSE := docker-compose -f compose.yml
GO_VERSION ?= $(shell awk '/^toolchain go/ { sub(/^toolchain go/, ""); print; found=1; exit } /^go / && !gover { gover=$$2 } END { if (!found && gover) print gover }' go.mod)
GO_CONSOLE_IMAGE ?= golang:$(GO_VERSION)-bookworm
CI_LOCAL_IMAGE ?= gomud-ci-local
CI_LOCAL_UID ?= $(shell id -u)
CI_LOCAL_GID ?= $(shell id -g)
CI_LOCAL_DOCKER_SOCK_GID ?= $(shell stat -c '%g' /var/run/docker.sock 2>/dev/null || id -g)
CI_LOCAL_HOME ?= /home/gomud
CI_LOCAL_ACT_CACHE_DIR ?= $(PWD)/.git/.cache/act
JSHINT_VERSION ?= 2.13.6
JS_LINT_PATHS := $(shell find _datafiles -name '*.js' -print)
WEBCLIENT_WINDOW_JS := $(shell find _datafiles/html/public/static/js/windows -name '*.js' -print)
WEBCLIENT_BASE_JS := $(filter-out $(WEBCLIENT_WINDOW_JS),$(JS_LINT_PATHS))
CI_LOCAL_RUN := docker run --rm \
	--user "$(CI_LOCAL_UID):$(CI_LOCAL_GID)" \
	--group-add "$(CI_LOCAL_DOCKER_SOCK_GID)" \
	-e HOME="$(CI_LOCAL_HOME)" \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-v "$(PWD)":/work \
	-v "$(CI_LOCAL_ACT_CACHE_DIR)":"$(CI_LOCAL_HOME)/.cache/act" \
	-w /work \
	$(CI_LOCAL_IMAGE)
ACT_FLAGS ?= --pull=false -P ubuntu-24.04=catthehacker/ubuntu:act-latest
ACT_DRYRUN_SECRETS ?= -s DISCORD_WEBHOOK_URL=https://example.invalid/webhook

export GOFLAGS := -mod=mod

## Build Targets

.PHONY: go-version
go-version: ### Print the Go version pinned in go.mod.
	@printf '%s\n' "$(GO_VERSION)"

.PHONY: docker_build 
docker_build: 
	GO_VERSION=$(GO_VERSION) TAG=$(VERSION) $(DOCKER_COMPOSE) build server

.PHONY: ci-local-image
ci-local-image: ### Build the local CI tool image.
	docker build \
		--build-arg GO_VERSION=$(GO_VERSION) \
		-f .github/Dockerfile.act \
		-t $(CI_LOCAL_IMAGE) .

.PHONY: ci-local
ci-local: ci-local-image ### Run local CI validation in a container.
	mkdir -p "$(CI_LOCAL_ACT_CACHE_DIR)"
	$(CI_LOCAL_RUN) make ci-local-inner

.PHONY: ci-local-inner
ci-local-inner: ### Run the local CI checks inside the CI tool image.
	actionlint .github/workflows/*.yml
	yamllint .github
	$(MAKE) validate
	$(MAKE) js-lint
	act $(ACT_FLAGS) --dryrun push \
		-e .github/act/push_master.json \
		-W .github/workflows/lint.yml
	act $(ACT_FLAGS) --dryrun pull_request \
		-e .github/act/pull_request.json \
		-W .github/workflows/lint.yml
	act $(ACT_FLAGS) --dryrun pull_request \
		-e .github/act/pull_request.json \
		-W .github/workflows/run-test.yml
	act $(ACT_FLAGS) --dryrun pull_request $(ACT_DRYRUN_SECRETS) \
		-e .github/act/pull_request.json \
		-W .github/workflows/discord-notify.yml
	act $(ACT_FLAGS) --dryrun push \
		-e .github/act/push_master.json \
		-W .github/workflows/build-and-release.yml
	act $(ACT_FLAGS) --dryrun push $(ACT_DRYRUN_SECRETS) \
		-e .github/act/push_master.json \
		-W .github/workflows/docker-package.yml
	act $(ACT_FLAGS) --dryrun pull_request $(ACT_DRYRUN_SECRETS) \
		-e .github/act/pull_request.json \
		-W .github/workflows/docker-package.yml

DOCKER_CMD ?= bash

.PHONY: console
console: ### Open a shell in the project Go toolchain container.
	@docker run --rm -it --init \
				-v "$(PWD)":/src \
				-w /src \
				$(GO_CONSOLE_IMAGE) \
				$(DOCKER_CMD)

docker-%:
	@$(MAKE) console DOCKER_CMD="make $(patsubst docker-%,%,$@)"

#
#
# For a complete list of GOOS/GOARCH combinations:
# Run: go tool dist list
#
#

.PHONY: build_rpi_zero2w
build_rpi_zero2w: generate ### Build a binary for a raspberry pi zero 2w
	env GOOS=linux GOARCH=arm64 go build -o $(BIN)-rpi

.PHONY: build_win64
build_win64: generate ### Build a binary for 64bit windows
	env GOOS=windows GOARCH=amd64 go build -o $(BIN)-win64.exe

.PHONY: build_linux64
build_linux64: generate ### Build a binary for linux
	env GOOS=linux GOARCH=amd64 go build -o $(BIN)-linux64

.PHONY: build
build: validate build_local  ### Validate the code and build the binary.

.PHONY: build_local
build_local: generate
	@go mod tidy
	CGO_ENABLED=0 go build -trimpath -a -o $(BIN)

.PHONY: generate
generate: ### Generates include directives for modules
	go generate

.PHONY: module
module: ### Manage community modules (usage: go run . module <subcommand>)
	@go run . module $(filter-out module,$(MAKECMDGOALS))


# Clean both development and production containers
.PHONY: clean
clean:
	$(DOCKER_COMPOSE) down --volumes --remove-orphans
	docker system prune -a

.PHONY: clean-instances
clean-instances: ### Deletes all room instance data. Starts the world fresh.
	rm -Rf _datafiles/world/default/rooms.instances
	rm -Rf _datafiles/world/empty/rooms.instances

## Run Targets

.PHONY: run 
run: generate  ### Build and run server.
	@go run .

.PHONY: run-new
run-new: clean-instances generate run ### Deletes instance data and runs server

.PHONY: run-docker
run-docker: ### Build and run server in docker.
	GO_VERSION=$(GO_VERSION) $(DOCKER_COMPOSE) up --build --remove-orphans server

.PHONY: https-setup
https-setup: ### Interactive HTTPS certificate setup helper.
	@sh ./scripts/https-setup.sh

.PHONY: reset-admin-pw
reset-admin-pw: ### Interactively reset the admin user's password.
	@go run ./cmd/reset-admin-pw


.PHONY: client
client: ### Build and run client terminal client
	$(DOCKER_COMPOSE) run --rm terminal telnet go-mud-server 33333

.PHONY: image_tag
image_tag:
	@echo $(VERSION)

.PHONY: port
port:
	@$(eval PORT := $(shell $(DOCKER_COMPOSE) port server 8080))
	@echo $(PORT)

.PHONY: shell
shell:
	@$(eval CONTAINER_NAME := $(shell docker ps --filter="name=mud" --format '{{.Names}}' ))
	docker exec -it $(CONTAINER_NAME) /bin/sh 

#
#
# Local code run/test
#
#
.PHONY: validate
validate: fmtcheck vet

.PHONY: test
test: generate js-lint ### Run code generation, lint, and Go tests.
	@go test -race ./...

.PHONY: coverage
coverage: 
	@mkdir -p bin/covdatafiles && \
	go test ./... -coverprofile=bin/covdatafiles/cover.out && \
	go tool cover -html=bin/covdatafiles/cover.out && \
	rm -rf bin

.PHONY: js-lint
js-lint:  ### Run Javascript linter
#   Grep filtering it to remove errors reported by docker image around npm packages
#   if "### errors" is found in the output, exits with an error code of 1
#   This should allow us to use it in CI/CD.
	@docker run --rm -v "$(PWD)":/app -w /app node:22 sh -lc "\
	  npx --yes jshint@$(JSHINT_VERSION) $(WEBCLIENT_BASE_JS) && \
	  npx --yes jshint@$(JSHINT_VERSION) --config .jshintrc.webclient-windows $(WEBCLIENT_WINDOW_JS)" \
	 2>&1 | grep -v "^npm " | tee /dev/stderr | grep -Eq "^[0-9]+ errors" && exit 1 || true

#
#
# Cert generation for testing
#
#
.PHONY: cert-clean
cert-clean:
	rm -f server.crt server.key

.PHONY: cert
cert: server.crt server.key

# This rule generates both files in one go using OpenSSL.
server.crt server.key:
	openssl req -x509 -nodes -newkey rsa:4096 \
		-keyout server.key -out server.crt \
		-days 365 -subj "/CN=localhost"



# Go targets

.PHONY: fmt
fmt:
	@go fmt ./...

.PHONY: fmtcheck
fmtcheck: fmt
	@set -e; \
	unformatted=$$(go fmt ./...); \
	if [ ! -z "$$unformatted" ]; then \
		echo Fixed inconsistent format in some files.; \
		echo $$unformatted; \
		exit 1; \
	fi

.PHONY: mod
mod:
	@go mod vendor
	@go mod tidy
	@go mod verify


.PHONY: vet
vet:
	@go vet -composites=false ./...           

.PHONY: set_gopath
set_gopath:
ifeq ($(OS),Windows_NT)
	setx PATH "$(PATH);mytest" -m
else
	export GOPATH=$GOPATH:$(pwd)
endif

.PHONY: view_pprof_mem
view_pprof_mem:
	go tool pprof -http=:8989 source/_datafiles/profiles/mem.pprof

#
# Help target - greps and formats special comments to form a "help" command for makefiles
#
## Help
.PHONY: help
help:                 ### List makefile targets.
# 1. grep for any lines starting with "##" or containing "\s###\s"
# 2. Align targets/comments with string padding
# 3. Wrap lines starting with "##" in ANSI escape codes (color) as headers
# 4. Wrap lines containing "\s###\s" in ANSI escape codes (color) as commands
# 5. Add new lines before any that aren't prefixed with space (Headers)
	@grep -hE "^##\s|\s###\s" $(MAKEFILE_LIST) \
		| awk -F'[[:space:]]###[[:space:]]' '{printf "%-36s### %s\n", substr($$1,1,35), $$2}' \
		| sed -E "s/^## ([^#]*)#*/`printf "\033[90;3m"`\1`printf "\033[0m"`/" \
		| sed "s/\(.*\):\(.*\)###\(.*\)$$/  `printf "\033[93m"`\1:`printf "\033[36m"`\2`printf "\033[97m"`-\3`printf "\033[0m"`/" \
    	| sed "/^[^[:blank:]]/{x;p;x;}"
	@printf "\n"
