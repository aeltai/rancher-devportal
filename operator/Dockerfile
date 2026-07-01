FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go platform.yaml ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /platform-operator .

FROM alpine:3.20
RUN apk add --no-cache git ca-certificates
COPY --from=builder /platform-operator /platform-operator
USER 65534:65534
ENTRYPOINT ["/platform-operator"]
