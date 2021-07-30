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
	helm package -d ./helm helm/charts/infra helm/charts/engine --version $(tag:v%=%) --app-version $(tag:v%=%)
	helm repo index ./helm

.PHONY: docs
docs:
	go run ./internal/docgen
	# Remove the $$HOME of the local system from the docs
	# This probably only works on MacOS
	grep -ilr $$HOME docs/* | xargs -I@ sed -i '' 's:'"$$HOME"':$$HOME:g' @

clean:
	rm -rf dist

proto:
	@protoc \
		--proto_path=./internal/v1 \
		--validate_out="lang=go:./internal/v1" --validate_opt paths=source_relative \
		--go_out ./internal/v1 --go_opt paths=source_relative \
		--go-grpc_out ./internal/v1 --go-grpc_opt paths=source_relative \
		--grpc-gateway_out ./internal/v1 --grpc-gateway_opt logtostderr=true --grpc-gateway_opt paths=source_relative --grpc-gateway_opt generate_unbound_methods=true \
		--grpc-gateway-ts_out=./internal/registry/ui/gen \
		./internal/v1/*.proto

.PHONY: build
build:
	goreleaser build --snapshot --rm-dist

dev:
	kubectl config use-context docker-desktop
	docker build . -t infrahq/infra:0.0.0-development
	helm upgrade --install infra ./helm/charts/infra --set image.pullPolicy=Never --set image.tag=0.0.0-development
	helm upgrade --install infra-engine ./helm/charts/engine --set image.pullPolicy=Never --set image.tag=0.0.0-development --set registry=infra --set apiKey=$(kubectl get secrets/infra --template={{.data.defaultApiKey}} | base64 -D)
	kubectl rollout restart deployment/infra
	kubectl rollout restart deployment/infra-engine

dev/clean:
	helm uninstall --namespace default infra || true

release:
	goreleaser release -f .goreleaser.yml --rm-dist

release/docker:
	docker buildx build --push --platform linux/amd64,linux/arm64 --build-arg BUILDVERSION=$(tag:v%=%) . -t infrahq/infra:$(tag:v%=%) -t infrahq/infra

release/helm:
	make helm
	aws s3 --region us-east-2 sync helm s3://helm.infrahq.com --exclude "*" --include "index.yaml" --include "*.tgz"

