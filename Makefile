SHELL := /bin/bash -euo pipefail

.PHONY: deps
deps:
	@go mod tidy && go mod vendor && go mod verify

.PHONY: update
update:
	@go get -mod= -u


.PHONY: format
format:
	@goimports -ungroup -w .


.PHONY: test
test:
	@go test -race -timeout 1s ./...


.PHONY: build
build:
	@echo not implemented yet

.PHONY: docker-build
docker-build:
	@docker build -f Dockerfile \
	              -t netramesh:latest \
	              --force-rm --no-cache --pull --rm \
	              .
