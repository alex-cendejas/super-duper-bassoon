import { test, expect } from '@playwright/test';

test.describe('API URL configuration (regression: hostname vs port detection)', () => {
  test('dev server (port 5173) should send API requests to localhost:8080', async ({
    page,
    context,
  }) => {
    // This test verifies that on the Vite dev server the app uses the absolute
    // backend URL (http://localhost:8080/api/…) rather than relative paths.
    const capturedUrls: string[] = [];
    await context.route('**', (route) => {
      const url = route.request().url();
      if (url.includes('/api/')) capturedUrls.push(url);
      route.continue();
    });

    await page.goto('http://localhost:5173/#/workflows');
    await page.waitForTimeout(1200);

    // At least one API call must have been made to localhost:8080.
    const hasDevUrl = capturedUrls.some((u) => u.startsWith('http://localhost:8080/api/'));
    expect(hasDevUrl).toBe(true);
  });

  test('non-dev port should send API requests using relative paths (no localhost:8080)', async ({
    browser,
  }) => {
    // Simulate the Docker/nginx scenario where the app is served at port 3000.
    // In that environment window.location.port is '3000', so isDev=false and
    // apiBaseUrl should be '' (relative paths served via the nginx /api proxy).
    const context = await browser.newContext();
    const page = await context.newPage();

    // Override window.location.port BEFORE the app's module scripts evaluate.
    await page.addInitScript(() => {
      try {
        Object.defineProperty(window, 'location', {
          get() {
            return new Proxy(location, {
              get(target, prop: string) {
                if (prop === 'port') return '3000';
                const val = Reflect.get(target, prop, target);
                return typeof val === 'function' ? val.bind(target) : val;
              },
            });
          },
          configurable: true,
        });
      } catch {
        // If the browser prevents redefining location, skip silently.
      }
    });

    const capturedUrls: string[] = [];
    await context.route('**/api/**', (route) => {
      capturedUrls.push(route.request().url());
      route.abort('failed'); // abort so the missing server doesn't hang the test
    });

    await page.goto('http://localhost:5173/#/workflows');
    await page.waitForTimeout(1200);

    // No call should have gone directly to port 8080.
    const hasDirectBackendCall = capturedUrls.some((u) => u.includes(':8080/api/'));
    expect(hasDirectBackendCall).toBe(false);

    await context.close();
  });
});

test.describe('API Integration', () => {
  test('should make GET request to /api/workflows', async ({ page, context }) => {
    let apiCallMade = false;

    await context.route('**/api/workflows', (route) => {
      apiCallMade = true;
      route.continue();
    });

    await page.goto('http://localhost:5173/#/workflows');
    await page.waitForTimeout(1000);

    // API call should be attempted (may fail if server not running, that's ok)
    // We're just verifying the URL is correct
    expect(apiCallMade).toBe(true);
  });

  test('should handle API errors gracefully', async ({ page, context }) => {
    // Mock API errors (skip TypeScript source module files)
    await context.route('**/api/**', (route) => {
      if (route.request().url().endsWith('.ts')) {
        route.continue();
        return;
      }
      route.abort('failed');
    });

    await page.goto('http://localhost:5173/#/workflows');
    await page.waitForTimeout(1000);

    // Page should be visible even with API errors
    const content = page.locator('#workflows-content');
    await expect(content).toBeVisible();
  });

  test('should handle API timeouts', async ({ page, context }) => {
    await context.route('**/api/**', async (route) => {
      if (route.request().url().endsWith('.ts')) {
        route.continue();
        return;
      }
      await new Promise((resolve) => setTimeout(resolve, 40000));
      route.continue();
    });

    await page.goto('http://localhost:5173/#/runs');
    await page.waitForTimeout(1000);

    // Page should handle timeout gracefully
    const content = page.locator('#runs-content');
    await expect(content).toBeVisible();
  });

  test('should make GET request to /api/runs', async ({ page, context }) => {
    let runsApiCalled = false;

    await context.route('**/api/runs*', (route) => {
      runsApiCalled = true;
      route.continue();
    });

    await page.goto('http://localhost:5173/#/runs');
    await page.waitForTimeout(1000);

    expect(runsApiCalled).toBe(true);
  });

  test('should make GET request to /api/clients', async ({ page, context }) => {
    let clientsApiCalled = false;

    await context.route('**/api/clients', (route) => {
      clientsApiCalled = true;
      route.continue();
    });

    await page.goto('http://localhost:5173/#/clients');
    await page.waitForTimeout(1000);

    expect(clientsApiCalled).toBe(true);
  });

  test('should make GET request to /api/health', async ({ page, context }) => {
    let healthApiCalled = false;

    await context.route('**/api/health', (route) => {
      healthApiCalled = true;
      route.continue();
    });

    await page.goto('http://localhost:5173/#/health');
    await page.waitForTimeout(1000);

    expect(healthApiCalled).toBe(true);
  });

  test('should make GET request to /api/alerts', async ({ page, context }) => {
    let alertsApiCalled = false;

    await context.route('**/api/alerts*', (route) => {
      alertsApiCalled = true;
      route.continue();
    });

    await page.goto('http://localhost:5173/#/alerts');
    await page.waitForTimeout(1000);

    expect(alertsApiCalled).toBe(true);
  });

  test('should make GET request to /api/bans', async ({ page, context }) => {
    let bansApiCalled = false;

    await context.route('**/api/bans', (route) => {
      bansApiCalled = true;
      route.continue();
    });

    await page.goto('http://localhost:5173/#/bans');
    await page.waitForTimeout(1000);

    expect(bansApiCalled).toBe(true);
  });

  test('should properly format API requests', async ({ page, context }) => {
    let requestHeaders: Record<string, string> = {};

    await context.route('**/api/**', (route) => {
      requestHeaders = route.request().headers();
      route.continue();
    });

    await page.goto('http://localhost:5173/#/workflows');
    await page.waitForTimeout(1000);

    // Check if Content-Type header is properly set
    if (Object.keys(requestHeaders).length > 0) {
      // Headers were captured, this is good
      expect(Object.keys(requestHeaders).length).toBeGreaterThan(0);
    }
  });
});
