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
# Entry runs migrate + seed then serves; overridden per-service in compose if needed.
CMD ["/bin/server", "serve"]
