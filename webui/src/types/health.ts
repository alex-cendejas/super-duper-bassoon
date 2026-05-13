export interface TypeHealth {
  workflow_type: string;
  window_size: number;
  runs_considered: number;
  success_percentage_avg: number;
  fail_percentage_avg: number;
  error_percentage_avg: number;
  trend: HealthTrend;
  calculated_at: string;
}

export type HealthTrend = 'improving' | 'degrading' | 'stable';

export interface CircuitBreakerState {
  workflow_id: string;
  workflow_type: string;
  state: 'closed' | 'open' | 'half_open';
  opened_at?: string;
  last_evaluated_at: string;
  opened_reason?: string;
  evaluation_count: number;
}
