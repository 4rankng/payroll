# Multi-stage build for React frontend with BuildKit optimizations.
#
# NOTE: No `# syntax=docker/dockerfile:1.4` pin on purpose. That directive
# forces BuildKit to fetch the frontend image from Docker Hub before every
# build; from Vietnam (and other flaky-Docker-Hub networks) that fetch often
# fails with "TLS handshake timeout" and blocks deploys. The builtin frontend
# bundled with BuildKit (v0.30+ in our `bb` builder) already supports every
# feature used below (--mount=type=cache, --platform=$BUILDPLATFORM, etc.) and
# is newer than dockerfile:1.4 anyway, so the pin buys nothing.
#
# The deps and builder stages run on $BUILDPLATFORM (native arch of the machine
# invoking buildx). The Vite/Workbox toolchain is heavy and its output (dist/)
# is purely static, architecture-independent files — so there is no reason to
# run it under QEMU x86_64 emulation when targeting linux/amd64 from an arm64
# host. Pinning these stages to $BUILDPLATFORM avoids a Node.js exit-time
# SIGSEGV that occurs when esbuild/SWC/workbox worker threads run under QEMU.
# Only the final production stage follows the --platform target (linux/amd64);
# it receives dist/ via COPY, which crosses architectures safely.

# Dependencies stage - caches node_modules separately (native arch, no QEMU)
#
# Base image is vendored into our own GHCR namespace (mirrored from Docker Hub
# via `make mirror-bases`) and pinned by manifest-list digest so builds never
# depend on Docker Hub (the #1 cause of "TLS handshake timeout" deploy failures
# from Vietnam). The digest is the multi-arch manifest list, so BuildKit still
# picks the correct arch per $BUILDPLATFORM.
FROM --platform=$BUILDPLATFORM ghcr.io/4rankng/node:22-alpine@sha256:c610fcdfb1d5b4740dd70c284ed3cb16bb857e0f7166196e36a5501df7a3aa32 AS deps
WORKDIR /app

# Enable corepack to use pnpm (version pinned via package.json's packageManager field)
RUN corepack enable

# Copy package files first for better layer caching
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./

# Install dependencies with cache mount for faster rebuilds.
# mkdir node_modules/.vite: the builder stage bind-mounts this node_modules
# read-only and overlays a writable cache mount at node_modules/.vite (Vite's
# dep-optimizer cacheDir) — runc needs the mountpoint to already exist in the
# read-only source, so it is created here, after install (pnpm leaves unknown
# dot-dirs in node_modules alone, same as a local dev checkout).
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile && mkdir -p node_modules/.vite

# Builder stage - runs the Vite/Workbox toolchain natively ($BUILDPLATFORM) to
# avoid the QEMU-emulation SIGSEGV. Output (dist/) is static, copied into the
# linux/amd64 production stage below.
FROM --platform=$BUILDPLATFORM ghcr.io/4rankng/node:22-alpine@sha256:c610fcdfb1d5b4740dd70c284ed3cb16bb857e0f7166196e36a5501df7a3aa32 AS builder
WORKDIR /app

# Enable corepack so pnpm (pinned via package.json#packageManager) is available
# for the build step. Dependencies themselves come from the deps stage.
RUN corepack enable

# Build arguments for environment variables
ARG VITE_API_BASE_URL=https://tingting.vip
ARG VITE_API_VERSION=/api/v1
ARG VITE_API_TIMEOUT=30000

# Set environment variables for build
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
ENV VITE_API_VERSION=${VITE_API_VERSION}
ENV VITE_API_TIMEOUT=${VITE_API_TIMEOUT}

# Copy source code (excluding files in .dockerignore). package.json /
# pnpm-lock.yaml come from the context too; the deps stage owns installs.
COPY . .

# Build with node_modules bind-mounted from the deps stage — NOT copied.
#
# `COPY --from=deps /app/node_modules` materializes ~100k files into a fresh
# layer and re-exports it via --cache-to mode=max through Docker Desktop's
# VM→host share on every builder-stage cache miss (~6 min per build, the
# single slowest step of the deploy). A stage bind mount creates no layer and
# exports nothing — pnpm/vite only need to READ node_modules. The one thing
# vite writes there is its dep-optimizer cache, whose real default cacheDir is
# node_modules/.vite (the previous /app/.vite mount was never used); it gets a
# writable cache mount overlaid on the read-only bind.
RUN --mount=type=bind,from=deps,source=/app/node_modules,target=/app/node_modules \
    --mount=type=cache,target=/app/node_modules/.vite \
    pnpm build

# Production stage - static files only.
# No --platform here → builds for the --platform target (linux/amd64). Same
# vendored GHCR base, digest-pinned.
FROM ghcr.io/4rankng/node:22-alpine@sha256:c610fcdfb1d5b4740dd70c284ed3cb16bb857e0f7166196e36a5501df7a3aa32 AS production

# Create app directory
WORKDIR /app

# Install serve and create the runtime user BEFORE copying dist. dist changes
# on every build, so layers that never change (pinned base → serve install,
# users, serve.json) sit above it and stay cached across builds.
RUN npm install -g serve

COPY serve.json ./dist/serve.json

RUN addgroup -g 1001 -S nodejs && adduser -S nextjs -u 1001

# Copy built assets from builder stage
COPY --from=builder /app/dist ./dist

# Expose port 3000
EXPOSE 3000

# Use non-root user for security
USER nextjs

# Serve the built app
CMD ["serve", "-s", "dist", "-l", "3000", "-c", "serve.json"]
