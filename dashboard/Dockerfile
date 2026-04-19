# syntax=docker/dockerfile:1.6

# ── Build ────────────────────────────────────────────────────────────────────
FROM node:20-alpine AS builder
WORKDIR /src

COPY package.json package-lock.json* ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci 2>/dev/null || npm install

COPY . .
RUN npm run build

# ── Runtime — static files served by tiny static server ──────────────────────
FROM node:20-alpine AS runtime
WORKDIR /app

RUN npm install -g serve@14

COPY --from=builder /src/build /app/build

EXPOSE 3000
HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -qO- http://localhost:3000/ || exit 1

CMD ["serve", "-s", "/app/build", "-l", "3000"]
