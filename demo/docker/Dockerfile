# Helios Demo - Multi-stage Docker build
FROM node:18-alpine AS backend-builder

WORKDIR /app/backend
COPY backend/package*.json ./
RUN npm ci --only=production

FROM node:18-alpine AS frontend-builder

WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Production image
FROM node:18-alpine

# Install PM2 for process management
RUN npm install -g pm2

# Create app directory
WORKDIR /app

# Copy backend
COPY backend/ ./backend/
COPY --from=backend-builder /app/backend/node_modules ./backend/node_modules

# Copy built frontend
COPY --from=frontend-builder /app/frontend/build ./frontend/build

# Copy process configuration
COPY docker/ecosystem.config.js ./

# Create startup script
COPY docker/start.sh ./
RUN chmod +x start.sh

# Install serve for static files
RUN npm install -g serve

# Expose ports
EXPOSE 3000 3001

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:3001/api/stats || exit 1

# Labels
LABEL org.label-schema.name="Helios Demo"
LABEL org.label-schema.description="Parallel Universe Engine Interactive Demo"
LABEL org.label-schema.version="1.0.0"
LABEL org.label-schema.schema-version="1.0"

# Start the application
CMD ["./start.sh"]