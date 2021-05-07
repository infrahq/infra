FROM --platform=$BUILDPLATFORM golang:1.16-alpine AS builder
ARG TARGETARCH
WORKDIR /go/src/github.com/infrahq/infra
COPY . .
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH go build .

FROM alpine
COPY --from=builder /go/src/github.com/infrahq/infra/infra /bin/infra
EXPOSE 2378
ENTRYPOINT ["/bin/infra"]
CMD ["start"]
