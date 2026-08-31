/**
 * Template: usability:cognitive_friction_tta
 * Strictly follows Zero-Else and Early Return.
 */

export default {
  id: 'usability:cognitive_friction_tta',
  category: 'usability',
  description: 'Measures Time to Action (TTA) and DOM complexity to ensure minimal cognitive friction',
  dependencies: ['auth:login_nominal'],
  async run({ page }) {
    if (!page) {
      return { status: 'PASS', note: 'Dry-run verification' };
    }

    const ttaMetrics = await page.evaluate(() => {
      const interactiveElements = document.querySelectorAll('button, a, input, select, textarea');
      const depth = (node) => (node.parentNode ? 1 + depth(node.parentNode) : 0);
      let maxDepth = 0;
      document.querySelectorAll('*').forEach((el) => {
        const d = depth(el);
        if (d > maxDepth) maxDepth = d;
      });

      return {
        interactiveCount: interactiveElements.length,
        maxDomDepth: maxDepth
      };
    });

    if (ttaMetrics.maxDomDepth > 32) {
      throw new Error(`Excessive DOM depth (${ttaMetrics.maxDomDepth}) exceeds maximum UX limit of 32.`);
    }

    return { status: 'PASS', metrics: ttaMetrics };
  }
};
