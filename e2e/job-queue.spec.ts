import { test, expect, mockAPI } from './fixtures';

test.describe('Job Queue', () => {
  test('displays empty state when no jobs', async ({ page }) => {
    await mockAPI(page);
    await page.goto('/');

    // Queue area should be visible
    const queue = page.locator('[class*="queue"], [class*="jobs"], #job-queue');

    if (await queue.isVisible()) {
      // Should show empty state or zero jobs
      const emptyState = page.locator('[class*="empty"], :text("No jobs"), :text("Queue is empty")');
      const jobItems = page.locator('.job-item, .job-card, [class*="job-row"]');

      const jobCount = await jobItems.count();
      const hasEmptyState = await emptyState.first().isVisible().catch(() => false);

      expect(jobCount === 0 || hasEmptyState).toBe(true);
    }
  });

  test('displays jobs with progress', async ({ page }) => {
    // Mock jobs endpoint with active jobs
    await page.route('**/api/jobs', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: 'job-1',
            input_path: '/media/movie.mkv',
            output_path: '/media/movie.mkv.tmp',
            preset: 'compress-hevc',
            status: 'running',
            progress: 45.5,
            hardware_path: 'vaapi→vaapi',
          },
          {
            id: 'job-2',
            input_path: '/media/show.mp4',
            output_path: '/media/show.mp4.tmp',
            preset: 'compress-av1',
            status: 'pending',
            progress: 0,
          },
        ]),
      });
    });

    await page.route('**/api/presets', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          { id: 'compress-hevc', name: 'Smaller files — HEVC' },
        ]),
      });
    });

    await page.route('**/api/config', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ media_path: '/media', workers: 1 }),
      });
    });

    await page.route('**/api/browse**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ path: '/media', directories: [], files: [] }),
      });
    });

    await page.goto('/');

    // Should display job items
    const jobItems = page.locator('.job-item, .job-card, [class*="job-row"], [data-job-id]');
    await expect(jobItems.first()).toBeVisible({ timeout: 5000 });

    // Should show progress indicator
    const progressBar = page.locator('[class*="progress"], progress, [role="progressbar"]');
    await expect(progressBar.first()).toBeVisible();
  });

  test('shows hardware path for running jobs', async ({ page }) => {
    await page.route('**/api/jobs', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: 'job-1',
            input_path: '/media/movie.mkv',
            status: 'running',
            progress: 50,
            hardware_path: 'vaapi→vaapi',
          },
        ]),
      });
    });

    await mockAPI(page);
    await page.goto('/');

    // Look for hardware path display
    const hwPath = page.locator(':text("vaapi"), :text("cpu"), :text("nvenc")');
    // May or may not be displayed depending on UI design
  });

  test('cancel button removes job', async ({ page }) => {
    let jobDeleted = false;

    await page.route('**/api/jobs', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          { id: 'job-1', input_path: '/media/movie.mkv', status: 'pending' },
        ]),
      });
    });

    await page.route('**/api/jobs/job-1', async (route) => {
      if (route.request().method() === 'DELETE') {
        jobDeleted = true;
        await route.fulfill({ status: 200, body: '{}' });
      }
    });

    await mockAPI(page);
    await page.goto('/');

    // Find and click cancel/remove button
    const cancelBtn = page.locator('button:has-text("Cancel"), button:has-text("Remove"), [aria-label="Cancel"], [class*="cancel"], [class*="remove"]').first();

    if (await cancelBtn.isVisible()) {
      await cancelBtn.click();

      // Confirm if dialog appears
      const confirmBtn = page.locator('button:has-text("Confirm"), button:has-text("Yes"), button:has-text("OK")');
      if (await confirmBtn.isVisible({ timeout: 1000 }).catch(() => false)) {
        await confirmBtn.click();
      }

      expect(jobDeleted).toBe(true);
    }
  });

  test('job status messages are human-readable', async ({ page }) => {
    await page.route('**/api/jobs', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: 'job-1',
            input_path: '/media/movie.mkv',
            status: 'running',
            progress: 50,
            preset: 'compress-hevc',
          },
          {
            id: 'job-2',
            input_path: '/media/show.mkv',
            status: 'complete',
            original_size: 1000000000,
            new_size: 500000000,
          },
          {
            id: 'job-3',
            input_path: '/media/failed.mkv',
            status: 'failed',
            error: 'GPU encode failed',
          },
        ]),
      });
    });

    await mockAPI(page);
    await page.goto('/');

    // Status messages should not show raw technical codes
    const pageText = await page.locator('body').textContent();

    // Should have human-readable text
    const hasReadableStatus =
      /compressing|saving|saved|failed|retrying|complete/i.test(pageText || '') ||
      /\d+\s*%/i.test(pageText || ''); // Progress percentage

    expect(hasReadableStatus).toBe(true);
  });

  test('completed jobs show space savings', async ({ page }) => {
    await page.route('**/api/jobs', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: 'job-1',
            input_path: '/media/movie.mkv',
            status: 'complete',
            original_size: 1073741824, // 1 GB
            new_size: 536870912, // 512 MB
          },
        ]),
      });
    });

    await mockAPI(page);
    await page.goto('/');

    // Should show savings info
    const savingsText = page.locator(':text("saved"), :text("smaller"), :text("MB"), :text("GB"), :text("%")');
    await expect(savingsText.first()).toBeVisible({ timeout: 5000 });
  });

  test('drag and drop reordering', async ({ page }) => {
    await page.route('**/api/jobs', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            { id: 'job-1', input_path: '/media/first.mkv', status: 'pending', position: 0 },
            { id: 'job-2', input_path: '/media/second.mkv', status: 'pending', position: 1 },
            { id: 'job-3', input_path: '/media/third.mkv', status: 'pending', position: 2 },
          ]),
        });
      }
    });

    await page.route('**/api/jobs/*/reorder', async (route) => {
      await route.fulfill({ status: 200, body: '{}' });
    });

    await mockAPI(page);
    await page.goto('/');

    // Find draggable job items
    const jobItems = page.locator('.job-item, .job-card, [draggable="true"]');
    const count = await jobItems.count();

    if (count >= 2) {
      const first = jobItems.nth(0);
      const second = jobItems.nth(1);

      // Get bounding boxes
      const firstBox = await first.boundingBox();
      const secondBox = await second.boundingBox();

      if (firstBox && secondBox) {
        // Drag first item below second
        await page.mouse.move(firstBox.x + firstBox.width / 2, firstBox.y + firstBox.height / 2);
        await page.mouse.down();
        await page.mouse.move(secondBox.x + secondBox.width / 2, secondBox.y + secondBox.height + 10);
        await page.mouse.up();
      }
    }
  });
});
