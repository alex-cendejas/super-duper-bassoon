# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: components.spec.ts >> Components and UI Elements >> error handling should display error messages
- Location: tests/components.spec.ts:117:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('#workflows-content')
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 5000ms
  - waiting for locator('#workflows-content')

```

# Test source

```ts
  28  |     // Wait for table to potentially render
  29  |     const table = page.locator('table.p-table');
  30  |     // Table might not be visible if there's an error or loading state
  31  |     // but the page structure should be correct
  32  |     const content = page.locator('#workflows-content');
  33  |     await expect(content).toBeVisible();
  34  |   });
  35  | 
  36  |   test('should have valid navigation structure', async ({ page }) => {
  37  |     await page.goto('http://localhost:5173/');
  38  | 
  39  |     // Check header is present
  40  |     const header = page.locator('.p-navigation');
  41  |     await expect(header).toBeVisible();
  42  | 
  43  |     // Navigation items live in the sidebar
  44  |     const sideNavItems = page.locator('.p-side-navigation__item');
  45  |     const itemCount = await sideNavItems.count();
  46  |     expect(itemCount).toBeGreaterThan(0);
  47  |   });
  48  | 
  49  |   test('should have badge elements for status display', async ({ page }) => {
  50  |     await page.goto('http://localhost:5173/#/workflows');
  51  |     await page.waitForTimeout(500);
  52  | 
  53  |     // Check if badge elements exist
  54  |     const badges = page.locator('.p-badge');
  55  |     const count = await badges.count();
  56  |     // Badges might appear in tables if data is loaded
  57  |     expect(count).toBeGreaterThanOrEqual(0);
  58  |   });
  59  | 
  60  |   test('pagination should be functional', async ({ page }) => {
  61  |     await page.goto('http://localhost:5173/#/runs');
  62  |     await page.waitForTimeout(500);
  63  | 
  64  |     // Check if pagination controls exist
  65  |     const paginationControls = page.locator('.pagination');
  66  |     const paginationCount = await paginationControls.count();
  67  |     // Pagination might exist even if tables are empty
  68  |     expect(paginationCount).toBeGreaterThanOrEqual(0);
  69  |   });
  70  | 
  71  |   test('page sections should have proper headers', async ({ page }) => {
  72  |     const pages = [
  73  |       { url: '#/workflows', title: 'Workflows' },
  74  |       { url: '#/runs', title: 'Runs' },
  75  |       { url: '#/health', title: 'Health' },
  76  |       { url: '#/clients', title: 'Clients' },
  77  |       { url: '#/alerts', title: 'Alerts' },
  78  |       { url: '#/bans', title: 'Ban' },
  79  |     ];
  80  | 
  81  |     for (const pageConfig of pages) {
  82  |       await page.goto(`http://localhost:5173/${pageConfig.url}`);
  83  |       await page.waitForTimeout(300);
  84  | 
  85  |       // Check if page has heading
  86  |       const heading = page.locator('h1, h2');
  87  |       const count = await heading.count();
  88  |       expect(count).toBeGreaterThan(0);
  89  |     }
  90  |   });
  91  | 
  92  |   test('should display proper Canonical branding', async ({ page }) => {
  93  |     await page.goto('http://localhost:5173/');
  94  | 
  95  |     // Check logo
  96  |     const logo = page.locator('.logo-img');
  97  |     await expect(logo).toBeVisible();
  98  | 
  99  |     // Check site name
  100 |     const siteName = page.locator('.site-name');
  101 |     await expect(siteName).toBeVisible();
  102 |     expect(await siteName.textContent()).toContain('super-duper-bassoon');
  103 |   });
  104 | 
  105 |   test('should have responsive layout', async ({ page }) => {
  106 |     await page.goto('http://localhost:5173/');
  107 | 
  108 |     // Check main layout exists
  109 |     const mainLayout = page.locator('.main-layout');
  110 |     await expect(mainLayout).toBeVisible();
  111 | 
  112 |     // Check layout containers
  113 |     const pLayout = page.locator('.p-layout');
  114 |     await expect(pLayout).toBeVisible();
  115 |   });
  116 | 
  117 |   test('error handling should display error messages', async ({ page, context }) => {
  118 |     // Intercept API calls to simulate errors
  119 |     await context.route('**/api/**', (route) => {
  120 |       route.abort('failed');
  121 |     });
  122 | 
  123 |     await page.goto('http://localhost:5173/#/workflows');
  124 |     await page.waitForTimeout(1000);
  125 | 
  126 |     // Page should still be visible with error state
  127 |     const content = page.locator('#workflows-content');
> 128 |     await expect(content).toBeVisible();
      |                           ^ Error: expect(locator).toBeVisible() failed
  129 |   });
  130 | });
  131 | 
```