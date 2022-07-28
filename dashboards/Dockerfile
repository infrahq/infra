FROM --platform=$BUILDPLATFORM golang:1.18

RUN go install github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb@latest; \
    go install github.com/google/go-jsonnet/cmd/jsonnet@latest; \
    go install github.com/google/go-jsonnet/cmd/jsonnetfmt@latest

WORKDIR /mnt
COPY jsonnetfile.json .
COPY jsonnetfile.lock.json .
RUN jb install

ENTRYPOINT ["jsonnet", "-J", "vendor"]
