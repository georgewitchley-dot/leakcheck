# --- Frontend: React + TypeScript (Vite) ---
FROM node:22-alpine AS web
WORKDIR /web
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm test && npm run build

# --- Backend: Go ---
FROM golang:1.24-alpine AS api
WORKDIR /src
COPY backend/go.mod backend/go.sum* ./
COPY backend/ ./
# Resolves and checksum-verifies dependencies (writes go.sum if missing).
RUN go mod tidy && go mod download
RUN go test -tags nopg ./... && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/pwcheck .

# --- Runtime ---
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=api /out/pwcheck /app/pwcheck
COPY --from=web /web/dist /app/public
ENV STATIC_DIR=/app/public PORT=8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/pwcheck"]
