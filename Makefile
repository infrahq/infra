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
	# Remove the $$HOME of the local system from the docs
	# This probably only works on MacOS
	grep -ilr $$HOME docs/* | xargs -I@ sed -i '' 's:'"$$HOME"':$$HOME:g' @

clean:
	rm -rf dist

openapi:
	@rm -rf ./internal/api/*.go
	@GO_POST_PROCESS_FILE="gofmt -s -w" openapi-generator generate -i ./openapi.yaml -g go -o ./internal/api --additional-properties packageName=api,isGoSubmodule=true --enable-post-process-file > /dev/null
	@rm -rf ./internal/api/api ./internal/api/.openapi-generator
	@rm -rf ./internal/registry/ui/api/apis/* ./internal/registry/ui/api/models/*
	@openapi-generator generate -i ./openapi.yaml -g typescript-fetch -o ./internal/registry/ui/api --additional-properties typescriptThreePlus=true > /dev/null
	@rm -rf ./internal/registry/ui/api/.openapi-generator

.PHONY: build
build:
	goreleaser build --snapshot --rm-dist

dev:
	kubectl config use-context docker-desktop
	docker build . -t infrahq/infra:0.0.0-development
	helm upgrade --install infra-registry ./helm/charts/registry --namespace infrahq --create-namespace --set image.pullPolicy=Never --set image.tag=0.0.0-development
	kubectl wait --for=condition=available --timeout=600s deployment/infra-registry --namespace infrahq
	helm upgrade --install infra-engine ./helm/charts/engine --namespace infrahq --set image.pullPolicy=Never --set image.tag=0.0.0-development --set name=dd --set endpoint=kubernetes.docker.internal:6443 --set registry=infra-registry --set apiKey=$$(kubectl get secrets/infra-registry --template={{.data.defaultApiKey}} --namespace infrahq | base64 -D)
	kubectl rollout restart deployment/infra-registry --namespace infrahq
	kubectl rollout restart deployment/infra-engine --namespace infrahq

dev/clean:
	kubectl config use-context docker-desktop
	helm uninstall --namespace infrahq infra-registry || true
	helm uninstall --namespace infrahq infra-engine || true

release:
	goreleaser release -f .goreleaser.yml --rm-dist

release/docker:
	docker buildx build --push --platform linux/amd64,linux/arm64 --build-arg BUILDVERSION=$(tag:v%=%) . -t infrahq/infra:$(tag:v%=%) -t infrahq/infra

release/helm:
	make helm
	aws s3 --region us-east-2 sync helm s3://helm.infrahq.com --exclude "*" --include "index.yaml" --include "*.tgz"

