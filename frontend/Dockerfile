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

# Install dependencies with cache mount for faster rebuilds
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile

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

# Copy dependencies from deps stage
COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /app/package.json /app/pnpm-lock.yaml ./

# Copy source code (excluding files in .dockerignore)
COPY . .

# Build the application with build cache
RUN --mount=type=cache,target=/app/.vite \
    pnpm build

# Production stage - static files only.
# No --platform here → builds for the --platform target (linux/amd64). Same
# vendored GHCR base, digest-pinned.
FROM ghcr.io/4rankng/node:22-alpine@sha256:c610fcdfb1d5b4740dd70c284ed3cb16bb857e0f7166196e36a5501df7a3aa32 AS production

# Create app directory
WORKDIR /app

# Copy built assets from builder stage
COPY --from=builder /app/dist ./dist

# Copy serve config into dist (adds Permissions-Policy headers for geolocation and FedCM / Google Sign-In)
COPY serve.json ./dist/serve.json

# Install serve to serve static files
RUN npm install -g serve

# Expose port 3000
EXPOSE 3000

# Use non-root user for security
RUN addgroup -g 1001 -S nodejs
RUN adduser -S nextjs -u 1001
USER nextjs

# Serve the built app
CMD ["serve", "-s", "dist", "-l", "3000", "-c", "serve.json"]
