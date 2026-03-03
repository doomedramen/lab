# Base stage for dependencies
FROM node:22-alpine AS base
RUN corepack enable && corepack prepare pnpm@latest --activate
WORKDIR /app

# Install dependencies
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml turbo.json ./
COPY packages ./packages
COPY apps/web/package.json ./apps/web/
RUN pnpm install --frozen-lockfile --ignore-scripts

# Build stage
FROM base AS builder
WORKDIR /app
COPY apps/web ./apps/web
ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
RUN pnpm --filter web build

# Production stage
FROM node:22-alpine AS production
WORKDIR /app

RUN corepack enable && corepack prepare pnpm@latest --activate

# Copy necessary files from build stage
COPY --from=builder /app/package.json /app/pnpm-lock.yaml /app/pnpm-workspace.yaml /app/turbo.json ./
COPY --from=builder /app/packages ./packages
COPY --from=builder /app/apps/web/package.json ./apps/web/
COPY --from=builder /app/apps/web/.next ./apps/web/.next
COPY --from=builder /app/apps/web/public ./apps/web/public

# Install only production dependencies
# Note: In a workspace, this can be tricky, but since we have the full workspace structure
# copied or symlinked, pnpm should be able to prune.
RUN pnpm install --frozen-lockfile --prod --ignore-scripts

EXPOSE 3000

WORKDIR /app/apps/web
ENV NODE_ENV=production
ENV PORT=3000

CMD ["pnpm", "start"]
