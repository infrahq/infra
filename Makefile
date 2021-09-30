tag := $(shell git describe --tags)

generate:
	go generate ./...

.PHONY: tools
tools:
	go generate -tags tools tools/tools.go

test:
	go test ./...

.PHONY: helm
helm:
	helm package -d ./helm helm/charts/registry helm/charts/engine --version $(tag:v%=%) --app-version $(tag:v%=%)
	helm repo index ./helm

.PHONY: docs
docs:
	go run ./internal/docgen

clean:
	$(RM) -r dist

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

dev:
	kubectl config use-context docker-desktop
	docker build . -t infrahq/infra:0.0.0-development
	helm upgrade --install infra-registry ./helm/charts/registry --namespace infrahq --create-namespace --set image.pullPolicy=Never --set image.tag=0.0.0-development
	kubectl wait --for=condition=available --timeout=600s deployment/infra-registry --namespace infrahq
	helm upgrade --install infra-engine ./helm/charts/engine --namespace infrahq --set image.pullPolicy=Never --set image.tag=0.0.0-development --set name=dd --set registry=infra-registry --set apiKey=$$(kubectl get secrets/infra-registry --template={{.data.defaultApiKey}} --namespace infrahq | base64 -D) --set service.ports[0].port=8443 --set service.ports[0].name=https --set service.ports[0].targetPort=443
	kubectl rollout restart deployment/infra-registry --namespace infrahq
	kubectl rollout restart deployment/infra-engine --namespace infrahq

dev/clean:
	kubectl config use-context docker-desktop
	helm uninstall --namespace infrahq infra-registry || true
	helm uninstall --namespace infrahq infra-engine || true

release: goreleaser
	goreleaser release -f .goreleaser.yml --rm-dist

release/docker:
	docker buildx build --push --platform linux/amd64,linux/arm64 --build-arg BUILDVERSION=$(tag:v%=%) . -t infrahq/infra:$(tag:v%=%) -t infrahq/infra

release/helm: helm
	aws s3 --region us-east-2 sync helm s3://helm.infrahq.com --exclude "*" --include "index.yaml" --include "*.tgz"

golangci-lint:
	@command -v golangci-lint >/dev/null || { echo "install golangci-lint @ https://golangci-lint.run/usage/install/#local-installation" && exit 1; }

lint: golangci-lint
	golangci-lint run ./...
