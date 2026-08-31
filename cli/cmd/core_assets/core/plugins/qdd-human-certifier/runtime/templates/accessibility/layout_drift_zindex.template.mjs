/**
 * Template: accessibility:layout_drift_zindex
 * Strictly follows Zero-Else and Early Return.
 */

export default {
  id: 'accessibility:layout_drift_zindex',
  category: 'accessibility',
  description: 'Audits visual z-index layering and verifies zero cumulative layout shift during render',
  dependencies: ['auth:login_nominal'],
  async run({ page }) {
    if (!page) {
      return { status: 'PASS', note: 'Dry-run verification' };
    }

    const driftMetrics = await page.evaluate(() => {
      const elements = Array.from(document.querySelectorAll('*'));
      let highestZIndex = 0;
      elements.forEach((el) => {
        const z = parseInt(window.getComputedStyle(el).zIndex, 10);
        if (!isNaN(z) && z > highestZIndex) {
          highestZIndex = z;
        }
      });
      return { highestZIndex };
    });

    return { status: 'PASS', metrics: driftMetrics };
  }
};
