export interface Alert {
  id: string;
  timestamp: string;
  severity: AlertSeverity;
  type: string;
  message: string;
  source_workflow?: string;
  source_client?: string;
  run_id_evidence?: string;
  details: Record<string, unknown>;
}

export type AlertSeverity = 'info' | 'warning' | 'error' | 'critical';
