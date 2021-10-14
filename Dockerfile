FROM --platform=$BUILDPLATFORM golang:1.16 AS builder
RUN apt-get update && \
    apt-get install -y gcc-aarch64-linux-gnu gcc-x86-64-linux-gnu && \
    ln -s /usr/bin/aarch64-linux-gnu-gcc /usr/bin/arm64-linux-gnu-gcc  && \
    ln -s /usr/bin/x86_64-linux-gnu-gcc /usr/bin/amd64-linux-gnu-gcc
ARG TARGETARCH
ARG BUILDVERSION=0.0.0-development
ARG TELEMETRY_WRITE_KEY
ARG CRASH_REPORTING_DSN
WORKDIR /go/src/github.com/infrahq/infra
COPY . .
VOLUME ["/root/.cache/go-build", "/go/pkg/mod"]
RUN CGO_ENABLED=1 GOOS=linux GOARCH=$TARGETARCH CC=$TARGETARCH-linux-gnu-gcc go build -ldflags '-s -X github.com/infrahq/infra/internal.Version='"$BUILDVERSION"' -X github.com/infrahq/infra/internal.TelemetryWriteKey='"$TELEMETRY_WRITE_KEY"' -X github.com/infrahq/infra/internal.CrashReportingDSN='"$CRASH_REPORTING_DSN"' -linkmode external -extldflags "-static"' .

FROM alpine
COPY --from=builder /go/src/github.com/infrahq/infra/infra /bin/infra
EXPOSE 80
EXPOSE 443
ENTRYPOINT ["/bin/infra"]
CMD ["registry"]
