FROM node:20-alpine AS deps
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

FROM node:20-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY frontend/ .
ENV NODE_ENV=production
ENV NEXT_PUBLIC_API_URL=https://data-api-production-575e.up.railway.app
ENV NEXT_PUBLIC_INGESTION_URL=https://ingestion-production-6bec.up.railway.app
ENV NEXT_PUBLIC_SSE_URL=https://processor-production-6c1e.up.railway.app
RUN npm run build

FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production

RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs

COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static
COPY --from=builder --chown=nextjs:nodejs /app/public ./public

USER nextjs
EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

# Verify server.js exists and is executable
RUN ls -la /app/server.js || (echo "server.js not found!" && exit 1)

CMD ["node", "server.js"]
