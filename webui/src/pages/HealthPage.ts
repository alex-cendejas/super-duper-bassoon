import { store } from '@/store';
import { healthAPI } from '@/api/health';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { ErrorAlert } from '@/components/common/ErrorAlert';
import { formatPercentage } from '@/utils/format';

export function HealthPage(): HTMLElement {
  const page = document.createElement('div');
  page.className = 'health-page';
  page.innerHTML = `
    <div class="p-section">
      <h1 class="p-heading--2">Health Dashboard</h1>
    </div>
    <div id="health-content"></div>
  `;

  const contentDiv = page.querySelector('#health-content')!;

  const render = (state = store.state) => {
    contentDiv.innerHTML = '';

    if (state.health.loading) {
      contentDiv.appendChild(LoadingSpinner('Loading health metrics...'));
      return;
    }

    if (state.health.error) {
      contentDiv.appendChild(ErrorAlert(state.health.error));
      return;
    }

    if (state.health.items.length === 0) {
      contentDiv.innerHTML = '<p>No health data available</p>';
      return;
    }

    const grid = document.createElement('div');
    grid.className = 'p-strips';

    state.health.items.forEach((health) => {
      const card = document.createElement('div');
      card.className = 'p-strip p-card';
      const circuitStatus = health.circuit_broken ? 'BROKEN' : 'ACTIVE';
      const statusClass = health.circuit_broken ? 'p-badge--negative' : 'p-badge--positive';

      card.innerHTML = `
        <h3>${health.workflow_type}</h3>
        <div class="health-metrics">
          <p>Success: ${formatPercentage(health.success_percentage)}</p>
          <p>Failures: ${formatPercentage(health.fail_percentage)}</p>
          <p>Errors: ${formatPercentage(health.error_percentage)}</p>
          <p>Trend: <span class="trend-${health.trend}">${health.trend}</span></p>
          <p>Circuit: <span class="p-badge ${statusClass}">${circuitStatus}</span></p>
        </div>
      `;
      grid.appendChild(card);
    });

    contentDiv.appendChild(grid);
  };

  store.subscribe(render);

  store.setHealthLoading(true);
  healthAPI
    .getAll()
    .then((health) => store.setHealth(health))
    .catch((err) => store.setState('health', { error: err.message, loading: false }));

  return page;
}
