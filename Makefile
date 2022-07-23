v ?= $(shell git describe --tags --abbrev=0)
IMAGEVERSION ?= $(v:v%=%)

test:
	go test -short ./...

test-all:
	go test ./...

# update the expected command output file
test/update:
	go test ./internal/cmd -test.update-golden

dev:
	docker buildx build . -t infrahq/infra:dev
	kubectl config use-context docker-desktop
	helm upgrade --install --wait  \
		--set global.image.pullPolicy=Never \
		--set global.image.tag=dev \
		--set global.podAnnotations.checksum=$$(docker images -q infrahq/infra:dev) \
		infra ./helm/charts/infra \
		$(flags)

dev/clean:
	kubectl config use-context docker-desktop
	helm uninstall infra || true

helm/lint:
	helm lint helm/charts/*

tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/goreleaser/goreleaser@latest

lint:
	golangci-lint run --fix

.PHONY: docs
docs: docs/api/openapi3.json
	go run ./internal/docgen

.PHONY: docs/api/openapi3.json
docs/api/openapi3.json:
	go run ./internal/openapigen $@
