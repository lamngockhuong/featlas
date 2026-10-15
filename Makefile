VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
BIN := bin/muninn

.PHONY: web build test test-web e2e lint dev demo capture-e2e clean

## web: build the React app into internal/webui/static (embedded by go build)
web:
	pnpm -C web install --frozen-lockfile
	pnpm -C web run build

## build: web app, then the binary with the app embedded
build: web
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/muninn

## test: Go and web unit tests
test:
	go test ./...
	pnpm -C web test

## e2e: Playwright tests of the web app against a copy of examples/demo
e2e:
	pnpm -C web run e2e

## capture-e2e: compare capture backends on the demo app (needs Chrome; Node for Playwright)
capture-e2e:
	npm --prefix plugins/muninn/skills/muninn-gen/scripts install
	MUNINN_CAPTURE_E2E=1 go test -run TestBackendsAgree -v ./internal/capture/

## lint: go vet, golangci-lint and Biome
lint:
	go vet ./...
	golangci-lint run
	pnpm -C web exec biome check .

## dev: API on :7078 against examples/demo and Vite on :5174 with hot reload
dev:
	@trap 'kill 0' INT TERM EXIT; \
	go run -tags dev ./cmd/muninn serve --dir examples/demo/muninn --port 7078 & \
	pnpm -C web run dev

## demo: serve the demo content with the embedded app
demo: build
	$(BIN) serve --dir examples/demo/muninn --open

clean:
	rm -rf bin site
	find internal/webui/static -mindepth 1 ! -name .keep -delete
