# Developing Infra

## Setup

Install tools:

```
brew install go
go get -u github.com/kevinburke/go-bindata/...
```

Clone the project:

```
git clone https://github.com/infrahq/infra
cd infra
```

Run locally:

```
go run .
```

## Generate docs

```
go run ./internal/docgen
```

## Test

Run tests:

```
go test ./...
```

## Release

```bash
# Install tools
brew install gh
go get https://github.com/mitchellh/gon

# Build sign and upload binaries
make release

# Build and push Docker images
make release/docker  # Build and push Docker images
```