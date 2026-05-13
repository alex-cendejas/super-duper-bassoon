import { apiClient } from './client';
import type { BanRecord, UnbanRequest } from '@/types';

export interface BanFilters {
  client_id?: string;
  workflow_type?: string;
  reason?: string;
}

export const bansAPI = {
  getAll(_filters?: BanFilters): Promise<BanRecord[]> {
    return apiClient.get<{ items: BanRecord[] }>('/bans').then((r) => r.items ?? []);
  },

  get(clientId: string): Promise<BanRecord[]> {
    return apiClient.get(`/bans/${clientId}`);
  },

  unban(clientId: string, req: UnbanRequest): Promise<void> {
    return apiClient.put(`/bans/${clientId}/unban`, req);
  },
};
