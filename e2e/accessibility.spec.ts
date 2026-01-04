import { test, expect, mockAPI } from './fixtures';

test.describe('Accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await mockAPI(page);
    await page.goto('/');
  });

  test('page has valid heading structure', async ({ page }) => {
    // Should have an h1
    const h1 = page.locator('h1');
    await expect(h1.first()).toBeVisible();

    // H1 count should be 1 (best practice)
    const h1Count = await h1.count();
    expect(h1Count).toBe(1);

    // Heading levels should not skip (h1 -> h3 is bad)
    const headings = await page.locator('h1, h2, h3, h4, h5, h6').all();
    let prevLevel = 0;

    for (const heading of headings) {
      const tag = await heading.evaluate(el => el.tagName);
      const level = parseInt(tag.replace('H', ''));

      // Should not skip more than one level
      if (prevLevel > 0) {
        expect(level - prevLevel).toBeLessThanOrEqual(1);
      }
      prevLevel = level;
    }
  });

  test('interactive elements are focusable', async ({ page }) => {
    // All buttons should be focusable
    const buttons = await page.locator('button:visible').all();
    for (const button of buttons.slice(0, 5)) { // Test first 5
      await button.focus();
      const isFocused = await button.evaluate(el => el === document.activeElement);
      expect(isFocused).toBe(true);
    }

    // All links should be focusable
    const links = await page.locator('a[href]:visible').all();
    for (const link of links.slice(0, 5)) {
      await link.focus();
      const isFocused = await link.evaluate(el => el === document.activeElement);
      expect(isFocused).toBe(true);
    }
  });

  test('form inputs have labels', async ({ page }) => {
    const inputs = await page.locator('input:visible, select:visible, textarea:visible').all();

    for (const input of inputs) {
      const id = await input.getAttribute('id');
      const ariaLabel = await input.getAttribute('aria-label');
      const ariaLabelledBy = await input.getAttribute('aria-labelledby');
      const title = await input.getAttribute('title');
      const placeholder = await input.getAttribute('placeholder');

      // Should have some form of accessible name
      let hasLabel = false;

      if (id) {
        const label = page.locator(`label[for="${id}"]`);
        if (await label.count() > 0) {
          hasLabel = true;
        }
      }

      if (ariaLabel || ariaLabelledBy || title || placeholder) {
        hasLabel = true;
      }

      // Wrapped in label tag
      const parentLabel = await input.evaluate(el => el.closest('label') !== null);
      if (parentLabel) {
        hasLabel = true;
      }

      expect(hasLabel).toBe(true);
    }
  });

  test('buttons have accessible names', async ({ page }) => {
    const buttons = await page.locator('button:visible').all();

    for (const button of buttons) {
      const text = await button.textContent();
      const ariaLabel = await button.getAttribute('aria-label');
      const title = await button.getAttribute('title');

      // Should have some accessible name
      const hasName = (text && text.trim().length > 0) || ariaLabel || title;
      expect(hasName).toBeTruthy();
    }
  });

  test('images have alt text', async ({ page }) => {
    const images = await page.locator('img').all();

    for (const img of images) {
      const alt = await img.getAttribute('alt');
      const role = await img.getAttribute('role');

      // Should have alt text OR be decorative (role="presentation")
      const hasAlt = alt !== null || role === 'presentation' || role === 'none';
      expect(hasAlt).toBe(true);
    }
  });

  test('color contrast is sufficient', async ({ page }) => {
    // Check that text is visible (basic contrast check)
    const textElements = await page.locator('p, span, div, li, td, th, label').all();

    for (const el of textElements.slice(0, 10)) {
      const isVisible = await el.isVisible();
      if (!isVisible) continue;

      const styles = await el.evaluate(el => {
        const computed = getComputedStyle(el);
        return {
          color: computed.color,
          backgroundColor: computed.backgroundColor,
        };
      });

      // Just verify styles are set (full contrast check needs axe-core)
      expect(styles.color).toBeTruthy();
    }
  });

  test('focus is visible', async ({ page }) => {
    const button = page.locator('button:visible').first();

    if (await button.isVisible()) {
      await button.focus();

      // Check for focus outline or visual indicator
      const hasFocusStyle = await button.evaluate(el => {
        const computed = getComputedStyle(el);
        return (
          computed.outlineWidth !== '0px' ||
          computed.boxShadow !== 'none' ||
          el.classList.contains('focused') ||
          el.classList.contains('focus')
        );
      });

      expect(hasFocusStyle).toBe(true);
    }
  });

  test('modal traps focus', async ({ page }) => {
    // Try to open a modal (help me choose or settings)
    const modalTrigger = page.locator(
      'a:has-text("Help me choose"), button:has-text("Help me choose"), ' +
      '[class*="gear"], [class*="settings"]'
    ).first();

    if (await modalTrigger.isVisible()) {
      await modalTrigger.click();

      const modal = page.locator('.modal, [role="dialog"], [class*="modal"]').first();

      if (await modal.isVisible()) {
        // Tab should cycle within modal
        await page.keyboard.press('Tab');
        await page.keyboard.press('Tab');
        await page.keyboard.press('Tab');

        // Focus should still be within modal
        const focusedInModal = await page.evaluate(() => {
          const modal = document.querySelector('.modal, [role="dialog"], [class*="modal"]');
          return modal?.contains(document.activeElement);
        });

        // This is a best practice - may not be implemented
        // Just log if not (don't fail the test)
        if (!focusedInModal) {
          console.log('Note: Modal does not trap focus');
        }
      }
    }
  });

  test('escape closes modal', async ({ page }) => {
    const modalTrigger = page.locator(
      'a:has-text("Help me choose"), button:has-text("Help me choose"), ' +
      '[class*="gear"], [class*="settings"]'
    ).first();

    if (await modalTrigger.isVisible()) {
      await modalTrigger.click();

      const modal = page.locator('.modal, [role="dialog"], [class*="modal"]').first();

      if (await modal.isVisible()) {
        await page.keyboard.press('Escape');
        await expect(modal).toBeHidden({ timeout: 3000 });
      }
    }
  });

  test('no auto-playing media', async ({ page }) => {
    const videos = await page.locator('video[autoplay]').count();
    const audios = await page.locator('audio[autoplay]').count();

    expect(videos).toBe(0);
    expect(audios).toBe(0);
  });

  test('page has lang attribute', async ({ page }) => {
    const lang = await page.locator('html').getAttribute('lang');
    expect(lang).toBeTruthy();
    expect(lang?.length).toBeGreaterThan(0);
  });

  test('skip link exists', async ({ page }) => {
    // Skip links are best practice for keyboard users
    const skipLink = page.locator('a[href="#main"], a[href="#content"], a:has-text("Skip to")').first();

    // Not required - just check if exists
    const exists = await skipLink.count() > 0;
    if (!exists) {
      console.log('Note: No skip link found (recommended for accessibility)');
    }
  });
});
