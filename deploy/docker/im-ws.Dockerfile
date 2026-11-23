FROM golang:1.26-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" \
    -o /out/app ./apps/im/ws

FROM alpine:3.23.5

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -G app -S app

COPY --from=builder /out/app /usr/local/bin/app

WORKDIR /app
USER app

ENTRYPOINT ["/usr/local/bin/app", "-f", "/app/config.yaml"]