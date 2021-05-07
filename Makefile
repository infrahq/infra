.PHONY: build
build:
	GOOS=darwin GOARCH=arm64 go build -o build/infra-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build -o build/infra-darwin-amd64 .
	GOOS=linux GOARCH=arm64 go build -o build/infra-linux-arm64 .
	GOOS=linux GOARCH=amd64 go build -o build/infra-linux-amd64 .
	GOOS=windows GOARCH=amd64 go build -o build/infra-windows-amd64 .

sign: build
	gon .gon.json > /dev/null
	unzip -o -d release release/infra-darwin-binaries.zip > /dev/null
	rm release/infra-darwin-binaries.zip > /dev/null

test:
	go test ./...

clean:
	rm -rf build release
