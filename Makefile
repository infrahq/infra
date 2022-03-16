tag := $(patsubst v%,%,$(shell git describe --tags))
version := $(tag:v%=%)

generate:
	go generate ./...

.PHONY: tools
tools:
	go generate -tags tools tools/tools.go

test:
	go test -short ./...

test-all:
	go test ./...

.PHONY: helm
helm:
	helm package -d $@ helm/charts/* --version $(version) --app-version $(version)
	helm repo index helm

helm/lint:
	helm lint helm/charts/*

helm/clean:
	$(RM) -r helm/*.tgz

.PHONY: docs
docs: openapi
	go run ./internal/docgen

clean: helm/clean
	$(RM) -r dist

goreleaser:
	@command -v goreleaser >/dev/null || { echo "install goreleaser @ https://goreleaser.com/install/#install-the-pre-compiled-binary" && exit 1; }

.PHONY: build
build: goreleaser
	goreleaser build --snapshot --rm-dist

export IMAGE_TAG=0.0.0-development

build/docker:
	docker build --build-arg TELEMETRY_WRITE_KEY=${TELEMETRY_WRITE_KEY} --build-arg CRASH_REPORTING_DSN=${CRASH_REPORTING_DSN} . -t infrahq/infra:$(IMAGE_TAG)

export OKTA_SECRET=infra-okta

%.yaml: %.yaml.in
	envsubst <$< >$@

docker-desktop.yaml: docker-desktop.yaml.in

NS = $(patsubst %,-n %,$(NAMESPACE))
VALUES ?= docker-desktop.yaml

dev: $(VALUES) build/docker
	# docker desktop setup for the dev environment
	# create a token and get the token secret from:
	# https://dev-02708987-admin.okta.com/admin/access/api/tokens
	# get client secret from:
	# https://dev-02708987-admin.okta.com/admin/app/oidc_client/instance/0oapn0qwiQPiMIyR35d6/#tab-general
	# create the required secret with:
	# kubectl $(NS) create secret generic $(OKTA_SECRET) --from-literal=clientSecret=$$OKTA_CLIENT_SECRET

	kubectl config use-context docker-desktop
	kubectl $(NS) get secrets $(INFRA_OKTA) >/dev/null
	helm $(NS) upgrade --install --create-namespace $(patsubst %,-f %,$(VALUES)) --wait infra helm/charts/infra
	@[ -z "$(NS)" ] || kubectl config set-context --current --namespace=$(NAMESPACE)

dev/clean:
	kubectl config use-context docker-desktop
	helm $(NS) uninstall infra || true

docker:
	docker buildx build --push --platform linux/amd64,linux/arm64 --build-arg BUILDVERSION=$(version) --build-arg TELEMETRY_WRITE_KEY=${TELEMETRY_WRITE_KEY} --build-arg CRASH_REPORTING_DSN=${CRASH_REPORTING_DSN} . -t infrahq/infra:$(version)

release: goreleaser
	goreleaser release -f .goreleaser.yml --rm-dist

release/docker:
	docker buildx build --push --platform linux/amd64,linux/arm64 --build-arg BUILDVERSION=$(version) --build-arg TELEMETRY_WRITE_KEY=${TELEMETRY_WRITE_KEY} --build-arg CRASH_REPORTING_DSN=${CRASH_REPORTING_DSN} . -t infrahq/infra:$(version) -t infrahq/infra

release/helm: helm
	aws s3 --region us-east-2 sync helm s3://helm.infrahq.com --exclude "*" --include "index.yaml" --include "*.tgz"

golangci-lint:
	@command -v golangci-lint >/dev/null || { echo "install golangci-lint @ https://golangci-lint.run/usage/install/#local-installation" && exit 1; }

lint: golangci-lint
	golangci-lint run ./...

openapi-lint: docs/api/openapi3.json
	@command -v openapi --version >/dev/null || { echo "openapi missing, try: npm install -g @redocly/openapi-cli" && exit 1; }
	openapi lint $<

docs/api/openapi3.json:
	go run . openapi
