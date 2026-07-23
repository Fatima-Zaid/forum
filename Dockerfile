# ---- Build stage ----
FROM golang:1.22-alpine AS builder

# gcc + musl-dev needed because go-sqlite3 uses cgo
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

# CGO_ENABLED=1 is required for go-sqlite3 to compile
RUN CGO_ENABLED=1 GOOS=linux go build -o forum main.go

# ---- Final stage ----
FROM alpine:latest

RUN apk add --no-cache sqlite-libs ca-certificates

WORKDIR /app

COPY --from=builder /app/forum .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/database/schema.sql ./database/schema.sql

EXPOSE 8080

CMD ["./forum"]