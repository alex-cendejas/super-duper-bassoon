import { apiClient } from './client';
import type { Alert } from '@/types';

export interface AlertFilters {
  severity?: string;
  type?: string;
  workflow?: string;
  limit?: number;
  offset?: number;
}

export const alertsAPI = {
  getAll(filters?: AlertFilters): Promise<{ items: Alert[]; total: number }> {
    const params = new URLSearchParams();
    if (filters) {
      if (filters.severity) params.append('severity', filters.severity);
      if (filters.type) params.append('type', filters.type);
      if (filters.workflow) params.append('workflow', filters.workflow);
      if (filters.limit) params.append('limit', filters.limit.toString());
      if (filters.offset) params.append('offset', filters.offset.toString());
    }
    const queryStr = params.toString();
    return apiClient.get(`/alerts${queryStr ? `?${queryStr}` : ''}`);
  },

  get(id: string): Promise<Alert> {
    return apiClient.get(`/alerts/${id}`);
  },
};
