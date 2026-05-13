import type { RunHealth } from '@/types';

export function HealthBar(health: RunHealth): HTMLElement {
  const container = document.createElement('div');
  container.className = 'health-bar';
  const total = health.success_count + health.fail_count + health.error_count + health.pending_count;

  const successPct = (health.success_count / total) * 100;
  const failPct = (health.fail_count / total) * 100;
  const errorPct = (health.error_count / total) * 100;
  const pendingPct = (health.pending_count / total) * 100;

  container.innerHTML = `
    <div class="health-bar__container">
      ${successPct > 0 ? `<div class="health-bar__segment success" style="width: ${successPct}%"></div>` : ''}
      ${failPct > 0 ? `<div class="health-bar__segment warning" style="width: ${failPct}%"></div>` : ''}
      ${errorPct > 0 ? `<div class="health-bar__segment error" style="width: ${errorPct}%"></div>` : ''}
      ${pendingPct > 0 ? `<div class="health-bar__segment pending" style="width: ${pendingPct}%"></div>` : ''}
    </div>
    <div class="health-bar__legend">
      <span>${health.success_count} success</span>
      <span>${health.fail_count} fail</span>
      <span>${health.error_count} error</span>
      <span>${health.pending_count} pending</span>
    </div>
  `;
  return container;
}
