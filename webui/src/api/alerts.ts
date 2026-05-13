import type { Alert } from '@/types';

export interface AlertFilters {
  severity?: string;
  type?: string;
  workflow?: string;
  limit?: number;
  offset?: number;
}

export const alertsAPI = {
  getAll(_filters?: AlertFilters): Promise<{ items: Alert[]; total: number }> {
    return Promise.resolve({ items: [], total: 0 });
  },

  get(_id: string): Promise<Alert> {
    return Promise.reject(new Error('Not implemented'));
  },
};
