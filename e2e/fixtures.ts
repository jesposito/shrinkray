import { test as base, expect, Page } from '@playwright/test';

/**
 * Custom test fixtures for Shrinkray E2E tests
 */

// Extend base test with Shrinkray-specific helpers
export const test = base.extend<{
  shrinkray: ShrinkrayPage;
}>({
  shrinkray: async ({ page }, use) => {
    const shrinkray = new ShrinkrayPage(page);
    await use(shrinkray);
  },
});

export { expect };

/**
 * Page Object Model for Shrinkray UI
 */
export class ShrinkrayPage {
  constructor(public page: Page) {}

  // Navigation - don't wait for networkidle due to SSE
  async goto() {
    await this.page.goto('/');
    await this.page.waitForLoadState('domcontentloaded');
    // Wait for initial data load
    await this.page.waitForTimeout(500);
  }

  // Selectors - using specific IDs from actual UI
  get presetDropdown() {
    return this.page.locator('#preset');
  }

  get startButton() {
    return this.page.locator('#start-btn');
  }

  get settingsPanel() {
    return this.page.locator('#settings-panel');
  }

  get queuePanel() {
    return this.page.locator('#queue-panel');
  }

  get activePanel() {
    return this.page.locator('#active-panel');
  }

  get fileBrowser() {
    return this.page.locator('#file-browser');
  }

  // Actions
  async selectPreset(presetId: string) {
    await this.presetDropdown.selectOption(presetId);
  }

  async openSettings() {
    const settingsBtn = this.page.locator('#settings-btn');
    await settingsBtn.click();
    await expect(this.settingsPanel).toBeVisible();
  }

  async closeSettings() {
    await this.page.keyboard.press('Escape');
  }

  async getJobCount(): Promise<number> {
    const jobs = await this.page.locator('.queue-item').count();
    return jobs;
  }
}

/**
 * Mock API responses for testing without a real backend
 */
export async function mockAPI(page: Page) {
  // Mock presets endpoint
  await page.route('**/api/presets', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { id: 'compress-hevc', name: 'Smaller files — HEVC', description: 'Widely compatible' },
        { id: 'compress-av1', name: 'Smaller files — AV1', description: 'Best quality per MB' },
        { id: '1080p', name: 'Reduce to 1080p — HEVC', description: 'Downscale to Full HD' },
        { id: '720p', name: 'Reduce to 720p — HEVC', description: 'Maximum compatibility' },
      ]),
    });
  });

  // Mock config endpoint
  await page.route('**/api/config', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          media_path: '/media',
          temp_path: '',
          original_handling: 'replace',
          workers: 1,
          allow_software_fallback: false,
        }),
      });
    } else {
      await route.fulfill({ status: 200, body: '{}' });
    }
  });

  // Mock jobs endpoint
  await page.route('**/api/jobs', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    } else {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ id: 'test-job-1', status: 'pending' }),
      });
    }
  });

  // Mock file browser
  await page.route('**/api/browse**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        path: '/media',
        directories: [
          { name: 'Movies', path: '/media/Movies' },
          { name: 'TV Shows', path: '/media/TV Shows' },
        ],
        files: [
          { name: 'sample.mkv', path: '/media/sample.mkv', size: 1073741824 },
          { name: 'test.mp4', path: '/media/test.mp4', size: 524288000 },
        ],
      }),
    });
  });

  // Mock SSE stream - return empty and close immediately
  await page.route('**/api/jobs/stream', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'text/event-stream',
      body: 'data: []\n\n',
    });
  });
}
