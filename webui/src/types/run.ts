export interface Run {
  run_id: string;
  workflow_id: string;
  workflow_type: string;
  triggered_at: string;
  dispatched_at?: string;
  participating_clients: string[];
  state: RunState;
  reason?: string;
}

export type RunState = 'pending' | 'in_progress' | 'completed' | 'failed';

export interface RunHealth {
  run_id: string;
  workflow_id: string;
  workflow_type: string;
  total_clients: number;
  success_count: number;
  fail_count: number;
  error_count: number;
  pending_count: number;
  banned_client_count: number;
  calculated_at: string;
}

export interface RunResult {
  run_id: string;
  client_id: string;
  workflow_id: string;
  status: ResultStatus;
  inner_state: Record<string, unknown>;
  error_msg?: string;
  payload?: Record<string, unknown>;
  received_at: string;
}

export type ResultStatus = 'success' | 'fail' | 'error' | 'pending';
