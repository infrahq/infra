.PHONY: build
build:
	GOOS=darwin GOARCH=arm64 go build -o build/infra-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build -o build/infra-darwin-amd64 .
	GOOS=linux GOARCH=arm64 go build -o build/infra-linux-arm64 .
	GOOS=linux GOARCH=amd64 go build -o build/infra-linux-amd64 .
	GOOS=windows GOARCH=amd64 go build -o build/infra-windows-amd64 .

sign:
	gon .gon.json
	unzip -o -d build build/infra-darwin-binaries.zip
	rm build/infra-darwin-binaries.zip

release:
	make build
	make sign
	gh release upload v0.0.1 build/* --clobber

test:
	go test ./...

clean:
	rm -rf build release
