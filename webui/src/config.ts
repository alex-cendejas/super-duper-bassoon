const isDev = typeof window !== 'undefined' && window.location.hostname === 'localhost';

export const config = {
  apiBaseUrl: isDev ? 'http://localhost:8080' : '',
  apiTimeout: 30000,
  pollingInterval: 5000,
};
