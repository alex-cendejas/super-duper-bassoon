import { apiClient } from './client';
import type {
  Workflow,
  CreateWorkflowRequest,
  EditWorkflowRequest,
  TriggerWorkflowRequest,
} from '@/types';

export const workflowsAPI = {
  getAll(): Promise<Workflow[]> {
    return apiClient.get<{ items: Workflow[] }>('/workflows').then((r) => r.items);
  },

  get(id: string): Promise<Workflow> {
    return apiClient.get(`/workflows/${id}`);
  },

  create(req: CreateWorkflowRequest): Promise<Workflow> {
    return apiClient.post('/workflows', req);
  },

  update(id: string, req: EditWorkflowRequest): Promise<Workflow> {
    return apiClient.put(`/workflows/${id}`, req);
  },

  delete(id: string): Promise<void> {
    return apiClient.delete(`/workflows/${id}`);
  },

  trigger(id: string, req?: TriggerWorkflowRequest): Promise<{ run_id: string }> {
    return apiClient.post(`/workflows/${id}/trigger`, req);
  },

  activate(id: string): Promise<Workflow> {
    return apiClient.post(`/workflows/${id}/activate`);
  },

  deactivate(id: string, reason: string): Promise<Workflow> {
    return apiClient.post(`/workflows/${id}/deactivate`, { reason });
  },
};
