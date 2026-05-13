import { store } from '@/store';
import { bansAPI } from '@/api/bans';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { ErrorAlert } from '@/components/common/ErrorAlert';
import { formatDate } from '@/utils/format';

export function BansPage(): HTMLElement {
  const page = document.createElement('div');
  page.className = 'bans-page';
  page.innerHTML = `
    <div class="p-section">
      <h1 class="p-heading--2">Ban Management</h1>
    </div>
    <div id="bans-content"></div>
  `;

  const contentDiv = page.querySelector('#bans-content')!;

  const render = (state = store.state) => {
    contentDiv.innerHTML = '';

    if (state.bans.loading) {
      contentDiv.appendChild(LoadingSpinner('Loading bans...'));
      return;
    }

    if (state.bans.error) {
      contentDiv.appendChild(ErrorAlert(state.bans.error));
      return;
    }

    if (state.bans.items.length === 0) {
      contentDiv.innerHTML = '<p>No active bans</p>';
      return;
    }

    const table = document.createElement('table');
    table.className = 'p-table';
    table.innerHTML = `
      <thead>
        <tr>
          <th>Client ID</th>
          <th>Workflow Type</th>
          <th>Reason</th>
          <th>Banned At</th>
          <th>Status</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        ${state.bans.items
          .map(
            (ban) => `
          <tr>
            <td>${ban.client_id}</td>
            <td>${ban.workflow_type}</td>
            <td>${ban.reason}</td>
            <td>${formatDate(ban.banned_at)}</td>
            <td>${ban.active ? '<span class="p-badge--negative">Active</span>' : '<span class="p-badge--neutral">Expired</span>'}</td>
            <td>
              <button class="p-button--link detail-btn" data-id="${ban.client_id}">Details</button>
              ${ban.active ? '<button class="p-button--link unban-btn" data-id="' + ban.client_id + '">Unban</button>' : ''}
            </td>
          </tr>
        `
          )
          .join('')}
      </tbody>
    `;

    contentDiv.appendChild(table);

    table.querySelectorAll('.detail-btn').forEach((btn) => {
      btn.addEventListener('click', (e) => {
        const id = (e.target as HTMLButtonElement).dataset.id!;
        store.openModal(`ban-detail-${id}`);
      });
    });

    table.querySelectorAll('.unban-btn').forEach((btn) => {
      btn.addEventListener('click', (e) => {
        const id = (e.target as HTMLButtonElement).dataset.id!;
        store.openModal(`unban-${id}`);
      });
    });
  };

  store.subscribe(render);

  store.setBansLoading(true);
  bansAPI
    .getAll()
    .then((bans) => store.setBans(bans))
    .catch((err) => store.setState('bans', { error: err.message, loading: false }));

  return page;
}
