v ?= $(shell git describe --tags --abbrev=0)
IMAGEVERSION ?= $(v:v%=%)

test:
	go test -short ./...

test-all:
	go test ./...

# update the expected command output file
test/update:
	go test ./internal/cmd -test.update-golden

.PHONY: helm
helm:
	helm package -d helm helm/charts/* --app-version $(IMAGEVERSION)
	helm repo index helm

helm/lint:
	helm lint helm/charts/*

helm/clean:
	rm -r helm/*.tgz

dev:
	docker build . -t infrahq/infra:dev
	kubectl config use-context docker-desktop
	helm upgrade --install --wait infra ./helm/charts/infra --set global.image.pullPolicy=Never --set global.image.tag=dev $(flags)
	kubectl rollout restart deployment/infra-server || true
	kubectl rollout restart deployment/infra-connector || true

dev/clean:
	kubectl config use-context docker-desktop
	helm uninstall infra || true

docker:
	docker buildx build --push \
		--platform linux/amd64,linux/arm64 \
		--build-arg BUILDVERSION_PRERELEASE=$(BUILDVERSION_PRERELEASE) \
		--build-arg TELEMETRY_WRITE_KEY=$(TELEMETRY_WRITE_KEY) \
		--tag infrahq/infra:$(IMAGEVERSION) \
		.

release:
	goreleaser release -f .goreleaser.yml --rm-dist

release/docker:
	docker buildx build --push \
		--platform linux/amd64,linux/arm64 \
		--build-arg TELEMETRY_WRITE_KEY=$(TELEMETRY_WRITE_KEY) \
		--tag infrahq/infra:$(IMAGEVERSION) \
		--tag infrahq/infra \
		.

release/helm: helm
	aws s3 --region us-east-2 sync helm s3://helm.infrahq.com --exclude "*" --include "index.yaml" --include "*.tgz"

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
