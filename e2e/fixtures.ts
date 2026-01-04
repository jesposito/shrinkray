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

  // Navigation
  async goto() {
    await this.page.goto('/');
    await this.page.waitForLoadState('networkidle');
  }

  // Selectors
  get fileBrowser() {
    return this.page.locator('#file-browser, .file-browser, [data-testid="file-browser"]');
  }

  get presetDropdown() {
    return this.page.locator('select#preset, select[name="preset"], #preset-select');
  }

  get startButton() {
    return this.page.locator('button:has-text("Start"), button:has-text("Transcode"), #start-btn');
  }

  get settingsButton() {
    return this.page.locator('button[aria-label="Settings"], .settings-btn, button:has-text("Settings"), [data-testid="settings"]');
  }

  get settingsPanel() {
    return this.page.locator('#settings-panel, .settings-panel, [data-testid="settings-panel"]');
  }

  get jobQueue() {
    return this.page.locator('#job-queue, .job-queue, [data-testid="job-queue"]');
  }

  get helpMeChooseLink() {
    return this.page.locator('a:has-text("Help me choose"), button:has-text("Help me choose"), .help-choose');
  }

  get helpModal() {
    return this.page.locator('#help-modal, .help-modal, [data-testid="help-modal"]');
  }

  // Actions
  async selectPreset(presetName: string) {
    await this.presetDropdown.selectOption({ label: new RegExp(presetName, 'i') });
  }

  async openSettings() {
    // Try multiple possible settings triggers
    const settingsBtn = this.page.locator('[class*="settings"], button:has-text("Settings"), .gear-icon, [aria-label*="settings" i]').first();
    if (await settingsBtn.isVisible()) {
      await settingsBtn.click();
    }
  }

  async closeSettings() {
    // Try clicking outside or close button
    const closeBtn = this.page.locator('.settings-panel .close, button:has-text("Close"), [aria-label="Close"]');
    if (await closeBtn.isVisible()) {
      await closeBtn.click();
    } else {
      await this.page.keyboard.press('Escape');
    }
  }

  async waitForSSEConnection() {
    // Wait for SSE stream to be established
    await this.page.waitForFunction(() => {
      return (window as any).eventSource?.readyState === 1;
    }, { timeout: 10000 }).catch(() => {
      // SSE may not be exposed globally, that's ok
    });
  }

  async getJobCount(): Promise<number> {
    const jobs = await this.page.locator('.job-item, .job-card, [data-testid="job"]').count();
    return jobs;
  }

  async toggleSetting(settingName: string, enabled: boolean) {
    const toggle = this.page.locator(`input[type="checkbox"]:near(:text("${settingName}"))`).first();
    const isChecked = await toggle.isChecked();
    if (isChecked !== enabled) {
      await toggle.click();
    }
  }

  // Assertions helpers
  async expectPresetSelected(presetName: string) {
    const selected = await this.presetDropdown.inputValue();
    expect(selected).toContain(presetName.toLowerCase());
  }

  async expectVisible(element: ReturnType<Page['locator']>) {
    await expect(element).toBeVisible({ timeout: 5000 });
  }

  async expectHidden(element: ReturnType<Page['locator']>) {
    await expect(element).toBeHidden({ timeout: 5000 });
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
}
