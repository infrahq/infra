FROM golang:1.16 AS builder
WORKDIR /go/src/github.com/infrahq/infra
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build .

FROM golang:1.16
COPY --from=builder /go/src/github.com/infrahq/infra/infra /bin/infra
EXPOSE 3001
ENTRYPOINT ["/bin/infra"]
CMD ["start"]
