export interface Workflow {
  id: string;
  name: string;
  description: string;
  workflow_type: string;
  activity: ActivityType;
  params: Record<string, unknown>;
  target_filter: string;
  trigger: TriggerSpec;
  success_threshold: number;
  loop_threshold_ms: number;
  timeout_ms: number;
  active: boolean;
  created_at: string;
  updated_at: string;
  deactivated_reason?: string;
}

export type ActivityType =
  | 'reboot'
  | 'install_package'
  | 'upgrade_package'
  | 'remove_package'
  | 'apply_config'
  | 'validate_config'
  | 'run_script';

export type TriggerKind = 'scheduled' | 'event' | 'state_change' | 'manual';

export interface TriggerSpec {
  kind: TriggerKind;
  cron?: string;
  on_complete?: string;
  on_state_change?: string;
}

export interface CreateWorkflowRequest {
  name: string;
  description: string;
  workflow_type: string;
  activity: ActivityType;
  params: Record<string, unknown>;
  target_filter: string;
  trigger: TriggerSpec;
  success_threshold: number;
  loop_threshold_ms: number;
  timeout_ms: number;
  active: boolean;
}

export interface EditWorkflowRequest extends CreateWorkflowRequest {}

export interface TriggerWorkflowRequest {
  reason?: string;
}
