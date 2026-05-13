export interface Run {
  id: string;
  workflow_id: string;
  workflow_type: string;
  triggered_at: string;
  state: RunState;
  health: RunHealth;
  completed_at?: string;
}

export type RunState = 'pending' | 'in_progress' | 'completed' | 'failed';

export interface RunHealth {
  total_clients: number;
  success_count: number;
  fail_count: number;
  error_count: number;
  pending_count: number;
  banned_count: number;
  success_percentage: number;
}

export interface RunResult {
  run_id: string;
  client_id: string;
  workflow_id: string;
  status: ResultStatus;
  inner_state: Record<string, unknown>;
  error_msg?: string;
  payload?: Record<string, unknown>;
  timestamp: string;
  os?: string;
  labels?: Record<string, string>;
}

export type ResultStatus = 'success' | 'fail' | 'error' | 'pending';
