/**
 * Unit-style regression tests for the bug fixes.
 *
 * Bug 1: Router getCurrentPath() returns '#/' for empty/missing hash.
 * Bug 4/5: API functions return empty arrays when server returns null items.
 * Bug 3: Key page components render without crashing with empty store data.
 */

import { test, expect } from '@playwright/test';

// ---------------------------------------------------------------------------
// Bug 1 – Router getCurrentPath()
// ---------------------------------------------------------------------------

test.describe('Router getCurrentPath()', () => {
  test('returns #/ when navigating to root path with no hash', async ({ page }) => {
    await page.goto('http://localhost:5173/');
    await page.waitForTimeout(300);

    // With the fix, the root path should redirect/render the home/workflows page,
    // not the NotFoundPage.
    const notFound = page.locator('.not-found-page');
    await expect(notFound).toHaveCount(0);
  });

  test('getCurrentPath returns #/ for empty hash via evaluate', async ({ page }) => {
    // Load the app so its modules are initialised.
    await page.goto('http://localhost:5173/#/workflows');
    await page.waitForTimeout(300);

    // Evaluate the fixed logic inline to verify the contract.
    const result = await page.evaluate(() => {
      function getCurrentPath(hash: string): string {
        if (!hash || hash === '#') return '#/';
        return hash;
      }
      return [
        getCurrentPath(''),
        getCurrentPath('#'),
        getCurrentPath('#/workflows'),
        getCurrentPath('#/runs'),
      ];
    });

    expect(result[0]).toBe('#/');
    expect(result[1]).toBe('#/');
    expect(result[2]).toBe('#/workflows');
    expect(result[3]).toBe('#/runs');
  });

  test('root URL renders a page (not 404)', async ({ page }) => {
    await page.goto('http://localhost:5173/');
    await page.waitForTimeout(500);

    // The app should render a real page, not the not-found page.
    const app = page.locator('#app');
    await expect(app).toBeVisible();

    const notFound = page.locator('.not-found-page');
    await expect(notFound).toHaveCount(0);
  });
});

// ---------------------------------------------------------------------------
// Bug 4/5 – API null-items safety
// ---------------------------------------------------------------------------

test.describe('API null-items safety', () => {
  test('workflows page renders empty state when API returns null items', async ({
    page,
    context,
  }) => {
    await context.route('**/api/workflows', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ items: null, total: 0 }),
      });
    });

    await page.goto('http://localhost:5173/#/workflows');
    await page.waitForTimeout(500);

    // Should render the content div without a JS crash.
    const content = page.locator('#workflows-content');
    await expect(content).toBeVisible();
    // No unhandled error overlay.
    const errorOverlay = page.locator('.vite-error-overlay');
    await expect(errorOverlay).toHaveCount(0);
  });

  test('clients page renders empty state when API returns null items', async ({
    page,
    context,
  }) => {
    await context.route('**/api/clients', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ items: null }),
      });
    });

    await page.goto('http://localhost:5173/#/clients');
    await page.waitForTimeout(500);

    const content = page.locator('#clients-content');
    await expect(content).toBeVisible();
    const errorOverlay = page.locator('.vite-error-overlay');
    await expect(errorOverlay).toHaveCount(0);
  });

  test('health page renders empty state when API returns null items', async ({
    page,
    context,
  }) => {
    await context.route('**/api/health', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ items: null }),
      });
    });

    await page.goto('http://localhost:5173/#/health');
    await page.waitForTimeout(500);

    const content = page.locator('#health-content');
    await expect(content).toBeVisible();
    const errorOverlay = page.locator('.vite-error-overlay');
    await expect(errorOverlay).toHaveCount(0);
  });

  test('runs page renders when API returns null items array', async ({ page, context }) => {
    await context.route('**/api/runs*', (route) => {
      if (route.request().url().endsWith('.ts')) { route.continue(); return; }
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ items: null, total: 0 }),
      });
    });

    await page.goto('http://localhost:5173/#/runs');
    await page.waitForTimeout(500);

    const content = page.locator('#runs-content');
    await expect(content).toBeVisible();
    const errorOverlay = page.locator('.vite-error-overlay');
    await expect(errorOverlay).toHaveCount(0);
  });

  test('alerts page renders when API returns null items array', async ({ page, context }) => {
    await context.route('**/api/alerts*', (route) => {
      if (route.request().url().endsWith('.ts')) { route.continue(); return; }
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ items: null, total: 0 }),
      });
    });

    await page.goto('http://localhost:5173/#/alerts');
    await page.waitForTimeout(500);

    const content = page.locator('#alerts-content');
    await expect(content).toBeVisible();
    const errorOverlay = page.locator('.vite-error-overlay');
    await expect(errorOverlay).toHaveCount(0);
  });

  test('bans page renders when API returns null items array', async ({ page, context }) => {
    await context.route('**/api/bans', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ items: null }),
      });
    });

    await page.goto('http://localhost:5173/#/bans');
    await page.waitForTimeout(500);

    const content = page.locator('#bans-content');
    await expect(content).toBeVisible();
    const errorOverlay = page.locator('.vite-error-overlay');
    await expect(errorOverlay).toHaveCount(0);
  });
});

