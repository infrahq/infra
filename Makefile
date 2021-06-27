tag := $(shell git describe --tags)
repo := infrahq/early-access

generate:
	go generate ./...

.PHONY: tools
tools:
	go generate -tags tools tools/tools.go

test:
	go test ./...

.PHONY: docs
docs:
	go run ./internal/docgen

clean:
	rm -rf build release
	docker rm temp

.PHONY: build
build:
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o build/infra-Darwin-arm64 -ldflags="-s -w" .
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o build/infra-Darwin-x86_64 -ldflags="-s -w" .

# TODO (jmorganca): find a better way to cross-compile linux & windows binaries
	docker buildx build --load --platform linux/amd64 . -t infrahq/infra:amd64
	docker create --platform linux/amd64 --name temp infrahq/infra:amd64 && docker cp temp:/bin/infra ./build/infra-Linux-amd64 && docker rm -f temp

	docker buildx build --load --platform linux/arm64 . -t infrahq/infra:arm64
	docker create --name temp infrahq/infra:arm64 && docker cp temp:/bin/infra ./build/infra-Linux-arm64 && docker rm -f temp

sign:
	gon .gon.json
	unzip -o -d build build/infra-darwin-binaries.zip
	rm build/infra-darwin-binaries.zip

release:
	make build
	make sign
	-gh release create $(tag) --title $(tag) -n "" -R $(repo)
	gh release upload $(tag) build/* --clobber -R $(repo)

release/docker:
	docker buildx build --push --platform linux/amd64,linux/arm64 . -t infrahq/infra
	docker buildx build --push --platform linux/amd64,linux/arm64 . -t infrahq/infra:$(tag:v%=%)

dev:
	kubectl config use-context docker-desktop
	docker build . -t infrahq/infra:dev
	helm upgrade --namespace infra --create-namespace --install infra ./helm/charts/registry --set image.pullPolicy=Never --set image.tag=dev
	kubectl rollout status -n infra deployment/infra
	kubectl wait -n infra --for=condition=available deployment/infra
	helm upgrade --namespace infra --create-namespace --install infra-engine ./helm/charts/engine --set image.pullPolicy=Never --set image.tag=dev --set apiKey=$$(kubectl exec -it -n infra deployment/infra -- infra apikey list | sed -n '2 p' | awk '{print $$2}') --set registry=infra --set name=default ;
	kubectl rollout restart -n infra deployment/infra-engine

make dev/clean:
	@helm uninstall --namespace infra infra || true
	@helm uninstall --namespace infra infra-engine || true
