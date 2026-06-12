/**
 * Desktop Helpers for Puppeteer QA Testing
 * Viewport: 1440×900 (standard desktop)
 */
const puppeteer = require('puppeteer-core');

const DESKTOP_VIEWPORT = { width: 1440, height: 900 };
const HOST = process.env.QA_HOST || 'http://localhost:3000';
const CHROME_PATH = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';

async function createDesktopBrowser() {
  const browser = await puppeteer.launch({
    executablePath: CHROME_PATH,
    headless: 'new',
    defaultViewport: DESKTOP_VIEWPORT,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1440,900'],
  });
  return browser;
}

async function loginAs(page, username, password = 'Admin123') {
  await page.goto(`${HOST}/login`, { waitUntil: 'networkidle0', timeout: 15000 });
  await page.waitForSelector('input[type="text"], input[type="email"], input[name="username"]', { timeout: 5000 });

  // Find and fill username field
  const usernameInput = await page.$('input[type="text"]') ||
                         await page.$('input[type="email"]') ||
                         await page.$('input[name="username"]');
  if (usernameInput) {
    await usernameInput.click({ clickCount: 3 });
    await usernameInput.type(username);
  }

  // Find and fill password field
  const passwordInput = await page.$('input[type="password"]');
  if (passwordInput) {
    await passwordInput.click({ clickCount: 3 });
    await passwordInput.type(password);
  }

  // Submit form
  const submitBtn = await page.$('button[type="submit"]') || await page.$('button:not([type])');
  if (submitBtn) {
    await submitBtn.click();
  }

  // Wait for navigation after login
  await page.waitForNavigation({ waitUntil: 'networkidle0', timeout: 10000 }).catch(() => {});
  await new Promise(r => setTimeout(r, 1000));

  return !page.url().includes('/login');
}

async function takeScreenshot(page, name) {
  const path = `qa/screenshots/${name}.png`;
  await page.screenshot({ path, fullPage: false });
  return path;
}

async function apiCall(method, path, body = null, token = null) {
  const fetch = (await import('node-fetch')).default;
  const baseUrl = process.env.API_HOST || 'http://localhost:8080';
  const opts = {
    method,
    headers: { 'Content-Type': 'application/json' },
  };
  if (token) opts.headers['Authorization'] = `Bearer ${token}`;
  if (body) opts.body = JSON.stringify(body);

  const res = await fetch(`${baseUrl}${path}`, opts);
  const text = await res.text();
  let data;
  try { data = JSON.parse(text); } catch { data = text; }
  return { status: res.status, data, ok: res.ok };
}

async function getAdminToken() {
  const { status, data } = await apiCall('POST', '/api/v1/auth/login', {
    username: 'frankng',
    password: 'Admin123',
  });
  if (status !== 200) throw new Error(`Login failed: ${status}`);
  return data.data?.access_token || data.access_token || data.data?.token || data.token;
}

async function getPartnerToken() {
  const { status, data } = await apiCall('POST', '/api/v1/auth/login', {
    username: 'thanhmai',
    password: 'Admin123',
  });
  if (status !== 200) throw new Error(`Partner login failed: ${status} ${JSON.stringify(data)}`);
  return data.data?.access_token || data.access_token || data.data?.token || data.token;
}

module.exports = {
  DESKTOP_VIEWPORT,
  HOST,
  CHROME_PATH,
  createDesktopBrowser,
  loginAs,
  takeScreenshot,
  apiCall,
  getAdminToken,
  getPartnerToken,
};
