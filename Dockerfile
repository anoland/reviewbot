FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download || true

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /forgejo-test-evaluator ./cmd/forgejo-test-evaluator

FROM alpine:3.19

RUN apk add --no-cache git ca-certificates

COPY --from=builder /forgejo-test-evaluator /usr/local/bin/forgejo-test-evaluator

ENTRYPOINT ["/usr/local/bin/forgejo-test-evaluator"]
