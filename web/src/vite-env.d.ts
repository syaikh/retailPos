/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_PRINT_MODE?: string;
  readonly VITE_PRINT_AGENT_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