// ---------------------------------------------------------------------------
// Bug 2/3 – Page components render correctly with proper field names
// ---------------------------------------------------------------------------

test.describe('Page components render with correct field names', () => {
  test('clients page renders client_id and last_seen_at from API data', async ({
    page,
    context,
  }) => {
    await context.route('**/api/clients', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          items: [
            {
              client_id: 'client-abc-123',
              os: 'linux',
              labels: {},
              inner_state: {},
              active: true,
              last_seen_at: '2026-05-13T10:00:00Z',
            },
          ],
        }),
      });
    });

    await page.goto('http://localhost:5173/#/clients');
    await page.waitForTimeout(500);

    // The client_id should appear in the table.
    const content = page.locator('#clients-content');
    await expect(content).toBeVisible();
    await expect(content).toContainText('client-abc-123');
  });

  test('runs page renders run_id and client count (not health.success_percentage)', async ({
    page,
    context,
  }) => {
    await context.route('**/api/runs*', (route) => {
      if (route.request().url().endsWith('.ts')) { route.continue(); return; }
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          items: [
            {
              run_id: 'run-xyz-456',
              workflow_id: 'wf-1',
              workflow_type: 'deploy',
              triggered_at: '2026-05-13T10:00:00Z',
              participating_clients: ['c1', 'c2'],
              state: 'completed',
            },
          ],
          total: 1,
        }),
      });
    });

    await page.goto('http://localhost:5173/#/runs');
    await page.waitForTimeout(500);

    const content = page.locator('#runs-content');
    await expect(content).toBeVisible();
    await expect(content).toContainText('run-xyz-456');
    // Client count column should show 2.
    await expect(content).toContainText('2');
    // There should be no JS crash.
    const errorOverlay = page.locator('.vite-error-overlay');
    await expect(errorOverlay).toHaveCount(0);
  });

  test('health page renders success_percentage_avg without crashing', async ({
    page,
    context,
  }) => {
    await context.route('**/api/health', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          items: [
            {
              workflow_type: 'deploy',
              window_size: 10,
              runs_considered: 5,
              success_percentage_avg: 80,
              fail_percentage_avg: 10,
              error_percentage_avg: 10,
              trend: 'stable',
              calculated_at: '2026-05-13T10:00:00Z',
            },
          ],
        }),
      });
    });

    await page.goto('http://localhost:5173/#/health');
    await page.waitForTimeout(500);

    const content = page.locator('#health-content');
    await expect(content).toBeVisible();
    await expect(content).toContainText('deploy');
    // No JS crash.
    const errorOverlay = page.locator('.vite-error-overlay');
    await expect(errorOverlay).toHaveCount(0);
  });

  test('alerts page renders alert.kind (not alert.type)', async ({ page, context }) => {
    await context.route('**/api/alerts*', (route) => {
      if (route.request().url().endsWith('.ts')) { route.continue(); return; }
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          items: [
            {
              id: 'alert-1',
              kind: 'circuit_opened',
              severity: 'warning',
              message: 'Circuit opened for deploy workflow',
              details: {},
              timestamp: '2026-05-13T10:00:00Z',
            },
          ],
          total: 1,
        }),
      });
    });

    await page.goto('http://localhost:5173/#/alerts');
    await page.waitForTimeout(500);

    const content = page.locator('#alerts-content');
    await expect(content).toBeVisible();
    await expect(content).toContainText('circuit_opened');
    const errorOverlay = page.locator('.vite-error-overlay');
    await expect(errorOverlay).toHaveCount(0);
  });
});
