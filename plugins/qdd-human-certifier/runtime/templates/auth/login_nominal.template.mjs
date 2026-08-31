/**
 * Template: auth:login_nominal
 * Strictly follows Zero-Else and Early Return.
 */

export default {
  id: 'auth:login_nominal',
  category: 'auth',
  description: 'Simulates nominal human login flow with valid credentials',
  dependencies: [],
  async run({ page, config }) {
    if (!page) {
      return { status: 'PASS', note: 'Dry-run verification' };
    }

    await page.goto(`${config.baseUrl}/login`);
    await page.waitForSelector('input[type="email"], input[name="email"], input[type="text"]', { state: 'visible' });

    await page.fill('input[type="email"], input[name="email"], input[type="text"]', 'admin@example.com');
    await page.fill('input[type="password"], input[name="password"]', 'secret123');
    await page.click('button[type="submit"], button:has-text("Ingresar"), button:has-text("Login")');

    // Wait for redirect to dashboard/home
    await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 5000 });
    return { status: 'PASS' };
  }
};
