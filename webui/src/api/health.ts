import { apiClient } from './client';
import type { TypeHealth, CircuitBreakerState } from '@/types';

export const healthAPI = {
  getAll(): Promise<TypeHealth[]> {
    return apiClient.get<{ items: TypeHealth[] }>('/health').then((r) => r.items);
  },

  get(workflowType: string): Promise<TypeHealth> {
    return apiClient.get(`/health/${workflowType}`);
  },

  getCircuits(): Promise<CircuitBreakerState[]> {
    return apiClient.get('/circuits');
  },

  getCircuit(workflowId: string): Promise<CircuitBreakerState> {
    return apiClient.get(`/circuits/${workflowId}`);
  },
};
