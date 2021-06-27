# Developing Infra

## Setup

Install Go:

```
brew install go
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
make docs
```

## Test

Run tests:

```
make test
```

## Release

```bash
# Install tools
brew install gh

# Build, sign and upload binaries
make release

# Build and push Docker images
make release/docker  # Build and push Docker images
```
