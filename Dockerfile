# --- Build stage ---
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /glab .

# --- Runtime stage (scratch + TLS certs) ---
FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /glab /glab
VOLUME /config
ENV GLAB_CONFIG_DIR=/config
EXPOSE 8080
ENTRYPOINT ["/glab", "mcp", "serve", "--transport", "http", "--host", "0.0.0.0"]
