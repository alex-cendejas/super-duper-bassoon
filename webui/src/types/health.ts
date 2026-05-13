export interface TypeHealth {
  workflow_type: string;
  total_runs: number;
  success_percentage: number;
  fail_percentage: number;
  error_percentage: number;
  trend: HealthTrend;
  circuit_broken: boolean;
  last_run_at?: string;
}

export type HealthTrend = 'improving' | 'degrading' | 'stable';

export interface CircuitBreakerState {
  workflow_id: string;
  workflow_type: string;
  active: boolean;
  deactivated_reason?: string;
  deactivated_at?: string;
}
