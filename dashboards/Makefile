JSONNET:=$(wildcard *.jsonnet)
DASHBOARDS:=$(JSONNET:.jsonnet=.json)

all: $(DASHBOARDS)

%.json: %.jsonnet docker
	$Q docker run -i jsonnet-bundler - <$< >$@

.PHONY: docker
docker:
	$Q docker build -f Dockerfile -t jsonnet-bundler .
