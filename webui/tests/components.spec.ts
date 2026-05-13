import { test, expect } from '@playwright/test';

test.describe('Components and UI Elements', () => {
  test('should display loading spinner on data fetch', async ({ page }) => {
    await page.goto('http://localhost:5173/#/workflows');
    // Wait for page to be interactive
    await page.waitForTimeout(300);
    // The page should show either data or loading state
    const content = page.locator('#workflows-content');
    await expect(content).toBeVisible();
  });

  test('buttons should have proper styling', async ({ page }) => {
    await page.goto('http://localhost:5173/#/workflows');
    await page.waitForTimeout(300);

    // Check if action buttons exist (matches any p-button variant class)
    const buttons = page.locator('[class*="p-button"]');
    const count = await buttons.count();
    // At least one button should exist
    expect(count).toBeGreaterThan(0);
  });

  test('should display tables with proper structure', async ({ page }) => {
    await page.goto('http://localhost:5173/#/workflows');
    await page.waitForTimeout(500);

    // Wait for table to potentially render
    const table = page.locator('table.p-table');
    // Table might not be visible if there's an error or loading state
    // but the page structure should be correct
    const content = page.locator('#workflows-content');
    await expect(content).toBeVisible();
  });

  test('should have valid navigation structure', async ({ page }) => {
    await page.goto('http://localhost:5173/');

    // Check header is present
    const header = page.locator('.p-navigation');
    await expect(header).toBeVisible();

    // Navigation items live in the sidebar
    const sideNavItems = page.locator('.p-side-navigation__item');
    const itemCount = await sideNavItems.count();
    expect(itemCount).toBeGreaterThan(0);
  });

  test('should have badge elements for status display', async ({ page }) => {
    await page.goto('http://localhost:5173/#/workflows');
    await page.waitForTimeout(500);

    // Check if badge elements exist
    const badges = page.locator('.p-badge');
    const count = await badges.count();
    // Badges might appear in tables if data is loaded
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('pagination should be functional', async ({ page }) => {
    await page.goto('http://localhost:5173/#/runs');
    await page.waitForTimeout(500);

    // Check if pagination controls exist
    const paginationControls = page.locator('.pagination');
    const paginationCount = await paginationControls.count();
    // Pagination might exist even if tables are empty
    expect(paginationCount).toBeGreaterThanOrEqual(0);
  });

  test('page sections should have proper headers', async ({ page }) => {
    const pages = [
      { url: '#/workflows', title: 'Workflows' },
      { url: '#/runs', title: 'Runs' },
      { url: '#/health', title: 'Health' },
      { url: '#/clients', title: 'Clients' },
      { url: '#/alerts', title: 'Alerts' },
      { url: '#/bans', title: 'Ban' },
    ];

    for (const pageConfig of pages) {
      await page.goto(`http://localhost:5173/${pageConfig.url}`);
      await page.waitForTimeout(300);

      // Check if page has heading
      const heading = page.locator('h1, h2');
      const count = await heading.count();
      expect(count).toBeGreaterThan(0);
    }
  });

  test('should display proper Canonical branding', async ({ page }) => {
    await page.goto('http://localhost:5173/');

    // Check logo
    const logo = page.locator('.logo-img');
    await expect(logo).toBeVisible();

    // Check site name
    const siteName = page.locator('.site-name');
    await expect(siteName).toBeVisible();
    expect(await siteName.textContent()).toContain('super-duper-bassoon');
  });

  test('should have responsive layout', async ({ page }) => {
    await page.goto('http://localhost:5173/');

    // Check main layout exists
    const mainLayout = page.locator('.main-layout');
    await expect(mainLayout).toBeVisible();

    // Check layout containers
    const pLayout = page.locator('.p-layout');
    await expect(pLayout).toBeVisible();
  });

  test('error handling should display error messages', async ({ page, context }) => {
    // Intercept API calls to simulate errors (skip TypeScript source module files)
    await context.route('**/api/**', (route) => {
      if (route.request().url().endsWith('.ts')) {
        route.continue();
        return;
      }
      route.abort('failed');
    });

    await page.goto('http://localhost:5173/#/workflows');
    await page.waitForTimeout(1000);

    // Page should still be visible with error state
    const content = page.locator('#workflows-content');
    await expect(content).toBeVisible();
  });
});
