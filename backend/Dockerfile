# --- build stage ---
FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/server .

# --- run stage ---
FROM alpine:3.20
WORKDIR /app
COPY --from=build /bin/server /bin/server
COPY migrations ./migrations
EXPOSE 8080
# Entry applies migrations, seeds the tax_rules table, then serves — so a fresh
# deploy (e.g. Railway) self-migrates. Compose overrides the command if needed.
CMD ["sh", "-c", "/bin/server migrate && /bin/server seed && /bin/server serve"]
