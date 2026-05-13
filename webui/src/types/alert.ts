export interface Alert {
  id: string;
  kind: string;
  severity: AlertSeverity;
  message: string;
  details: Record<string, unknown>;
  timestamp: string;
}

export type AlertSeverity = 'info' | 'warning' | 'critical';
