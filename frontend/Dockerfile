# ============================================================================
# Build stage: Install dependencies
# ============================================================================
FROM node:20-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .

# ============================================================================
# Production build
# ============================================================================
FROM builder AS prod-build

RUN npm run build

# ============================================================================
# Production: Serve with runtime config injection
# ============================================================================
FROM node:20-alpine AS prod

WORKDIR /app

RUN npm install -g serve

# Copy dist from build stage
COPY --from=prod-build /app/dist /app/dist

COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

EXPOSE 3000

ENTRYPOINT ["/app/docker-entrypoint.sh"]

# ============================================================================
# Development
# ============================================================================
FROM builder AS dev

USER root
EXPOSE 3000
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]
