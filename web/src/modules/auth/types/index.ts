export interface User {
  id: number;
  username: string;
  email: string;
  role?: { id: number; name: string } | string;
  role_id?: number;
  store_id?: number;
  reports_to?: number | null;
  reports_to_username?: string;
  is_active?: boolean;
  last_login?: string;
  created_at?: string;
  updated_at?: string;
  permissions?: string[];
  language?: string;
  theme?: string;
  /** True when the account still owes a first-login password rotation. */
  must_change_password?: boolean;
}

export interface AuthState {
  isAuthenticated: boolean;
  user: User | null;
  loading: boolean;
}

export type LoginResult =
  { access_token: string; refresh_token: string; user: User } | false;
