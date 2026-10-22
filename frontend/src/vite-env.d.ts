/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_DATA_MODE: string;
  readonly VITE_API_BASE_URL: string;
  readonly VITE_SSE_ENDPOINT: string;
  readonly VITE_MAPBOX_TOKEN: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

