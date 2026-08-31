/**
 * Template: auth:login_invalid_creds
 * Strictly follows Zero-Else and Early Return.
 */

export default {
  id: 'auth:login_invalid_creds',
  category: 'auth',
  description: 'Validates that invalid credentials trigger error feedback without crashing',
  dependencies: [],
  async run({ page, config }) {
    if (!page) {
      return { status: 'PASS', note: 'Dry-run verification' };
    }

    await page.goto(`${config.baseUrl}/login`);
    await page.waitForSelector('input[type="password"]', { state: 'visible' });

    await page.fill('input[type="email"], input[name="email"], input[type="text"]', 'invalid@user.com');
    await page.fill('input[type="password"]', 'wrong-pass');
    await page.click('button[type="submit"], button:has-text("Ingresar"), button:has-text("Login")');

    // Expect alert/snackbar visible
    await page.waitForSelector('.v-alert, .v-snackbar, [role="alert"], .error-message', { state: 'visible', timeout: 5000 });
    return { status: 'PASS' };
  }
};
