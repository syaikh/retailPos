export interface User {
  id: number;
  username: string;
  email: string;
  role_id: number;
  role?: { id: number; name: string } | string;
  is_active: boolean;
  last_login?: string;
  created_at?: string;
}

export interface CreateUserPayload {
  username: string;
  email: string;
  password: string;
  role_id: number;
  is_active: boolean;
}

export interface UpdateUserPayload {
  username?: string;
  email?: string;
  password?: string;
  role_id?: number;
  is_active?: boolean;
}

export interface Role {
  id: number;
  name: string;
  description: string;
  is_system: boolean;
  permissions: string[];
  created_at?: string;
}

export interface CreateRolePayload {
  name: string;
  description: string;
}

export interface UpdateRolePayload {
  name?: string;
  description?: string;
}

export interface Permission {
  id: number;
  name: string;
  code: string;
  description?: string;
}

export interface AuditLog {
  id: number;
  user_id: number;
  username: string;
  action: string;
  entity_type: string;
  entity_id: number;
  details?: string;
  old_values?: Record<string, unknown>;
  new_values?: Record<string, unknown>;
  ip_address?: string;
  user_agent?: string;
  created_at: string;
}

export interface AuditLogFilters {
  limit: number;
  offset: number;
  search: string;
  start_date: string;
  end_date: string;
  action?: string;
  entity_type?: string;
}
