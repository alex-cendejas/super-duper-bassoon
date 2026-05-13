// Dev mode is detected by the Vite dev server port (5173), not by hostname.
// When served through Docker/nginx at localhost:3000, hostname is still 'localhost'
// but we must use relative paths so the nginx proxy handles /api/* correctly.
// Port 8080 is internal-only in Docker and not accessible from the browser.
const isDev = typeof window !== 'undefined' && window.location.port === '5173';

export const config = {
  apiBaseUrl: isDev ? 'http://localhost:8080' : '',
  apiTimeout: 30000,
  pollingInterval: 5000,
};
