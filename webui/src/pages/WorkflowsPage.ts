import { store } from '@/store';
import { workflowsAPI } from '@/api/workflows';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { ErrorAlert } from '@/components/common/ErrorAlert';
import { formatDate } from '@/utils/format';

export function WorkflowsPage(): HTMLElement {
  const page = document.createElement('div');
  page.className = 'workflows-page';
  page.innerHTML = `
    <div class="p-section">
      <div class="p-section__header">
        <h1 class="p-heading--2">Workflows</h1>
        <button class="p-button--positive" id="create-workflow-btn">Create Workflow</button>
      </div>
    </div>
    <div id="workflows-content"></div>
  `;

  const contentDiv = page.querySelector('#workflows-content')!;

  const render = (state = store.state) => {
    contentDiv.innerHTML = '';

    if (state.workflows.loading) {
      contentDiv.appendChild(LoadingSpinner('Loading workflows...'));
      return;
    }

    if (state.workflows.error) {
      contentDiv.appendChild(ErrorAlert(state.workflows.error));
      return;
    }

    if (state.workflows.items.length === 0) {
      contentDiv.innerHTML = '<p>No workflows found</p>';
      return;
    }

    const table = document.createElement('table');
    table.className = 'p-table';
    table.innerHTML = `
      <thead>
        <tr>
          <th>Name</th>
          <th>Type</th>
          <th>Activity</th>
          <th>Status</th>
          <th>Created</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        ${state.workflows.items
          .map(
            (wf) => `
          <tr>
            <td>${wf.name}</td>
            <td>${wf.workflow_type}</td>
            <td>${wf.activity}</td>
            <td>${wf.active ? '<span class="p-badge--positive">Active</span>' : '<span class="p-badge--neutral">Inactive</span>'}</td>
            <td>${formatDate(wf.created_at)}</td>
            <td>
              <button class="p-button p-button--base p-button--small view-btn" data-id="${wf.id}">View</button>
              <button class="p-button p-button--positive p-button--small trigger-btn" data-id="${wf.id}">Trigger</button>
            </td>
          </tr>
        `
          )
          .join('')}
      </tbody>
    `;

    contentDiv.appendChild(table);

    table.querySelectorAll('.view-btn').forEach((btn) => {
      btn.addEventListener('click', (e) => {
        const id = (e.target as HTMLButtonElement).dataset.id!;
        store.openModal(`workflow-detail-${id}`);
      });
    });

    table.querySelectorAll('.trigger-btn').forEach((btn) => {
      btn.addEventListener('click', async (e) => {
        const id = (e.target as HTMLButtonElement).dataset.id!;
        try {
          await workflowsAPI.trigger(id);
          store.showToast('Workflow triggered successfully', 'success');
        } catch (err) {
          store.showToast('Failed to trigger workflow', 'error');
        }
      });
    });
  };

  store.subscribe(render);

  const createBtn = page.querySelector('#create-workflow-btn') as HTMLButtonElement;
  createBtn.addEventListener('click', () => {
    store.openModal('create-workflow');
  });

  store.setWorkflowLoading(true);
  workflowsAPI
    .getAll()
    .then((workflows) => store.setWorkflows(workflows))
    .catch((err) => store.setState('workflows', { error: err.message, loading: false }));

  return page;
}
