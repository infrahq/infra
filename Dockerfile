FROM golang:1.16 AS builder
WORKDIR /go/src/github.com/infrahq/infra
COPY . .
RUN go build .

FROM alpine:3.10
COPY --from=builder /go/src/github.com/infrahq/infra/infra /bin/infra
EXPOSE 3001
CMD ["/bin/infra", "server"]
