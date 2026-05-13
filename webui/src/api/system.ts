import { apiClient } from './client';
import type { SystemStatus } from '@/types';

export const systemAPI = {
  getStatus(): Promise<SystemStatus> {
    return apiClient.get('/status');
  },
};
