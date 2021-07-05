FROM --platform=$BUILDPLATFORM golang:1.16 AS builder
RUN apt-get update && \
    apt-get install -y gcc-aarch64-linux-gnu gcc-x86-64-linux-gnu && \
    ln -s /usr/bin/aarch64-linux-gnu-gcc /usr/bin/arm64-linux-gnu-gcc  && \
    ln -s /usr/bin/x86_64-linux-gnu-gcc /usr/bin/amd64-linux-gnu-gcc
ARG TARGETARCH
WORKDIR /go/src/github.com/infrahq/infra
COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=$TARGETARCH CC=$TARGETARCH-linux-gnu-gcc go build -ldflags '-linkmode external -w -extldflags "-static"' .

FROM alpine
COPY --from=builder /go/src/github.com/infrahq/infra/infra /bin/infra
EXPOSE 2378
ENTRYPOINT ["/bin/infra"]
CMD ["registry"]
