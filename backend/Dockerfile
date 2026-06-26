FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY crd/ ./crd/
RUN CGO_ENABLED=0 GOOS=linux go build -o devportal-backend .

FROM alpine:latest
RUN apk add --no-cache curl kubectl
COPY --from=builder /app/devportal-backend /usr/local/bin/devportal-backend
ENV PORT=3000
HEALTHCHECK --interval=30s --timeout=10s --retries=3 CMD curl -f http://localhost:3000/health || exit 1
EXPOSE 3000
CMD ["devportal-backend"]
