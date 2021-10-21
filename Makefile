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
	helm package -d helm $< --version $(tag) --app-version $(tag)

helm/charts/infra/charts:
	mkdir -p $@

helm/charts/infra/charts/%.tgz: helm/%.tgz helm/charts/infra/charts/
	ln -sf $(realpath $<) $(@D)

helm/infra.tgz: helm/charts/infra/charts/engine-$(tag).tgz

.PHONY: docs
docs:
	go run ./internal/docgen

clean:
	$(RM) -r dist
	$(RM) -r helm/*.tgz helm/charts/infra/charts

.PHONY: openapi
openapi:
	@$(RM) -r ./internal/api/*.go
	@GO_POST_PROCESS_FILE="gofmt -s -w" openapi-generator generate -i ./openapi.yaml -g go -o ./internal/api --additional-properties packageName=api,isGoSubmodule=true --enable-post-process-file > /dev/null
	@$(RM) -r ./internal/api/api ./internal/api/.openapi-generator
	@$(RM) -r ./internal/registry/ui/api/apis/* ./internal/registry/ui/api/models/*
	@openapi-generator generate -i ./openapi.yaml -g typescript-fetch -o ./internal/registry/ui/api --additional-properties typescriptThreePlus=true > /dev/null
	@$(RM) -r ./internal/registry/ui/api/.openapi-generator

goreleaser:
	@command -v goreleaser >/dev/null || { echo "install goreleaser @ https://goreleaser.com/install/#install-the-pre-compiled-binary" && exit 1; }

.PHONY: build
build: goreleaser
	goreleaser build --snapshot --rm-dist

build/docker:
	docker build --build-arg TELEMETRY_WRITE_KEY=${TELEMETRY_WRITE_KEY} --build-arg CRASH_REPORTING_DSN=${CRASH_REPORTING_DSN} . -t infrahq/infra:0.0.0-development

dev:
	# docker desktop setup for the dev environment
	# create a token and get the token secret from:
	# https://dev-02708987-admin.okta.com/admin/access/api/tokens
	# get client secret from:
	# https://dev-02708987-admin.okta.com/admin/app/oidc_client/instance/0oapn0qwiQPiMIyR35d6/#tab-general
	# create the required secret with:
	# kubectl create secret generic infra-registry-okta -n infrahq --from-literal=clientSecret=$$OKTA_CLIENT_SECRET --from-literal=apiToken=$$OKTA_API_TOKEN

	kubectl config use-context docker-desktop
	make build/docker
	helm upgrade --install infra-registry ./helm/charts/registry --namespace infrahq --create-namespace --set image.pullPolicy=Never --set image.tag=0.0.0-development --set-file config=./infra.yaml --set logLevel=debug
	kubectl config set-context --current --namespace=infrahq
	kubectl wait --for=condition=available --timeout=600s deployment/infra-registry --namespace infrahq
	helm upgrade --install infra-engine ./helm/charts/engine --namespace infrahq --set image.pullPolicy=Never --set image.tag=0.0.0-development --set name=dd --set registry=infra-registry --set apiKey=$$(kubectl get secrets/infra-registry --template={{.data.engineApiKey}} --namespace infrahq | base64 -D) --set service.ports[0].port=8443 --set service.ports[0].name=https --set service.ports[0].targetPort=443 --set logLevel=debug
	kubectl rollout restart deployment/infra-registry --namespace infrahq
	kubectl rollout restart deployment/infra-engine --namespace infrahq
	ROOT_TOKEN=$$(kubectl --namespace infrahq get secrets infra-registry -o jsonpath='{.data.rootApiKey}' | base64 -D); \
    echo Root token is $$ROOT_TOKEN

dev/clean:
	kubectl config use-context docker-desktop
	helm uninstall --namespace infrahq infra-registry || true
	helm uninstall --namespace infrahq infra-engine || true

dev/helm: helm/charts/infra/charts
	ln -sf $(realpath helm/charts/engine helm/charts/registry) $<

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
