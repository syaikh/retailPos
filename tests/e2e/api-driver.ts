import { API_BASE, authHeader, getToken, decodeJWT, TEST_USERS } from './fixtures';

export type RoleKey = keyof typeof TEST_USERS;

export interface ApiResult {
  status: number;
  ok: boolean;
  body: any;
  headers: Record<string, string>;
}

async function resolve(res: any): Promise<ApiResult> {
  const text = await res.text();
  let body: any = null;
  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      body = text;
    }
  }
  return { status: res.status(), ok: res.ok(), body, headers: res.headers() };
}

/**
 * API driver — the "different driver" from the acceptance-test playbook. It
 * speaks HTTP instead of clicking the DOM, so behaviour (CRUD, RBAC) is tested
 * at the cheapest reliable layer and never flakes on render timing or hydration.
 *
 * Specs describe essential behaviour (e.g. "a created brand appears in the
 * list") against this driver; the DOM/browser specs are kept for genuine UI
 * behaviour only (rendering, validation messages, navigation).
 */
export class ApiDriver {
  constructor(private request: any, public token: string) {}

  private headers() {
    return authHeader(this.token);
  }

  private url(path: string) {
    return `${API_BASE}${path}`;
  }

  async get(path: string) {
    return resolve(await this.request.get(this.url(path), { headers: this.headers() }));
  }
  async post(path: string, data: any) {
    return resolve(await this.request.post(this.url(path), { headers: this.headers(), data }));
  }
  async put(path: string, data: any) {
    return resolve(await this.request.put(this.url(path), { headers: this.headers(), data }));
  }
  async patch(path: string, data: any) {
    return resolve(await this.request.patch(this.url(path), { headers: this.headers(), data }));
  }
  async del(path: string) {
    return resolve(await this.request.delete(this.url(path), { headers: this.headers() }));
  }

  /**
   * POST a single file as multipart/form-data (defaults to the field name
   * `file`). Useful for upload endpoints (import), where Playwright accepts
   * `{ name, mimeType, buffer }`.
   */
  async multipart(path: string, file: { name: string; mimeType: string; buffer: Buffer }, fieldName = 'file') {
    return resolve(
      await this.request.post(this.url(path), {
        headers: this.headers(),
        multipart: { [fieldName]: file },
      }),
    );
  }

  permissions(): string[] {
    const claims: any = decodeJWT(this.token);
    return Array.isArray(claims.permissions) ? claims.permissions : [];
  }

  /** True if the role holds `prefix.*` (or any code under that group). */
  hasGroup(prefix: string): boolean {
    return this.permissions().some((p) => p === `${prefix}.*` || p.startsWith(`${prefix}.`));
  }
}

export async function apiAs(request: any, role: RoleKey): Promise<ApiDriver> {
  const token = await getToken(request, TEST_USERS[role].username, TEST_USERS[role].password);
  return new ApiDriver(request, token);
}

/** Login as an arbitrary username/password (incl. dynamically created users). */
export async function loginDriver(request: any, username: string, password: string): Promise<ApiDriver> {
  const token = await getToken(request, username, password);
  return new ApiDriver(request, token);
}
