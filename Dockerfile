FROM golang:1.15.7 AS builder
ENV CGO_ENABLED=0
ENV GOOS=linux
WORKDIR /go/src/github.com/infrahq/infra
COPY . .
RUN go build .

FROM alpine:3.10
COPY --from=builder /go/src/github.com/infrahq/infra/infra /bin/infra
EXPOSE 3090
CMD ["/bin/infra", "server"]
