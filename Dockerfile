FROM golang:1.21-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN apk add --no-cache gcc musl-dev && CGO_ENABLED=1 go build -ldflags '-w -s' -o /miniomatic

FROM alpine:latest
WORKDIR /app
COPY --from=builder /miniomatic /app/
RUN apk add --no-cache ca-certificates
ENTRYPOINT ["/app/miniomatic"]