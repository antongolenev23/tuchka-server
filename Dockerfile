FROM golang:1.25-alpine AS builder

ARG VERSION=dev

# build
WORKDIR /usr/local/src
RUN apk --no-cache add musl-dev
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd
COPY internal/ ./internal
COPY docs/docs.go ./docs/docs.go
COPY pkg/ ./pkg

RUN go build -ldflags="-s -w -X main.version=${VERSION}" -o ./bin/tuchka_server cmd/tuchka-server/main.go

# run
FROM alpine AS runner

WORKDIR /tuchka-server
COPY --from=builder /usr/local/src/bin/tuchka_server ./
COPY ./config /etc/tuchka-server
COPY migrations /migrations

CMD ["./tuchka_server"]




