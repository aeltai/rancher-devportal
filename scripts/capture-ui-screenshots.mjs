#!/usr/bin/env node
/**
 * Capture real Geeko-Ops UI screenshots for docs.
 * Usage: RANCHER_PASSWORD=... node scripts/capture-ui-screenshots.mjs
 */
import { chromium } from 'playwright';
import { mkdirSync } from 'fs';
import { join } from 'path';

const outDir = process.argv[2] || join(process.cwd(), 'docs/pages/screenshots');
mkdirSync(outDir, { recursive: true });

const base = process.env.DEV_UI_URL || 'https://localhost:8005';
const rancher = process.env.RANCHER_URL || 'https://localhost:8449';
const adminPass = process.env.RANCHER_PASSWORD || 'WOLTIkVzmU-q4Kxy92nu';
const testPass = process.env.TEST_PASSWORD || 'testtest123testtest123';
const portal = `${base}/platform/c/_/portal`;

async function loginLocal(page, username, password) {
  await page.goto(`${rancher}/dashboard/auth/login`, { waitUntil: 'domcontentloaded', timeout: 60000 });
  await page.waitForTimeout(1500);
  const user = page.locator('input[type="text"], input[name="username"]').first();
  const pass = page.locator('input[type="password"]').first();
  await user.fill(username);
  await pass.fill(password);
  await page.locator('button[type="submit"]').first().click();
  await page.waitForTimeout(3000);
  if (page.url().includes('/auth/setup')) {
    await page.locator('button:has-text("Continue")').click();
    await page.waitForTimeout(3000);
  }
}

async function shot(page, name, waitMs = 2000) {
  await page.waitForTimeout(waitMs);
  const path = join(outDir, `${name}.png`);
  await page.screenshot({ path, fullPage: false });
  console.log('wrote', path);
}

async function openPortal(page) {
  await page.goto(portal, { waitUntil: 'domcontentloaded', timeout: 60000 });
  await page.waitForTimeout(4000);
}

const browser = await chromium.launch({ headless: true });
const ctxOpts = { viewport: { width: 1440, height: 900 }, ignoreHTTPSErrors: true };

try {
  // --- Test user: marketplace + environments ---
  const userCtx = await browser.newContext(ctxOpts);
  const userPage = await userCtx.newPage();
  await loginLocal(userPage, 'test', testPass);
  await openPortal(userPage);
  await shot(userPage, '01-user-marketplace', 1500);

  const reqBtn = userPage.locator('button:has-text("Request environment")').first();
  if (await reqBtn.count()) {
    await reqBtn.click();
    await shot(userPage, '02-request-wizard', 2500);
    // Step through wizard if Next visible
    for (let i = 0; i < 2; i++) {
      const next = userPage.locator('button:has-text("Next")').first();
      if (!(await next.count()) || !(await next.isEnabled())) break;
      await next.click();
      await userPage.waitForTimeout(1200);
    }
    await shot(userPage, '03-wizard-configure', 1500);
    const cancel = userPage.locator('button:has-text("Cancel")').first();
    if (await cancel.count()) await cancel.click();
    await userPage.waitForTimeout(1000);
  }
  await openPortal(userPage);
  await shot(userPage, '04-my-environments', 1500);
  await userCtx.close();

  // --- Admin: requests, request env, settings ---
  const adminCtx = await browser.newContext(ctxOpts);
  const adminPage = await adminCtx.newPage();
  await loginLocal(adminPage, 'admin', adminPass);
  await openPortal(adminPage);
  await shot(adminPage, '05-admin-requests', 2000);

  const reqEnvTab = adminPage.locator('button:has-text("Request env")').first();
  if (await reqEnvTab.count()) {
    await reqEnvTab.click();
    await shot(adminPage, '06-admin-request-env', 2000);
  }

  const settingsTab = adminPage.locator('button:has-text("Platform settings")').first();
  if (await settingsTab.count()) {
    await settingsTab.click();
    await shot(adminPage, '07-platform-settings', 3000);
  }

  const requestsTab = adminPage.locator('button:has-text("Platform requests")').first();
  if (await requestsTab.count()) {
    await requestsTab.click();
    await adminPage.waitForTimeout(1000);
    const row = adminPage.locator('.dp-request-row, .dp-user-env-card-main').first();
    if (await row.count()) {
      await row.click();
      await shot(adminPage, '08-request-detail', 2000);
    }
  }
  await adminCtx.close();
} catch (e) {
  console.error('capture error:', e);
  process.exitCode = 1;
}

await browser.close();
console.log('done:', outDir);
