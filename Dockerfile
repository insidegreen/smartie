FROM golang:1.21 AS builder

COPY . /go/src/smartie
WORKDIR /go/src/smartie

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/smartie cmd/smartie/main.go

FROM alpine:latest as certs
RUN apk --no-cache add ca-certificates

FROM alpine:latest AS executor

COPY --from=builder /tmp/smartie /app/smartie
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

EXPOSE 2112
ENV SERVICE_PORT=2112

WORKDIR /app
CMD ["./smartie"]
