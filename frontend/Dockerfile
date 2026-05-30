# syntax=docker/dockerfile:1.4
# Multi-stage build for React frontend with BuildKit optimizations

# Dependencies stage - caches node_modules separately
FROM node:20-alpine AS deps
WORKDIR /app

# Configure yarn for better performance
RUN yarn config set network-timeout 600000 && \
    yarn config set cache-folder /yarn-cache

# Copy package files first for better layer caching
COPY package.json yarn.lock ./

# Install dependencies with cache mount for faster rebuilds
RUN --mount=type=cache,target=/yarn-cache \
    --mount=type=cache,target=/root/.yarn \
    yarn install --frozen-lockfile --network-timeout 600000

# Builder stage - handles source code and build
FROM node:20-alpine AS builder
WORKDIR /app

# Build arguments for environment variables
ARG VITE_API_BASE_URL=https://tingting.vip
ARG VITE_API_VERSION=/api/v1
ARG VITE_API_TIMEOUT=30000
ARG VITE_GOOGLE_CLIENT_ID

# Set environment variables for build
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
ENV VITE_API_VERSION=${VITE_API_VERSION}
ENV VITE_API_TIMEOUT=${VITE_API_TIMEOUT}
ENV VITE_GOOGLE_CLIENT_ID=${VITE_GOOGLE_CLIENT_ID}

# Copy dependencies from deps stage
COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /app/package.json /app/yarn.lock ./

# Copy source code (excluding files in .dockerignore)
COPY . .

# Build the application with build cache
RUN --mount=type=cache,target=/app/.vite \
    yarn build

# Production stage - static files only
FROM node:20-alpine AS production

# Create app directory
WORKDIR /app

# Copy built assets from builder stage
COPY --from=builder /app/dist ./dist

# Copy serve config into dist (adds Permissions-Policy header required for FedCM / Google Sign-In)
COPY serve.json ./dist/serve.json

# Install serve to serve static files
RUN yarn global add serve

# Expose port 3000
EXPOSE 3000

# Use non-root user for security
RUN addgroup -g 1001 -S nodejs
RUN adduser -S nextjs -u 1001
USER nextjs

# Serve the built app
CMD ["serve", "-s", "dist", "-l", "3000", "-c", "serve.json"]