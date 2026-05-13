import { store } from '@/store';
import { clientsAPI } from '@/api/clients';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { ErrorAlert } from '@/components/common/ErrorAlert';
import { formatDate } from '@/utils/format';
import type { ClientMetadata } from '@/types';

export function ClientDetailModal(clientId: string): HTMLElement {
  const container = document.createElement('div');
  container.className = 'p-modal';
  container.innerHTML = `
    <div class="p-modal__dialog" role="dialog">
      <header class="p-modal__header">
        <h2 class="p-modal__title">Client Details</h2>
        <button class="p-modal__close-button" aria-label="Close"></button>
      </header>
      <div class="p-modal__body">
        <div id="client-detail-content"></div>
      </div>
      <footer class="p-modal__footer">
        <button class="p-button p-button--base close-btn">Close</button>
      </footer>
    </div>
  `;

  const contentDiv = container.querySelector('#client-detail-content')!;
  const closeBtn = container.querySelector('.close-btn') as HTMLButtonElement;
  const closeIconBtn = container.querySelector('.p-modal__close-button') as HTMLButtonElement;

  const handleClose = () => {
    store.closeModal(`client-detail-${clientId}`);
  };

  const renderContent = (client: ClientMetadata | null, loading: boolean, error: string | null) => {
    contentDiv.innerHTML = '';

    if (loading) {
      contentDiv.appendChild(LoadingSpinner('Loading client details...'));
      return;
    }

    if (error) {
      contentDiv.appendChild(ErrorAlert(error));
      return;
    }

    if (!client) {
      contentDiv.innerHTML = '<p>Client not found</p>';
      return;
    }

    const detailsHtml = document.createElement('div');
    detailsHtml.className = 'client-details';
    detailsHtml.innerHTML = `
      <div class="detail-section">
        <h3>Basic Information</h3>
        <table class="p-table">
          <tbody>
            <tr>
              <th>Client ID</th>
              <td>${escapeHtml(client.client_id)}</td>
            </tr>
            <tr>
              <th>Operating System</th>
              <td>${escapeHtml(client.os)}</td>
            </tr>
            <tr>
              <th>Status</th>
              <td>${client.active ? '<span class="p-badge--positive">Active</span>' : '<span class="p-badge--neutral">Inactive</span>'}</td>
            </tr>
            <tr>
              <th>Last Seen</th>
              <td>${formatDate(client.last_seen_at)}</td>
            </tr>
          </tbody>
        </table>
      </div>

      ${client.labels && Object.keys(client.labels).length > 0 ? `
        <div class="detail-section">
          <h3>Labels</h3>
          <table class="p-table">
            <tbody>
              ${Object.entries(client.labels)
                .map(([key, value]) => `
                <tr>
                  <th>${escapeHtml(key)}</th>
                  <td>${escapeHtml(value)}</td>
                </tr>
              `)
                .join('')}
            </tbody>
          </table>
        </div>
      ` : ''}

      ${client.inner_state && Object.keys(client.inner_state).length > 0 ? `
        <div class="detail-section">
          <h3>Internal State</h3>
          <pre><code>${escapeHtml(JSON.stringify(client.inner_state, null, 2))}</code></pre>
        </div>
      ` : ''}
    `;

    contentDiv.appendChild(detailsHtml);
  };

  closeBtn.addEventListener('click', handleClose);
  closeIconBtn.addEventListener('click', handleClose);

  let loading = true;
  let error: string | null = null;
  let clientData: ClientMetadata | null = null;

  renderContent(clientData, loading, error);

  clientsAPI
    .get(clientId)
    .then((client) => {
      clientData = client;
      loading = false;
      error = null;
      renderContent(clientData, loading, error);
    })
    .catch((err) => {
      loading = false;
      error = err.message || 'Failed to load client details';
      renderContent(clientData, loading, error);
    });

  return container;
}

function escapeHtml(text: string): string {
  const map: Record<string, string> = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#039;',
  };
  return text.replace(/[&<>"']/g, (char) => map[char]);
}
