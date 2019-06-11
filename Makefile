SHELL   := /bin/bash -euo pipefail
TIMEOUT := 1s
GOFLAGS := -mod=vendor


.PHONY: deps
deps:
	@go mod tidy && go mod vendor && go mod verify

.PHONY: update
update:
	@go get -d -mod= -u


.PHONY: format
format:
	@goimports -local golang_org,github.com/Lookyan/netramesh -ungroup -w ./cmd/ ./internal/ ./pkg/


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
