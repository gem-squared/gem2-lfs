FROM golang:1.25-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /gem2-lfs ./cmd/gem2-lfs/

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /gem2-lfs /usr/local/bin/gem2-lfs

VOLUME /data
EXPOSE 9090

ENTRYPOINT ["gem2-lfs"]
CMD ["serve", "--port", "9090", "--db-path", "/data/gem2-lfs.db"]
