// Pure re-export shim: the implementation moved to $shared/api/session.ts so
// shared code (http-client) can read the token without importing $modules.
// See docs/adr/ADR_Frontend_Module_Import_Boundaries.md.
export * from "$shared/api/session";
