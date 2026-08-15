import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'LoginPage.svelte'), 'utf-8');
}

describe('LoginPage.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Button, Input, and LanguageSwitch from shared/ui', () => {
    expect(src).toContain("import { Button, Input, LanguageSwitch } from '$shared/ui'");
  });

  it('imports labels from shared/i18n', () => {
    expect(src).toContain("import { labels, t } from '$shared/i18n'");
  });

  it('imports login and useAuthStore from auth module', () => {
    expect(src).toContain("import { login } from '$modules/auth'");
    expect(src).toContain("import { useAuthStore } from '$modules/auth'");
  });

  it('uses $state for form fields', () => {
    expect(src).toContain('let username = $state');
    expect(src).toContain('let password = $state');
    expect(src).toContain('let loading = $state');
    expect(src).toContain('let errorMsg = $state');
    expect(src).toContain('let showPassword = $state');
  });

  it('has handleLogin function with validation', () => {
    expect(src).toContain('async function handleLogin');
    expect(src).toContain('errorMsg = labels.usernamePasswordRequired');
    expect(src).toContain('errorMsg = labels.invalidCredentials');
  });

  it('has password visibility toggle', () => {
    expect(src).toContain('showPassword ? \'text\' : \'password\'');
    expect(src).toContain('aria-label={showPassword ? labels.hidePassword : labels.showPassword}');
  });

  it('renders error message with role="alert"', () => {
    expect(src).toContain('{#if errorMsg}');
    expect(src).toContain('role="alert"');
  });

  it('has login form with username and password inputs', () => {
    expect(src).toContain('id="username"');
    expect(src).toContain('id="password"');
    expect(src).toContain('autocomplete="username"');
    expect(src).toContain('autocomplete="current-password"');
  });

  it('has sign in button with loading state', () => {
    expect(src).toContain('{labels.login}');
    expect(src).toContain('{labels.signingIn}');
    expect(src).toContain('{#if loading}');
  });

  it('navigates to default route on successful login', () => {
    expect(src).toContain('goto(getDefaultRoute(result.user))');
  });
});
