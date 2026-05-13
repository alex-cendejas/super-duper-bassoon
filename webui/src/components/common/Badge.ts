import type { ResultStatus, AlertSeverity } from '@/types';
import { getStatusBadgeClass } from '@/utils/color';

export function Badge(
  text: string,
  type: 'status' | 'severity' | 'default' | 'success' | 'warning' | 'error',
  options?: { class?: string }
): HTMLElement {
  const span = document.createElement('span');
  let badgeClass = 'p-badge';

  if (type === 'status') badgeClass += ' is-status';
  else if (type === 'severity') badgeClass += ' is-severity';
  else if (type === 'success') badgeClass += ' p-badge--positive';
  else if (type === 'warning') badgeClass += ' p-badge--caution';
  else if (type === 'error') badgeClass += ' p-badge--negative';

  span.className = `${badgeClass} ${options?.class || ''}`;
  span.textContent = text;
  return span;
}

export function StatusBadge(status: ResultStatus | string): HTMLElement {
  const span = document.createElement('span');
  span.className = `p-badge ${getStatusBadgeClass(status as ResultStatus)}`;
  span.textContent = status;
  return span;
}

export function SeverityBadge(severity: AlertSeverity | string): HTMLElement {
  const span = document.createElement('span');
  let badgeClass = 'p-badge';
  if (severity === 'critical' || severity === 'error') badgeClass += ' p-badge--negative';
  else if (severity === 'warning') badgeClass += ' p-badge--caution';
  else badgeClass += ' p-badge--positive';

  span.className = badgeClass;
  span.textContent = severity;
  return span;
}
