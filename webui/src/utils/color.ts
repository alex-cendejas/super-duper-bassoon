import type { ResultStatus, AlertSeverity, HealthTrend } from '@/types';

export function getStatusColor(status: ResultStatus): string {
  const colors: Record<ResultStatus, string> = {
    success: '#2ba700',
    fail: '#f58700',
    error: '#c7162b',
    pending: '#bdbdbd',
  };
  return colors[status];
}

export function getSeverityColor(severity: AlertSeverity): string {
  const colors: Record<AlertSeverity, string> = {
    info: '#0068d6',
    warning: '#f58700',
    critical: '#c7162b',
  };
  return colors[severity];
}

export function getHealthColor(percentage: number): string {
  if (percentage >= 80) return '#2ba700';
  if (percentage >= 60) return '#f58700';
  return '#c7162b';
}

export function getTrendColor(trend: HealthTrend): string {
  const colors: Record<HealthTrend, string> = {
    improving: '#2ba700',
    stable: '#0068d6',
    degrading: '#c7162b',
  };
  return colors[trend];
}

export function getStatusBadgeClass(status: ResultStatus): string {
  const classes: Record<ResultStatus, string> = {
    success: 'is-success',
    fail: 'is-warning',
    error: 'is-error',
    pending: 'is-neutral',
  };
  return classes[status];
}
