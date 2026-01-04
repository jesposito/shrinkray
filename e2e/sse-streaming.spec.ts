import { test, expect, mockAPI } from './fixtures';

test.describe('SSE Real-time Updates', () => {
  test('page loads with mocked SSE', async ({ page }) => {
    await mockAPI(page);
    await page.goto('/');

    // Page should load successfully even with mocked SSE
    await expect(page.locator('body')).toBeVisible();
  });

  test('SSE endpoint is requested', async ({ page }) => {
    let sseRequested = false;

    await page.route('**/api/jobs/stream', async (route) => {
      sseRequested = true;
      await route.fulfill({
        status: 200,
        contentType: 'text/event-stream',
        body: 'data: []\n\n',
      });
    });

    await mockAPI(page);
    await page.goto('/');
    await page.waitForTimeout(1000);

    expect(sseRequested).toBe(true);
  });
});
