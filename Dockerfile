FROM golang:1.21-alpine as builder
WORKDIR /app
COPY . .
RUN apk add --no-cache gcc musl-dev && CGO_ENABLED=1 go build -o /miniomatic

FROM alpine:latest
WORKDIR /app
COPY --from=builder /miniomatic /app/
CMD ["/app/miniomatic"]