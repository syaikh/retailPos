const { devices } = require('@playwright/test');

/**
 * Playwright E2E Test Configuration
 * 
 * For local development:
 *   npx playwright test
 * 
 * In CI/CD:
 *   CI=1 npx playwright test
 * 
 * With services auto-start:
 *   npx playwright test -- --config=playwright.config.js
 */
const config = {
  // Test files location
  testDir: 'tests/e2e',

  // Run tests sequentially within each file
  fullyParallel: false,

  // Don't fail on CI if tests marked with .only
  forbidOnly: !!process.env.CI,

  // Retry failed tests (more retries in CI to handle flaky networks)
  retries: process.env.CI ? 2 : 0,

  // 1 worker: sequential execution to avoid memory issues (Chrome + Vite + Go)
  workers: 1,

  // Reporter
  reporter: [
    ['list'],
    ['html', { outputFolder: 'playwright-report' }],
    ['json', { outputFile: 'test-results.json' }]
  ],

  // Global test timeout (increased for container startup)
  timeout: 60 * 1000,

  // Test hook timeouts
  expect: {
    timeout: 15000
  },

// Use baseURL from environment or default
   use: {
    baseURL: process.env.FRONTEND_BASE_URL || 'http://localhost:5173',

    // Capture trace on first retry for debugging
    trace: 'on-first-retry',

    // Viewport
    viewport: { width: 1280, height: 720 },

    // Ignore HTTPS errors (self-signed certs)
    ignoreHTTPSErrors: true,

    // Increase navigation timeout (for slow container startup)
    navigationTimeout: 30000,

    // Action timeout (click, fill, etc)
    actionTimeout: 20000,

    // Screenshots on failure
    screenshot: 'only-on-failure',

    // Video on failure (optional, makes report large)
    // video: 'on-first-retry',

    launchOptions: {
      executablePath: '/usr/bin/google-chrome',
    },
  },

  // Configure projects for different browsers
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    // Uncomment for cross-browser testing:
    // {
    //   name: 'firefox',
    //   use: { ...devices['Desktop Firefox'] },
    // },
    // {
    //   name: 'webkit',
    //   use: { ...devices['Desktop Safari'] },
    // },
  ],

  // Global setup (optional - for starting test environment)
  // globalSetup: require.resolve('./tests/e2e/setup-global'),

  // Global teardown (optional - for cleanup)
  // globalTeardown: require.resolve('./tests/e2e/teardown-global'),
};

module.exports = config;
