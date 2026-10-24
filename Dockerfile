# Builder stage
FROM node:24.22.1-bookworm-slim AS builder

# Keep the package manager and Node runtime aligned with CI.
COPY --from=oven/bun:1.4.3 /usr/local/bin/bun /usr/local/bin/bun

# Install Go using official Go image as a base
COPY --from=golang:1.26 /usr/local/go /usr/local/go

ENV PATH=/usr/local/go/bin:$PATH

WORKDIR /app

# Copy Go module files (go.sum may not exist if no dependencies)
COPY go.mod go.sum* ./

# Download Go dependencies
RUN go mod download

# Copy Go source code
COPY src/ src/
COPY scripts/build-wasm.sh scripts/build-wasm.sh

# Copy web directory
WORKDIR /app/web
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile

# Copy web source code
COPY web/src/ src/
COPY web/public/ public/
COPY web/index.html ./
COPY web/vite.config.ts ./
COPY web/build/ build/
COPY web/tsconfig.json ./
COPY web/rescript.json ./

# Build after copying public assets so committed WASM cannot replace this build.
# The shared script also copies the matching Go JavaScript runtime.
RUN cd /app && sh scripts/build-wasm.sh

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
