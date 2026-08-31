/**
 * QDD Browser Context & Human Interaction Harness
 * Strictly follows Zero-Else and Early Return.
 */

export class BrowserHarness {
  constructor(options = {}) {
    this.baseUrl = options.baseUrl || 'http://localhost:5173';
    this.headless = options.headless !== false;
    this.browser = null;
    this.desktopContext = null;
    this.mobileContext = null;
    this.desktopPage = null;
    this.mobilePage = null;
    this.apiMocks = new Map();
  }

  async initialize(playwrightChromium) {
    if (!playwrightChromium) {
      throw new Error('Chromium instance is required to initialize BrowserHarness');
    }

    this.browser = await playwrightChromium.launch({
      headless: this.headless,
      args: ['--no-sandbox', '--disable-setuid-sandbox']
    });

    // 1. Desktop Context (1440x900)
    this.desktopContext = await this.browser.newContext({
      viewport: { width: 1440, height: 900 },
      deviceScaleFactor: 1,
      hasTouch: false
    });
    this.desktopPage = await this.desktopContext.newPage();

    // 2. Mobile Context (iPhone 13: 390x844)
    this.mobileContext = await this.browser.newContext({
      viewport: { width: 390, height: 844 },
      deviceScaleFactor: 3,
      isMobile: true,
      hasTouch: true,
      userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1'
    });
    this.mobilePage = await this.mobileContext.newPage();

    this.setupNetworkInterceptor(this.desktopPage);
    this.setupNetworkInterceptor(this.mobilePage);
  }

  registerApiMock(urlPattern, mockResponse) {
    this.apiMocks.set(urlPattern, mockResponse);
  }

  setupNetworkInterceptor(page) {
    if (!page) {
      return;
    }

    page.route('**/*', async (route) => {
      const url = route.request().url();

      for (const [pattern, mockData] of this.apiMocks.entries()) {
        if (!url.includes(pattern)) {
          continue;
        }

        const isFunc = typeof mockData === 'function';
        const body = isFunc ? mockData(route.request()) : mockData;
        const status = (body && body.status) || 200;
        const jsonPayload = (body && body.json) !== undefined ? body.json : body;

        await route.fulfill({
          status: status,
          contentType: 'application/json',
          body: JSON.stringify(jsonPayload)
        });
        return;
      }

      await route.continue();
    });
  }

  async humanType(page, selector, text, delayMs = 40) {
    if (!page || !selector) {
      return;
    }
    await page.waitForSelector(selector, { state: 'visible' });
    await page.click(selector);
    await page.fill(selector, '');
    for (const char of text) {
      await page.type(selector, char, { delay: delayMs });
    }
  }

  async captureScreenshot(page, outputPath) {
    if (!page) {
      return;
    }
    await page.screenshot({ path: outputPath, fullPage: false });
  }

  async close() {
    if (this.desktopContext) {
      await this.desktopContext.close();
    }
    if (this.mobileContext) {
      await this.mobileContext.close();
    }
    if (this.browser) {
      await this.browser.close();
    }
  }
}
