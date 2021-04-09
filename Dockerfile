FROM golang:1.16 AS builder
WORKDIR /go/src/github.com/infrahq/infra
COPY . .
RUN go build .

FROM envoyproxy/envoy-alpine:v1.17-latest as envoy

FROM alpine:3.10
RUN wget -q -O /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub && wget https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.33-r0/glibc-2.33-r0.apk && apk add glibc-2.33-r0.apk
COPY --from=builder /go/src/github.com/infrahq/infra/infra /bin/infra
COPY --from=envoy /usr/local/bin/envoy /bin/envoy
EXPOSE 3090
CMD ["/bin/infra", "server"]

