tag := $(shell git describe --tags)
repo := infrahq/infra
pwd = $(shell pwd)

generate:
	go generate ./...

test:
	go test ./...

clean:
	rm -rf build release

.PHONY: build
build:
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o build/infra-darwin-arm64 -ldflags="-s -w" .
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o build/infra-darwin-x86_64 -ldflags="-s -w" .

	make build/docker

	docker create --name temp infrahq/infra
	docker cp temp:/bin/infra ./build/infra-linux-arm64
	docker rm -f temp

	docker --context builder create --name temp infrahq/infra
	docker --context builder cp temp:/bin/infra ./build/infra-linux-amd64
	docker --context builder rm -f temp

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
	docker build . -t infrahq/infra -t infrahq/infra:$(tag:v%=%)
	docker --context builder build . -t infrahq/infra -t infrahq/infra:$(tag:v%=%)

release/docker:
	docker build . -t infrahq/infra -t infrahq/infra:$(tag:v%=%)
	docker push infrahq/infra
	docker push infrahq/infra:$(tag:v%=%)
	docker --context builder build . -t infrahq/infra  -t infrahq/infra:$(tag:v%=%)
	docker --context builder push infrahq/infra
	docker --context builder push infrahq/infra:$(tag:v%=%)
