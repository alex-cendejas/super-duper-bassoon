import { apiClient } from './client';
import type { ClientMetadata, UpdateClientRequest } from '@/types';

export interface ClientFilters {
  os?: string;
  active?: boolean;
  search?: string;
}

export const clientsAPI = {
  getAll(_filters?: ClientFilters): Promise<ClientMetadata[]> {
    return apiClient.get<{ items: ClientMetadata[] }>('/clients').then((r) => r.items);
  },

  get(id: string): Promise<ClientMetadata> {
    return apiClient.get(`/clients/${id}`);
  },

  updateLabels(id: string, labels: Record<string, string>): Promise<ClientMetadata> {
    return apiClient.put(`/clients/${id}`, { labels });
  },

  update(id: string, req: UpdateClientRequest): Promise<ClientMetadata> {
    return apiClient.put(`/clients/${id}`, req);
  },

  delete(id: string): Promise<void> {
    return apiClient.delete(`/clients/${id}`);
  },
};
