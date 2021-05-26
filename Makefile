tag := $(or $(git describe --tags), v0.0.1)
repo := infrahq/infra

.PHONY: build
build:
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o build/infra-darwin-arm64 -ldflags="-s -w" .
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o build/infra-darwin-x86_64 -ldflags="-s -w" .

generate:
	go generate ./...

test:
	go test ./...

clean:
	rm -rf build release

sign:
	gon .gon.json
	unzip -o -d build build/infra-darwin-binaries.zip
	rm build/infra-darwin-binaries.zip

release:
	make build
	make sign
	-gh release create $(tag) --title $(tag) -n "" -R $(repo)
	gh release upload $(tag) build/* --clobber -R $(repo)

dev/docker:
	docker build . -t infrahq/infra:dev
	kubectl apply -f ./deploy/dev.yaml
	kubectl rollout restart -n infra statefulset/infra

build/docker:
	docker buildx build --platform linux/amd64,linux/arm64 . -t infrahq/infra:$(tag:v%=%)
	docker buildx build --platform linux/amd64,linux/arm64 . -t infrahq/infra

release/docker:
	docker buildx build --push --platform linux/amd64,linux/arm64 . -t infrahq/infra:$(tag:v%=%)
	docker buildx build --push --platform linux/amd64,linux/arm64 . -t infrahq/infra
