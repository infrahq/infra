FROM --platform=$BUILDPLATFORM node:18 AS ui-builder
WORKDIR /ui
COPY ui/package.json .
RUN npm install
COPY ui .
RUN npm run build
RUN npm run export

FROM --platform=$BUILDPLATFORM golang:1.18 AS builder
RUN apt-get update && \
    apt-get install -y gcc-aarch64-linux-gnu gcc-x86-64-linux-gnu && \
    ln -s /usr/bin/aarch64-linux-gnu-gcc /usr/bin/arm64-linux-gnu-gcc  && \
    ln -s /usr/bin/x86_64-linux-gnu-gcc /usr/bin/amd64-linux-gnu-gcc

# 1. Precompile the entire go standard library into the first Docker cache layer: useful for other projects too!
RUN CGO_ENABLED=0 GOOS=linux go install -v -installsuffix cgo -a std

ARG TARGETARCH
# {x-release-please-start-version}
ARG BUILDVERSION=0.13.1
# {x-release-please-end}
ARG BUILDVERSION_PRERELEASE
ARG BUILDVERSION_METADATA
ARG TELEMETRY_WRITE_KEY
WORKDIR /go/src/github.com/infrahq/infra

# get deps first so it's cached
COPY go.mod .
COPY go.sum .
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    go mod download

COPY . .

# copy static ui files
COPY --from=ui-builder /ui/out /go/src/github.com/infrahq/infra/internal/server/ui/static

RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux GOARCH=$TARGETARCH \
    CC=$TARGETARCH-linux-gnu-gcc \
    go build \
    -ldflags '-s -X github.com/infrahq/infra/internal.Version='"$BUILDVERSION"' -X github.com/infrahq/infra/internal.Prerelease='"$BUILDVERSION_PRERELEASE"' -X github.com/infrahq/infra/internal.Metadata='"$BUILDVERSION_METADATA"' -X github.com/infrahq/infra/internal.TelemetryWriteKey='"$TELEMETRY_WRITE_KEY"' -linkmode external -extldflags "-static"' \
    .

FROM alpine
COPY --from=builder /go/src/github.com/infrahq/infra/infra /bin/infra
EXPOSE 80
EXPOSE 443
ENTRYPOINT ["/bin/infra"]
CMD ["server"]
