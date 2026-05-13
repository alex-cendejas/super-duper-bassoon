import { apiClient } from './client';
import type { Run, RunResult, PaginatedResponse } from '@/types';

export interface RunFilters {
  workflow_type?: string;
  state?: string;
  limit?: number;
  offset?: number;
}

export const runsAPI = {
  getAll(filters?: RunFilters): Promise<PaginatedResponse<Run>> {
    const params = new URLSearchParams();
    if (filters) {
      if (filters.workflow_type) params.append('workflow_type', filters.workflow_type);
      if (filters.state) params.append('state', filters.state);
      if (filters.limit) params.append('limit', filters.limit.toString());
      if (filters.offset) params.append('offset', filters.offset.toString());
    }
    const queryStr = params.toString();
    return apiClient.get(`/runs${queryStr ? `?${queryStr}` : ''}`);
  },

  get(id: string): Promise<Run> {
    return apiClient.get(`/runs/${id}`);
  },

  getResults(id: string): Promise<RunResult[]> {
    return apiClient.get(`/runs/${id}/results`);
  },

  getByWorkflow(workflowId: string): Promise<Run[]> {
    return apiClient.get(`/workflows/${workflowId}/runs`);
  },
};
