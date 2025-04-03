
# Base image for development
FROM golang:1.23-alpine AS baseimage

WORKDIR /app

RUN apk add --no-cache make gcc git sqlite-dev musl-dev

COPY ./go.mod ./go.sum /app/

RUN go mod download

# Base image for development
FROM baseimage AS dev

WORKDIR /app

RUN go install github.com/air-verse/air@latest

CMD ["air"]

# Builder image
FROM baseimage AS builder

COPY pkg/ .

RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o main .

# Prodoct Image
FROM alpine:latest AS prod

WORKDIR /app/

RUN apk add --no-cache sqlite

COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]
