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

export type LoginResult =
  { access_token: string; refresh_token: string; user: User } | false;
