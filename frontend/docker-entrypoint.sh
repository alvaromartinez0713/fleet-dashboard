#!/bin/sh
set -e

echo "Generating config.js with environment variables..."
cat > /app/dist/config.js << CONFIGEOF
window.ENV = {
  VITE_DATA_MODE: '${VITE_DATA_MODE:-mock}',
  VITE_API_BASE_URL: '${VITE_API_BASE_URL:-http://localhost:8080}',
  VITE_SSE_ENDPOINT: '${VITE_SSE_ENDPOINT:-http://localhost:8080/api/vehicles/updates}',
  VITE_MAPBOX_TOKEN: '${VITE_MAPBOX_TOKEN:-}',
};
CONFIGEOF

echo "Starting server..."
exec serve -s /app/dist -l 3000

