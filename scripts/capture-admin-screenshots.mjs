#!/usr/bin/env node
import { chromium } from 'playwright';
import { join } from 'path';

const outDir = join(process.cwd(), 'docs/pages/screenshots');
const portal = 'https://localhost:8005/platform/c/_/portal';
const rancher = 'https://localhost:8449';
const adminPass = process.env.RANCHER_PASSWORD || 'WOLTIkVzmU-q4Kxy92nu';

async function loginAdmin(page) {
  await page.goto(`${rancher}/dashboard/auth/login`, { waitUntil: 'domcontentloaded', timeout: 60000 });
  await page.waitForTimeout(1500);
  await page.locator('input[type="text"]').first().fill('admin');
  await page.locator('input[type="password"]').first().fill(adminPass);
  await page.locator('button[type="submit"]').first().click();
  await page.waitForTimeout(3000);
  if (page.url().includes('/auth/setup')) {
    await page.locator('button:has-text("Continue")').click();
    await page.waitForTimeout(3000);
  }
}

const browser = await chromium.launch({ headless: true });
const page = await browser.newPage({ viewport: { width: 1440, height: 900 }, ignoreHTTPSErrors: true });

await loginAdmin(page);
await page.goto(portal, { waitUntil: 'domcontentloaded', timeout: 60000 });
await page.waitForTimeout(6000);

if (!(await page.locator('.dp-admin-view').count())) {
  throw new Error('Admin view not loaded');
}

async function tab(label, file) {
  await page.locator('.dp-admin-main-tab', { hasText: label }).click();
  await page.waitForTimeout(3000);
  await page.screenshot({ path: join(outDir, file) });
  console.log('wrote', file);
}

await tab('Ops queue', '05-admin-ops-queue.png');
await tab('Request env', '06-admin-request-env.png');
await tab('Catalog & config', '07-catalog-config.png');

await page.locator('.dp-admin-main-tab', { hasText: 'Ops queue' }).click();
await page.waitForTimeout(1500);
const row = page.locator('.dp-request-row').first();
if (await row.count()) {
  await row.click();
  await page.waitForTimeout(1500);
  await page.screenshot({ path: join(outDir, '08-request-detail.png') });
  console.log('wrote 08-request-detail.png');
}

await browser.close();
