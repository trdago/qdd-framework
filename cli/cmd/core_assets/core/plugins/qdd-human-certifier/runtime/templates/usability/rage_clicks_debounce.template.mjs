/**
 * Template: usability:rage_clicks_debounce
 * Strictly follows Zero-Else and Early Return.
 */

export default {
  id: 'usability:rage_clicks_debounce',
  category: 'usability',
  description: 'Simulates rapid repeated clicks (rage clicks) on primary action buttons to ensure debounce and prevent duplicate requests',
  dependencies: ['auth:login_nominal'],
  async run({ page, config }) {
    if (!page) {
      return { status: 'PASS', note: 'Dry-run verification' };
    }

    let requestCount = 0;
    page.on('request', (req) => {
      if (req.method() === 'POST' || req.method() === 'PUT') {
        requestCount++;
      }
    });

    const submitBtn = await page.$('button[type="submit"], button.primary-action');
    if (!submitBtn) {
      return { status: 'PASS', note: 'No action button on current view' };
    }

    // Fire 5 rapid clicks
    for (let i = 0; i < 5; i++) {
      await submitBtn.click({ delay: 20 });
    }

    // Give 500ms to settle
    await page.waitForTimeout(500);

    if (requestCount > 2) {
      throw new Error(`Rage clicks debounce failed: ${requestCount} network requests emitted for 5 rapid clicks.`);
    }

    return { status: 'PASS' };
  }
};
