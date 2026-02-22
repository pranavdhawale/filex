# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Enable corepack for pnpm if needed, but we'll use npm as per package.json
# COPY package.json package-lock.json ./
COPY package*.json ./

RUN npm ci

COPY . .

# Build args for Next.js (baked at build time)
ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL

RUN npm run build

# Final stage
FROM node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production

# Copy next.config.ts if it exists, or .js/.mjs
COPY --from=builder /app/next.config.ts ./
COPY --from=builder /app/package.json ./
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/node_modules ./node_modules

EXPOSE 3000

CMD ["npm", "start"]
