import { store } from '@/store';
import { alertsAPI } from '@/api/alerts';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { ErrorAlert } from '@/components/common/ErrorAlert';
import { Pagination } from '@/components/common/Pagination';
import { formatDate } from '@/utils/format';

export function AlertsPage(): HTMLElement {
  const page = document.createElement('div');
  page.className = 'alerts-page';
  page.innerHTML = `
    <div class="p-section">
      <h1 class="p-heading--2">Alerts</h1>
    </div>
    <div id="alerts-content"></div>
  `;

  const contentDiv = page.querySelector('#alerts-content')!;

  const render = (state = store.state) => {
    contentDiv.innerHTML = '';

    if (state.alerts.loading) {
      contentDiv.appendChild(LoadingSpinner('Loading alerts...'));
      return;
    }

    if (state.alerts.error) {
      contentDiv.appendChild(ErrorAlert(state.alerts.error));
      return;
    }

    if (state.alerts.items.length === 0) {
      contentDiv.innerHTML = '<p>No alerts found</p>';
      return;
    }

    const table = document.createElement('table');
    table.className = 'p-table';
    table.innerHTML = `
      <thead>
        <tr>
          <th>Timestamp</th>
          <th>Severity</th>
          <th>Type</th>
          <th>Message</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        ${state.alerts.items
          .map(
            (alert) => `
          <tr>
            <td>${formatDate(alert.timestamp)}</td>
            <td>${alert.severity}</td>
            <td>${alert.kind}</td>
            <td>${alert.message.substring(0, 50)}...</td>
            <td>
              <button class="p-button p-button--base p-button--small detail-btn" data-id="${alert.id}">View</button>
            </td>
          </tr>
        `
          )
          .join('')}
      </tbody>
    `;

    contentDiv.appendChild(table);

    const paginationDiv = document.createElement('div');
    paginationDiv.appendChild(
      Pagination({
        page: state.alerts.page,
        limit: state.alerts.limit,
        total: state.alerts.total,
        onPageChange: (page) => {
          store.setAlertsLoading(true);
          alertsAPI
            .getAll({ limit: state.alerts.limit, offset: (page - 1) * state.alerts.limit })
            .then((response) => store.setAlerts(response))
            .catch((err) => store.setState('alerts', { error: err.message, loading: false }));
        },
        onLimitChange: (limit) => {
          store.setAlertsLoading(true);
          alertsAPI
            .getAll({ limit, offset: 0 })
            .then((response) => store.setAlerts(response))
            .catch((err) => store.setState('alerts', { error: err.message, loading: false }));
        },
      })
    );
    contentDiv.appendChild(paginationDiv);

    table.querySelectorAll('.detail-btn').forEach((btn) => {
      btn.addEventListener('click', (e) => {
        const id = (e.target as HTMLButtonElement).dataset.id!;
        store.openModal(`alert-detail-${id}`);
      });
    });
  };

  store.subscribe(render);

  store.setAlertsLoading(true);
  alertsAPI
    .getAll({ limit: store.state.alerts.limit })
    .then((response) => store.setAlerts(response))
    .catch((err) => store.setState('alerts', { error: err.message, loading: false }));

  return page;
}
