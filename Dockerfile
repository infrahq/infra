FROM golang:1.16 AS builder
WORKDIR /go/src/github.com/infrahq/infra
COPY . .
RUN CGO_ENABLED=1 go build -ldflags "-linkmode external -extldflags -static" .

FROM alpine
COPY --from=builder /go/src/github.com/infrahq/infra/infra /bin/infra
EXPOSE 2378
ENTRYPOINT ["/bin/infra"]
CMD ["start"]
