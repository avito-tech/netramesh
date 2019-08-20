SHELL   := /bin/bash -euo pipefail
TIMEOUT := 1s
GOFLAGS := -mod=vendor
PKGS    := go list ./... | grep -v pkg/http
TARGET  := netramesh

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
	@$(PKGS) | xargs -I {} go test -race -timeout $(TIMEOUT) {}

.PHONY: test-with-coverage
test-with-coverage:
	@$(PKGS) | xargs -I {} sh -c "go test -cover -timeout $(TIMEOUT) {} | column -t | sort -r"

.PHONY: clean
clean:
	rm -rf bin/

.PHONY: build
build:
	mkdir -p bin/
	for target_os in "darwin" "linux"; do \
		GOOS=$$target_os go build -o ./bin/$(TARGET)_$$target_os ./cmd ;\
	done

.PHONY: docker-build
docker-build:
	@docker build -f Dockerfile \
	              -t netramesh:latest \
	              --force-rm --no-cache --pull --rm \
	              .
