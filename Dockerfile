FROM golang:1.16-alpine AS builder
RUN apk add --no-cache gcc musl-dev
ARG TARGETARCH
WORKDIR /go/src/github.com/infrahq/infra
COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=$TARGETARCH CC=gcc go build .

FROM alpine
COPY --from=builder /go/src/github.com/infrahq/infra/infra /bin/infra
EXPOSE 2378
ENTRYPOINT ["/bin/infra"]
CMD ["server"]
