// Pure re-export shim: the store moved to $shared/stores/auth.svelte.ts so
// shared code (http-client, useRBAC) can read session state without importing
// $modules. Importing either path resolves to the same module instance —
// guarded by kernel-identity.test.ts in services/__tests__/.
// See docs/adr/ADR_Frontend_Module_Import_Boundaries.md.
export { useAuthStore } from "$shared/stores/auth.svelte";
