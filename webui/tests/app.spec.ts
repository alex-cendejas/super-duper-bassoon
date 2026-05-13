import { test, expect } from '@playwright/test';

test.describe('super-duper-bassoon WebUI', () => {
  test('should load the application', async ({ page }) => {
    await page.goto('http://localhost:5173/');
    await expect(page.locator('#app')).toBeVisible();
  });

  test('should display the header with logo', async ({ page }) => {
    await page.goto('http://localhost:5173/');
    // Check if header is present
    await expect(page.locator('.p-navigation')).toBeVisible();
    // Check if logo is present
    const logo = page.locator('.logo-img');
    await expect(logo).toBeVisible();
  });

  test('should display the sidebar', async ({ page }) => {
    await page.goto('http://localhost:5173/');
    await expect(page.locator('.p-side-navigation')).toBeVisible();
  });

  test('should navigate to workflows page', async ({ page }) => {
    await page.goto('http://localhost:5173/');
    await page.click('text=Workflows');
    // Wait for page to load
    await page.waitForTimeout(500);
    // Check if we're on the workflows page
    expect(page.url()).toContain('#/workflows');
    await expect(page.locator('.workflows-page')).toBeVisible();
  });

  test('should navigate to runs page', async ({ page }) => {
    await page.goto('http://localhost:5173/');
    await page.click('text=Runs');
    await page.waitForTimeout(500);
    expect(page.url()).toContain('#/runs');
    await expect(page.locator('.runs-page')).toBeVisible();
  });

  test('should navigate to health page', async ({ page }) => {
    await page.goto('http://localhost:5173/');
    await page.click('text=Health');
    await page.waitForTimeout(500);
    expect(page.url()).toContain('#/health');
    await expect(page.locator('.health-page')).toBeVisible();
  });

  test('should navigate to clients page', async ({ page }) => {
    await page.goto('http://localhost:5173/');
    await page.click('text=Clients');
    await page.waitForTimeout(500);
    expect(page.url()).toContain('#/clients');
    await expect(page.locator('.clients-page')).toBeVisible();
  });

  test('should navigate to alerts page', async ({ page }) => {
    await page.goto('http://localhost:5173/');
    await page.click('text=Alerts');
    await page.waitForTimeout(500);
    expect(page.url()).toContain('#/alerts');
    await expect(page.locator('.alerts-page')).toBeVisible();
  });

  test('should navigate to bans page', async ({ page }) => {
    await page.goto('http://localhost:5173/');
    await page.click('text=Bans');
    await page.waitForTimeout(500);
    expect(page.url()).toContain('#/bans');
    await expect(page.locator('.bans-page')).toBeVisible();
  });

  test('should display 404 page for invalid routes', async ({ page }) => {
    await page.goto('http://localhost:5173/#/invalid-route');
    await page.waitForTimeout(500);
    await expect(page.locator('.not-found-page')).toBeVisible();
    await expect(page.locator('text=404')).toBeVisible();
  });

  test('should toggle sidebar visibility', async ({ page }) => {
    await page.goto('http://localhost:5173/');
    const sidebar = page.locator('.p-side-navigation');
    const toggleBtn = page.locator('#toggle-sidebar');

    // Initial state - sidebar should be visible
    await expect(sidebar).toBeVisible();

    // Click toggle button to hide sidebar
    await toggleBtn.click();
    await page.waitForTimeout(300);
    await expect(sidebar).toHaveClass(/is-hidden/);

    // Click toggle button to show sidebar
    await toggleBtn.click();
    await page.waitForTimeout(300);
    await expect(sidebar).not.toHaveClass(/is-hidden/);
  });

  test('should have working navigation links in sidebar', async ({ page }) => {
    await page.goto('http://localhost:5173/');

    const links = [
      { text: 'Workflows', path: '#/workflows' },
      { text: 'Runs', path: '#/runs' },
      { text: 'Health', path: '#/health' },
      { text: 'Clients', path: '#/clients' },
      { text: 'Alerts', path: '#/alerts' },
      { text: 'Bans', path: '#/bans' },
    ];

    for (const link of links) {
      await page.click(`.p-side-navigation__link:has-text("${link.text}")`);
      await page.waitForTimeout(300);
      expect(page.url()).toContain(link.path);
    }
  });
});
