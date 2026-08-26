/// <reference types="vite/client" />

declare const __FRONTEND_PORT__: string;
declare const __BACKEND_PORT__: string;

interface ImportMetaEnv {
  readonly VITE_PRINT_MODE?: string;
  readonly VITE_PRINT_AGENT_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
