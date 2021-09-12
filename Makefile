GOCMD=go
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
BINARY_NAME=webdl
VERSION?=1.0.0
GOOS?=$(GOCMD env GOOS)
SERVICE_PORT?=3000
DOCKER_REGISTRY?= #if set it should finished by /
EXPORT_RESULT?=false # for CI please set EXPORT_RESULT to true

GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)

.PHONY: all test build vendor

all: help


## Build:
build: clean vendor ## Build your project and put the output binary in out/bin/
	@mkdir -p out/bin

	@echo "Compiling for macOS amd64 and arm64..."
	@GOOS=darwin GOARCH=amd64 GO111MODULE=on $(GOCMD) build -mod vendor -o out/bin/$(BINARY_NAME)-darwin-amd64 ./cmd/webdl.go
	@GOOS=darwin GOARCH=arm64 GO111MODULE=on $(GOCMD) build -mod vendor -o out/bin/$(BINARY_NAME)-darwin-arm64 ./cmd/webdl.go

	@echo "Compiling for Windows, 386 and arm64..."
	@GOOS=windows GOARCH=386 GO111MODULE=on $(GOCMD) build -mod vendor -o out/bin/$(BINARY_NAME)-windows-386.exe ./cmd/webdl.go
	@GOOS=windows GOARCH=arm64 GO111MODULE=on $(GOCMD) build -mod vendor -o out/bin/$(BINARY_NAME)-windows-arm64.exe ./cmd/webdl.go

	@echo "Compiling for Linux 386, amd64, arm and arm64..."
	@GOOS=linux GOARCH=386 GO111MODULE=on $(GOCMD) build -mod vendor -o out/bin/$(BINARY_NAME)-linux-386 ./cmd/webdl.go
	@GOOS=linux GOARCH=amd64 GO111MODULE=on $(GOCMD) build -mod vendor -o out/bin/$(BINARY_NAME)-linux-amd64 ./cmd/webdl.go
	@GOOS=linux GOARCH=arm GO111MODULE=on $(GOCMD) build -mod vendor -o out/bin/$(BINARY_NAME)-linux-arm ./cmd/webdl.go
	@GOOS=linux GOARCH=arm64 GO111MODULE=on $(GOCMD) build -mod vendor -o out/bin/$(BINARY_NAME)-linux-arm64 ./cmd/webdl.go

	@echo "Compiling for FreeBSD 386, amd64 and arm64..."
	@GOOS=freebsd GOARCH=386 GO111MODULE=on $(GOCMD) build -mod vendor -o out/bin/$(BINARY_NAME)-freebsd-386 ./cmd/webdl.go
	@GOOS=freebsd GOARCH=amd64 GO111MODULE=on $(GOCMD) build -mod vendor -o out/bin/$(BINARY_NAME)-freebsd-amd64 ./cmd/webdl.go
	@GOOS=freebsd GOARCH=arm64 GO111MODULE=on $(GOCMD) build -mod vendor -o out/bin/$(BINARY_NAME)-freebsd-arm64 ./cmd/webdl.go

clean: ## Remove build related file
	rm -fr ./bin
	rm -fr ./out
	rm -fr ./vendor
	rm -f ./junit-report.xml checkstyle-report.xml ./coverage.xml ./profile.cov yamllint-checkstyle.xml

vendor: ## Copy of all packages needed to support builds and tests in the vendor directory
	@$(GOCMD) mod vendor

watch: ## Run the code with cosmtrek/air to have automatic reload on changes
	$(eval PACKAGE_NAME=$(shell head -n 1 go.mod | cut -d ' ' -f2))
	@docker run -it --rm -w /go/src/$(PACKAGE_NAME) -v $(shell pwd):/go/src/$(PACKAGE_NAME) -p $(SERVICE_PORT):$(SERVICE_PORT) cosmtrek/air

## Test:
test: ## Run the tests of the project
ifeq ($(EXPORT_RESULT), true)
	GO111MODULE=off go get -u github.com/jstemmer/go-junit-report
	$(eval OUTPUT_OPTIONS = | tee /dev/tty | go-junit-report -set-exit-code > junit-report.xml)
endif
	$(GOTEST) -v -race ./... $(OUTPUT_OPTIONS)

coverage: ## Run the tests of the project and export the coverage
	$(GOTEST) -cover -covermode=count -coverprofile=profile.cov ./...
	$(GOCMD) tool cover -func profile.cov
ifeq ($(EXPORT_RESULT), true)
	GO111MODULE=off go get -u github.com/AlekSi/gocov-xml
	GO111MODULE=off go get -u github.com/axw/gocov/gocov
	gocov convert profile.cov | gocov-xml > coverage.xml
endif

## Lint:
lint: lint-go lint-dockerfile lint-yaml ## Run all available linters

lint-dockerfile: ## Lint your Dockerfile
# If dockerfile is present we lint it.
ifeq ($(shell test -e ./Dockerfile && echo -n yes),yes)
	$(eval CONFIG_OPTION = $(shell [ -e $(shell pwd)/.hadolint.yaml ] && echo "-v $(shell pwd)/.hadolint.yaml:/root/.config/hadolint.yaml" || echo "" ))
	$(eval OUTPUT_OPTIONS = $(shell [ "${EXPORT_RESULT}" == "true" ] && echo "--format checkstyle" || echo "" ))
	$(eval OUTPUT_FILE = $(shell [ "${EXPORT_RESULT}" == "true" ] && echo "| tee /dev/tty > checkstyle-report.xml" || echo "" ))
	@docker run --rm -i $(CONFIG_OPTION) hadolint/hadolint hadolint $(OUTPUT_OPTIONS) - < ./Dockerfile $(OUTPUT_FILE)
endif

lint-go: ## Use golintci-lint on your project
	$(eval OUTPUT_OPTIONS = $(shell [ "${EXPORT_RESULT}" == "true" ] && echo "--out-format checkstyle ./... | tee /dev/tty > checkstyle-report.xml" || echo "" ))
	@docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:latest-alpine golangci-lint run --deadline=65s $(OUTPUT_OPTIONS)

lint-yaml: ## Use yamllint on the yaml file of your projects
ifeq ($(EXPORT_RESULT), true)
	GO111MODULE=off go get -u github.com/thomaspoignant/yamllint-checkstyle
	$(eval OUTPUT_OPTIONS = | tee /dev/tty | yamllint-checkstyle > yamllint-checkstyle.xml)
endif
	@docker run --rm -it -v $(shell pwd):/data cytopia/yamllint -f parsable $(shell git ls-files '*.yml' '*.yaml') $(OUTPUT_OPTIONS)

## Docker:
docker-build: ## Use the dockerfile to build the container
	docker build --rm --tag $(BINARY_NAME) .

docker-release: ## Release the container with tag latest and version
	docker tag $(BINARY_NAME) $(DOCKER_REGISTRY)$(BINARY_NAME):latest
	docker tag $(BINARY_NAME) $(DOCKER_REGISTRY)$(BINARY_NAME):$(VERSION)
	# Push the docker images
	docker push $(DOCKER_REGISTRY)$(BINARY_NAME):latest
	docker push $(DOCKER_REGISTRY)$(BINARY_NAME):$(VERSION)


## Help:
help: ## Show this help.
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    ${YELLOW}%-20s${GREEN}%s${RESET}\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  ${CYAN}%s${RESET}\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)
