import { store } from '@/store';
import { clientsAPI } from '@/api/clients';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { ErrorAlert } from '@/components/common/ErrorAlert';
import { formatDate } from '@/utils/format';

export function ClientsPage(): HTMLElement {
  const page = document.createElement('div');
  page.className = 'clients-page';
  page.innerHTML = `
    <div class="p-section">
      <h1 class="p-heading--2">Clients</h1>
    </div>
    <div id="clients-content"></div>
  `;

  const contentDiv = page.querySelector('#clients-content')!;

  const render = (state = store.state) => {
    contentDiv.innerHTML = '';

    if (state.clients.loading) {
      contentDiv.appendChild(LoadingSpinner('Loading clients...'));
      return;
    }

    if (state.clients.error) {
      contentDiv.appendChild(ErrorAlert(state.clients.error));
      return;
    }

    if (state.clients.items.length === 0) {
      contentDiv.innerHTML = '<p>No clients found</p>';
      return;
    }

    const table = document.createElement('table');
    table.className = 'p-table';
    table.innerHTML = `
      <thead>
        <tr>
          <th>Client ID</th>
          <th>OS</th>
          <th>Status</th>
          <th>Last Seen</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        ${state.clients.items
          .map(
            (client) => `
          <tr>
            <td>${client.client_id}</td>
            <td>${client.os}</td>
            <td>${client.active ? '<span class="p-badge--positive">Active</span>' : '<span class="p-badge--neutral">Inactive</span>'}</td>
            <td>${formatDate(client.last_seen_at)}</td>
            <td>
              <button class="p-button p-button--base p-button--small detail-btn" data-id="${client.client_id}">Details</button>
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
        store.openModal(`client-detail-${id}`);
      });
    });
  };

  store.subscribe(render);

  store.setClientsLoading(true);
  clientsAPI
    .getAll()
    .then((clients) => store.setClients(clients))
    .catch((err) => store.setState('clients', { error: err.message, loading: false }));

  return page;
}
