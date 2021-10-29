tag := $(patsubst v%,%,$(shell git describe --tags))

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
helm: helm/engine.tgz helm/infra.tgz
	helm repo index helm

helm/%.tgz: helm/charts/%
	$(RM) $(@D)/$*-*.tgz
	helm package -d $(@D) $< --version $(tag) --app-version $(tag)

helm/charts/infra/charts/:
	mkdir -p $@

helm/charts/infra/charts/%.tgz: helm/%.tgz helm/charts/infra/charts/
	$(RM) $(@D)/*.tgz
	ln -sf $(realpath $<) $(@D)

helm/infra.tgz: helm/charts/infra/charts/engine-$(tag).tgz

helm/lint: helm
	helm lint helm/charts/*

helm/clean:
	$(RM) -r helm/*.tgz helm/charts/infra/charts

.PHONY: docs
docs:
	go run ./internal/docgen

clean: helm/clean
	$(RM) -r dist

export GO_POST_PROCESS_FILE=bash openapi/go-post-process.sh

.PHONY: openapi
openapi: openapi/clean
	openapi-generator-cli generate --generator-key go-client
	openapi-generator-cli generate --generator-key typescript-client
	# manually post process some files since openapi-generator forgot
	find internal/api -name '*.go' -exec $(GO_POST_PROCESS_FILE) {} \;

openapi/clean:
	$(RM) -r internal/api/*.go
	$(RM) -r internal/api/api internal/api/.openapi-generator
	$(RM) -r internal/registry/ui/api/apis internal/registry/ui/api/models
	$(RM) -r internal/api/.openapi-generator

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

dev: $(VALUES) helm build/docker
	# docker desktop setup for the dev environment
	# create a token and get the token secret from:
	# https://dev-02708987-admin.okta.com/admin/access/api/tokens
	# get client secret from:
	# https://dev-02708987-admin.okta.com/admin/app/oidc_client/instance/0oapn0qwiQPiMIyR35d6/#tab-general
	# create the required secret with:
	# kubectl $(NS) create secret generic $(OKTA_SECRET) --from-literal=clientSecret=$$OKTA_CLIENT_SECRET --from-literal=apiToken=$$OKTA_API_TOKEN

	kubectl config use-context docker-desktop
	kubectl $(NS) get secrets $(INFRA_OKTA) >/dev/null
	helm $(NS) upgrade --install --create-namespace $(patsubst %,-f %,$(VALUES)) --wait infra helm/charts/infra
	@[ -z "$(NS)" ] || kubectl config set-context --current --namespace=$(NAMESPACE)
	@echo Root token is $$(kubectl $(NS) get secrets infra -o jsonpath='{.data.root-api-key}' | base64 --decode)

dev/clean:
	kubectl config use-context docker-desktop
	helm $(NS) uninstall infra || true
	helm $(NS) uninstall infra-engine || true

release: goreleaser
	goreleaser release -f .goreleaser.yml --rm-dist

release/docker:
	docker buildx build --push --platform linux/amd64,linux/arm64 --build-arg BUILDVERSION=$(tag) --build-arg TELEMETRY_WRITE_KEY=${TELEMETRY_WRITE_KEY} --build-arg CRASH_REPORTING_DSN=${CRASH_REPORTING_DSN} . -t infrahq/infra:$(tag) -t infrahq/infra

release/helm: helm
	aws s3 --region us-east-2 sync helm s3://helm.infrahq.com --exclude "*" --include "index.yaml" --include "*.tgz"

golangci-lint:
	@command -v golangci-lint >/dev/null || { echo "install golangci-lint @ https://golangci-lint.run/usage/install/#local-installation" && exit 1; }

lint: golangci-lint
	golangci-lint run ./...
