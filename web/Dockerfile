# ==========================================
# Web frontend - Multi-stage build
# ==========================================

# --- Build stage ---
FROM node:25-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

ARG VITE_API_BASE_URL
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}

COPY src/ src/
COPY public/ public/
COPY index.html tsconfig.json tsconfig.app.json tsconfig.node.json vite.config.ts ./
RUN npm run build

# --- Runtime stage ---
FROM nginxinc/nginx-unprivileged:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]
