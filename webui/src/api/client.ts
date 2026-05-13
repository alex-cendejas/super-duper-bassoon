import { config } from '@/config';

export class APIError extends Error {
  constructor(
    public code: string,
    public status: number,
    message: string
  ) {
    super(message);
    this.name = 'APIError';
  }
}

export class APIClient {
  private baseUrl: string;
  private timeout: number;

  constructor(baseUrl: string = config.apiBaseUrl, timeout: number = config.apiTimeout) {
    this.baseUrl = baseUrl;
    this.timeout = timeout;
  }

  async request<T = unknown>(
    method: 'GET' | 'POST' | 'PUT' | 'DELETE',
    path: string,
    body?: unknown
  ): Promise<T> {
    const url = `${this.baseUrl}/api${path}`;
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);

    try {
      const response = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: body ? JSON.stringify(body) : undefined,
        signal: controller.signal,
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new APIError(
          errorData.code || 'UNKNOWN_ERROR',
          response.status,
          errorData.message || response.statusText
        );
      }

      const data: T = await response.json();
      return data;
    } finally {
      clearTimeout(timeoutId);
    }
  }

  get<T = unknown>(path: string): Promise<T> {
    return this.request('GET', path);
  }

  post<T = unknown>(path: string, body?: unknown): Promise<T> {
    return this.request('POST', path, body);
  }

  put<T = unknown>(path: string, body?: unknown): Promise<T> {
    return this.request('PUT', path, body);
  }

  delete<T = unknown>(path: string): Promise<T> {
    return this.request('DELETE', path);
  }
}

export const apiClient = new APIClient();
