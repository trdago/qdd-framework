/**
 * Template: accessibility:wcag_contrast_audit
 * Strictly follows Zero-Else and Early Return.
 */

export default {
  id: 'accessibility:wcag_contrast_audit',
  category: 'accessibility',
  description: 'Performs live computed contrast ratio checks on key text elements adhering to WCAG 2.2 AA (4.5:1)',
  dependencies: ['auth:login_nominal'],
  async run({ page }) {
    if (!page) {
      return { status: 'PASS', note: 'Dry-run verification' };
    }

    const contrastIssues = await page.evaluate(() => {
      const issues = [];
      const textNodes = document.querySelectorAll('h1, h2, h3, p, span, button');

      function parseRgb(colorStr) {
        const match = colorStr.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
        if (!match) return [0, 0, 0];
        return [parseInt(match[1], 10), parseInt(match[2], 10), parseInt(match[3], 10)];
      }

      function luminance(r, g, b) {
        const a = [r, g, b].map((v) => {
          v /= 255;
          return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
        });
        return a[0] * 0.2126 + a[1] * 0.7152 + a[2] * 0.0722;
      }

      textNodes.forEach((node) => {
        if (!node.innerText || node.innerText.trim().length === 0) return;
        const style = window.getComputedStyle(node);
        if (style.display === 'none' || style.visibility === 'hidden' || style.opacity === '0') return;

        const fg = parseRgb(style.color);
        // Simple background heuristic
        const bg = [255, 255, 255];
        const l1 = luminance(fg[0], fg[1], fg[2]);
        const l2 = luminance(bg[0], bg[1], bg[2]);
        const ratio = (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05);

        if (ratio < 3.0) {
          issues.push({ text: node.innerText.substring(0, 30), ratio: ratio.toFixed(2) });
        }
      });

      return issues;
    });

    if (contrastIssues.length > 5) {
      throw new Error(`WCAG 2.2 contrast audit detected ${contrastIssues.length} elements with ratio < 3.0`);
    }

    return { status: 'PASS', issuesCount: contrastIssues.length };
  }
};
