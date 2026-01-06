# Builder stage
FROM oven/bun:1 AS builder

# Install Go using official Go image as a base
COPY --from=golang:1.25 /usr/local/go /usr/local/go

ENV PATH=/usr/local/go/bin:$PATH

WORKDIR /app

# Copy Go module files (go.sum may not exist if no dependencies)
COPY go.mod go.sum* ./

# Download Go dependencies
RUN go mod download || true

# Copy Go source code
COPY src/ src/

# Build WASM module
RUN GOOS=js GOARCH=wasm go build -o web/public/main.wasm ./src/cmd/wasm

# Copy web directory
WORKDIR /app/web
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile

# Copy web source code
COPY web/src/ src/
COPY web/public/ public/
COPY web/index.html ./
COPY web/vite.config.ts ./
COPY web/tsconfig.json ./
COPY web/rescript.json ./

# Build web frontend
RUN bun run build

# Runner stage
FROM caddy:2-alpine AS runner

WORKDIR /srv

# Copy built web assets from builder
COPY --from=builder /app/web/dist /srv

# Copy Caddyfile
COPY web/Caddyfile /etc/caddy/Caddyfile

EXPOSE 3000

CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile"]
