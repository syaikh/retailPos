import type { User } from "$shared/types/user";

// The User type itself lives in $shared/types/user.ts: session state is a
// shared kernel (http-client, useRBAC depend on it) and re-derives User from
// there. This module keeps the auth-facing aliases.
export type { User } from "$shared/types/user";

export interface AuthState {
  isAuthenticated: boolean;
  user: User | null;
  loading: boolean;
}

// login() resolves with a discriminated union so callers can tell a rate
// limit (retryable, wait out Retry-After) apart from bad credentials (fix the
// input) instead of collapsing both into a single failure value.
export interface LoginSuccess {
  ok: true;
  data: { access_token: string; refresh_token: string; user: User };
}

export interface LoginFailure {
  ok: false;
  reason: "rate_limited" | "invalid";
  /** Seconds from the Retry-After header when reason === "rate_limited". */
  retryAfter?: number;
}

export type LoginResult = LoginSuccess | LoginFailure;
