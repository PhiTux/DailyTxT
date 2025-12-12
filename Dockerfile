## Multi-stage build: frontend (SvelteKit) + backend (Go) + Nginx runtime

# ---------- FRONTEND BUILD ----------
FROM node:24-alpine AS frontend-builder
WORKDIR /app/frontend

# Install dependencies and build SvelteKit (outputs to build/)
COPY frontend/package*.json ./
RUN apk update && apk add build-base cairo-dev pango-dev giflib-dev g++ make py3-pip && npm ci
COPY frontend/ ./
RUN npm run build
RUN npm prune --production

# ---------- BACKEND BUILD ----------
FROM golang:1.25-alpine AS backend-builder
WORKDIR /src

# Install git and CA certs for `go mod download`
RUN apk add --no-cache git ca-certificates && update-ca-certificates

COPY backend/go.mod backend/go.sum ./backend/
WORKDIR /src/backend
RUN go mod download

# Copy backend source and build static binary
COPY backend/ ./
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -o /out/dailytxt ./

# ---------- RUNTIME (NGINX + BACKEND) ----------
FROM nginx:alpine-slim AS runtime

# Create app directories
RUN mkdir -p /usr/share/nginx/html \
	&& mkdir -p /etc/nginx/conf.d \
	&& mkdir -p /data

# Copy frontend build
COPY --from=frontend-builder /app/frontend/build/ /usr/share/nginx/html/

# Copy backend binary
COPY --from=backend-builder /out/dailytxt /usr/local/bin/dailytxt

# Copy application version file (read by backend at startup)
COPY backend/version /version

# Copy nginx config and entrypoint
COPY nginx/default.conf /etc/nginx/conf.d/default.conf
COPY docker-entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

VOLUME ["/data"]

EXPOSE 80
ENTRYPOINT ["/entrypoint.sh"]

