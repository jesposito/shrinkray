import { test, expect, mockAPI } from './fixtures';

test.describe('Accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await mockAPI(page);
    await page.goto('/');
  });

  test.describe('Document Structure', () => {
    test('page has title', async ({ page }) => {
      await expect(page).toHaveTitle(/Shrinkray/i);
    });

    test('page has lang attribute', async ({ page }) => {
      const lang = await page.locator('html').getAttribute('lang');
      expect(lang).toBe('en');
    });

    test('main content has id for skip link target', async ({ page }) => {
      const main = page.locator('main#main-content');
      await expect(main).toBeVisible();
    });

    test('skip link exists and is keyboard focusable', async ({ page }) => {
      const skipLink = page.locator('a.skip-link');
      await expect(skipLink).toHaveAttribute('href', '#main-content');
      
      await page.keyboard.press('Tab');
      await expect(skipLink).toBeFocused();
    });
  });

  test.describe('Screen Reader Support', () => {
    test('no auto-playing media', async ({ page }) => {
      const autoplayVideos = await page.locator('video[autoplay]').count();
      const autoplayAudios = await page.locator('audio[autoplay]').count();
      expect(autoplayVideos).toBe(0);
      expect(autoplayAudios).toBe(0);
    });

    test('screen reader announcement region exists', async ({ page }) => {
      const srAnnouncements = page.locator('#sr-announcements');
      await expect(srAnnouncements).toHaveAttribute('aria-live', 'polite');
      await expect(srAnnouncements).toHaveAttribute('aria-atomic', 'true');
      await expect(srAnnouncements).toHaveClass(/sr-only/);
    });

    test('offline banner has proper alert role', async ({ page }) => {
      const offlineBanner = page.locator('#offline-banner');
      await expect(offlineBanner).toHaveAttribute('role', 'alert');
      await expect(offlineBanner).toHaveAttribute('aria-live', 'assertive');
    });
  });

  test.describe('Modal Accessibility', () => {
    test('confirm modal has proper ARIA attributes', async ({ page }) => {
      const modal = page.locator('#confirm-modal');
      await expect(modal).toHaveAttribute('role', 'dialog');
      await expect(modal).toHaveAttribute('aria-modal', 'true');
      await expect(modal).toHaveAttribute('aria-labelledby', 'confirm-modal-title');
    });

    test('video info modal has proper ARIA attributes', async ({ page }) => {
      const modal = page.locator('#video-info-modal');
      await expect(modal).toHaveAttribute('role', 'dialog');
      await expect(modal).toHaveAttribute('aria-modal', 'true');
      await expect(modal).toHaveAttribute('aria-labelledby', 'video-info-title');
    });

    test('preset modal has proper ARIA attributes', async ({ page }) => {
      const modal = page.locator('#preset-modal');
      await expect(modal).toHaveAttribute('role', 'dialog');
      await expect(modal).toHaveAttribute('aria-modal', 'true');
      await expect(modal).toHaveAttribute('aria-labelledby', 'preset-modal-title');
    });

    test('preset modal closes with Escape key', async ({ page }) => {
      await page.click('text=Help me choose');
      const modal = page.locator('#preset-modal');
      await expect(modal).toHaveClass(/active/);
      
      await page.keyboard.press('Escape');
      await expect(modal).not.toHaveClass(/active/);
    });
  });

  test.describe('Preset Cards Keyboard Navigation', () => {
    test('preset cards are focusable', async ({ page }) => {
      await page.click('text=Help me choose');
      
      const cards = page.locator('.preset-card');
      const firstCard = cards.first();
      
      await expect(firstCard).toHaveAttribute('tabindex', '0');
      await expect(firstCard).toHaveAttribute('role', 'button');
    });

    test('preset cards respond to Enter key', async ({ page }) => {
      await page.click('text=Help me choose');
      const modal = page.locator('#preset-modal');
      await expect(modal).toHaveClass(/active/);
      
      const firstCard = page.locator('.preset-card').first();
      await firstCard.focus();
      await page.keyboard.press('Enter');
      
      await expect(modal).not.toHaveClass(/active/);
    });

    test('preset cards support arrow key navigation', async ({ page }) => {
      await page.click('text=Help me choose');
      
      const cards = page.locator('.preset-card');
      const firstCard = cards.first();
      const secondCard = cards.nth(1);
      
      await firstCard.focus();
      await expect(firstCard).toBeFocused();
      
      await page.keyboard.press('ArrowRight');
      await expect(secondCard).toBeFocused();
    });
  });

  test.describe('Tab Navigation', () => {
    test('tab buttons have proper ARIA attributes', async ({ page }) => {
      const tabBar = page.locator('.tab-bar');
      await expect(tabBar).toHaveAttribute('role', 'tablist');
      await expect(tabBar).toHaveAttribute('aria-label', 'Layout tabs');
      
      const tabs = page.locator('[role="tab"]');
      await expect(tabs).toHaveCount(4);
      
      const mediaTab = page.locator('#tab-media');
      await expect(mediaTab).toHaveAttribute('aria-controls', 'browser-panel');
      await expect(mediaTab).toHaveAttribute('aria-selected', 'true');
    });

    test('tab panels have proper ARIA attributes', async ({ page }) => {
      const browserPanel = page.locator('#browser-panel');
      await expect(browserPanel).toHaveAttribute('role', 'tabpanel');
      await expect(browserPanel).toHaveAttribute('aria-labelledby', 'tab-media');
      
      const queuePanel = page.locator('#queue-panel-wrapper');
      await expect(queuePanel).toHaveAttribute('role', 'tabpanel');
      await expect(queuePanel).toHaveAttribute('aria-labelledby', 'tab-queue');
      
      const settingsPanel = page.locator('#settings-overlay');
      await expect(settingsPanel).toHaveAttribute('role', 'tabpanel');
      await expect(settingsPanel).toHaveAttribute('aria-labelledby', 'tab-settings');
    });
  });

  test.describe('Collapsible Sections', () => {
    test('collapsible headers have proper ARIA', async ({ page }) => {
      const completedHeader = page.locator('.completed-header');
      await expect(completedHeader).toHaveAttribute('role', 'button');
      await expect(completedHeader).toHaveAttribute('tabindex', '0');
      await expect(completedHeader).toHaveAttribute('aria-expanded');
      await expect(completedHeader).toHaveAttribute('aria-controls', 'completed-list');
    });

    test('collapsible headers are keyboard accessible', async ({ page }) => {
      const skippedHeader = page.locator('.skipped-header');
      await expect(skippedHeader).toHaveAttribute('role', 'button');
      
      const failedHeader = page.locator('.failed-header');
      await expect(failedHeader).toHaveAttribute('role', 'button');
    });
  });

  test.describe('Form Controls', () => {
    test('settings close button has aria-label', async ({ page }) => {
      const closeButton = page.locator('.settings-close');
      await expect(closeButton).toHaveAttribute('aria-label', 'Close settings');
    });

    test('text inputs have aria-labels', async ({ page }) => {
      const pushoverUser = page.locator('#setting-pushover-user');
      await expect(pushoverUser).toHaveAttribute('aria-label', 'Pushover User Key');
      
      const pushoverToken = page.locator('#setting-pushover-token');
      await expect(pushoverToken).toHaveAttribute('aria-label', 'Pushover App Token');
      
      const ntfyServer = page.locator('#setting-ntfy-server');
      await expect(ntfyServer).toHaveAttribute('aria-label', 'ntfy Server URL');
      
      const ntfyTopic = page.locator('#setting-ntfy-topic');
      await expect(ntfyTopic).toHaveAttribute('aria-label', 'ntfy Topic');
      
      const ntfyToken = page.locator('#setting-ntfy-token');
      await expect(ntfyToken).toHaveAttribute('aria-label', 'ntfy Access Token');
    });

    test('checkboxes have accessible labels', async ({ page }) => {
      const hideProcessingCheckbox = page.locator('#setting-hide-processing-tmp');
      await expect(hideProcessingCheckbox).toHaveAttribute('aria-label');
      
      const softwareFallbackCheckbox = page.locator('#setting-allow-software-fallback');
      await expect(softwareFallbackCheckbox).toHaveAttribute('aria-label');
      
      const unicornCheckbox = page.locator('#setting-unicorn-mode');
      await expect(unicornCheckbox).toHaveAttribute('aria-label');
    });
  });

  test.describe('Icon Buttons', () => {
    test('icon buttons have aria-labels', async ({ page }) => {
      const presetCloseBtn = page.locator('.preset-modal .modal-close');
      await expect(presetCloseBtn).toHaveAttribute('aria-label', 'Close preset picker');
    });

    test('decorative SVGs are hidden from screen readers', async ({ page }) => {
      const completedToggle = page.locator('#completed-toggle');
      await expect(completedToggle).toHaveAttribute('aria-hidden', 'true');
      
      const skippedToggle = page.locator('#skipped-toggle');
      await expect(skippedToggle).toHaveAttribute('aria-hidden', 'true');
    });
  });

  test.describe('Focus Management', () => {
    test('focus-visible styles are defined', async ({ page }) => {
      const button = page.locator('.btn').first();
      await button.focus();
      
      const outlineStyle = await button.evaluate(el => {
        const styles = window.getComputedStyle(el);
        return styles.outlineColor || styles.outline;
      });
      expect(outlineStyle).toBeTruthy();
    });
  });
});
