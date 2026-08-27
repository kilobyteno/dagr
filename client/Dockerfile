# syntax=docker/dockerfile:1

FROM node:20-bookworm-slim AS base
WORKDIR /app
ENV COREPACK_ENABLE_DOWNLOAD_PROMPT=0
RUN corepack enable && corepack prepare pnpm@10.34.5 --activate

FROM base AS deps
COPY client/package.json client/pnpm-lock.yaml client/.npmrc ./
RUN pnpm install --frozen-lockfile

FROM base AS builder
COPY --from=deps /app/node_modules ./node_modules
COPY client/package.json client/pnpm-lock.yaml client/.npmrc client/vite.config.ts client/tsconfig.json client/tsconfig.node.json client/index.html client/components.json ./
COPY client/public ./public
COPY client/src ./src
ARG VITE_DAGR_CLOUD_URL
ARG VITE_DAGR_SELF_HOSTED_URL
ARG VITE_DAGR_DEFAULT_MODE
ENV VITE_DAGR_CLOUD_URL=$VITE_DAGR_CLOUD_URL
ENV VITE_DAGR_SELF_HOSTED_URL=$VITE_DAGR_SELF_HOSTED_URL
ENV VITE_DAGR_DEFAULT_MODE=$VITE_DAGR_DEFAULT_MODE
RUN pnpm build:web

FROM nginx:1.27-alpine AS runner
COPY client/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1/healthz >/dev/null || exit 1
