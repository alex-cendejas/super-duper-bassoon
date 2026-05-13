import { apiClient } from './client';
import type { Run, RunResult } from '@/types';

export interface RunFilters {
  workflow_type?: string;
  state?: string;
  limit?: number;
  offset?: number;
}

export const runsAPI = {
  getAll(filters?: RunFilters): Promise<{ items: Run[]; total: number }> {
    const params = new URLSearchParams();
    if (filters) {
      if (filters.workflow_type) params.append('workflow_type', filters.workflow_type);
      if (filters.state) params.append('state', filters.state);
      if (filters.limit) params.append('limit', filters.limit.toString());
      if (filters.offset) params.append('offset', filters.offset.toString());
    }
    const queryStr = params.toString();
    return apiClient
      .get<{ items: Run[]; total: number }>(`/runs${queryStr ? `?${queryStr}` : ''}`)
      .then((r) => ({ items: r.items ?? [], total: r.total ?? 0 }));
  },

  get(id: string): Promise<Run> {
    return apiClient.get(`/runs/${id}`);
  },

  getResults(id: string): Promise<RunResult[]> {
    return apiClient.get<{ items: RunResult[] }>(`/runs/${id}/results`).then((r) => r.items ?? []);
  },

  getByWorkflow(workflowId: string): Promise<Run[]> {
    return apiClient.get(`/workflows/${workflowId}/runs`);
  },
};
