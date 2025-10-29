FROM golang:alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /subs-service

COPY app/go.mod /subs-service/

RUN go mod download

COPY app/ /subs-service/

RUN go build -o build/main cmd/main.go

FROM alpine:latest AS runner

WORKDIR /app

COPY --from=builder /subs-service/build/main /app/
COPY /config.yaml /app/config.yaml
COPY /migrations /app/migrations

ENV CONFIG_PATH=/app/config.yaml
ENV APP_MIGRATION_DIR=/app/migrations

CMD [ "/app/main" ]