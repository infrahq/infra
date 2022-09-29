test:
	go test -short ./...

test-all:
	go test ./...

# update the expected command output file
test/update:
	go test ./internal/cmd -test.update-golden

dev:
	docker buildx build . --load -t infrahq/infra:dev
	docker buildx build ui --load -t infrahq/ui:dev
	kubectl config use-context docker-desktop
	helm upgrade --install --wait  \
		--set-string global.image.pullPolicy=Never \
		--set-string global.image.tag=dev \
		--set-string server.podAnnotations.checksum=$$(docker images -q infrahq/infra:dev) \
		--set-string connector.podAnnotations.checksum=$$(docker images -q infrahq/infra:dev) \
		--set-string ui.podAnnotations.checksum=$$(docker images -q infrahq/ui:dev) \
		infra ./helm/charts/infra \
		$(flags)

dev/clean:
	kubectl config use-context docker-desktop
	helm uninstall infra || true

helm/lint:
	helm lint helm/charts/*

tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/goreleaser/goreleaser@latest

postgres:
	docker run -d --name=postgres-dev --rm \
		-e POSTGRES_PASSWORD=password123 \
		--tmpfs=/var/lib/postgresql/data \
		-p 5433:5432 \
		postgres:14-alpine -c fsync=off -c full_page_writes=off
	@echo
	@echo Copy the line below into the shell used to run tests
	@echo 'export POSTGRESQL_CONNECTION="host=localhost port=5432 user=postgres dbname=postgres password=password123"'


LINT_ARGS ?= --fix

lint:
	(cd ./internal/tools/querylinter/cmd; go build -o ./querylinter.so -buildmode=plugin .)
	golangci-lint run $(LINT_ARGS)

.PHONY: docs
docs: docs/api/openapi3.json
	go run ./internal/docgen

.PHONY: docs/api/openapi3.json
docs/api/openapi3.json:
	go run ./internal/openapigen $@
