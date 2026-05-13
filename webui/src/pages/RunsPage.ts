import { store } from '@/store';
import { runsAPI } from '@/api/runs';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { ErrorAlert } from '@/components/common/ErrorAlert';
import { Pagination } from '@/components/common/Pagination';
import { formatDate, formatPercentage } from '@/utils/format';

export function RunsPage(): HTMLElement {
  const page = document.createElement('div');
  page.className = 'runs-page';
  page.innerHTML = `
    <div class="p-section">
      <h1 class="p-heading--2">Runs</h1>
    </div>
    <div id="runs-content"></div>
  `;

  const contentDiv = page.querySelector('#runs-content')!;

  const render = (state = store.state) => {
    contentDiv.innerHTML = '';

    if (state.runs.loading) {
      contentDiv.appendChild(LoadingSpinner('Loading runs...'));
      return;
    }

    if (state.runs.error) {
      contentDiv.appendChild(ErrorAlert(state.runs.error));
      return;
    }

    const table = document.createElement('table');
    table.className = 'p-table';
    table.innerHTML = `
      <thead>
        <tr>
          <th>Run ID</th>
          <th>Workflow</th>
          <th>Triggered</th>
          <th>State</th>
          <th>Success %</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        ${state.runs.items
          .map(
            (run) => `
          <tr>
            <td class="run-id">${run.id}</td>
            <td>${run.workflow_type}</td>
            <td>${formatDate(run.triggered_at)}</td>
            <td><span class="p-badge">${run.state}</span></td>
            <td>${formatPercentage(run.health.success_percentage)}</td>
            <td>
              <button class="p-button p-button--base p-button--small details-btn" data-id="${run.id}">Details</button>
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
        page: state.runs.page,
        limit: state.runs.limit,
        total: state.runs.total,
        onPageChange: (page) => {
          store.setRunsLoading(true);
          runsAPI
            .getAll({ limit: state.runs.limit, offset: (page - 1) * state.runs.limit })
            .then((response) => store.setRuns(response))
            .catch((err) => store.setState('runs', { error: err.message, loading: false }));
        },
        onLimitChange: (limit) => {
          store.setRunsLoading(true);
          runsAPI
            .getAll({ limit, offset: 0 })
            .then((response) => store.setRuns(response))
            .catch((err) => store.setState('runs', { error: err.message, loading: false }));
        },
      })
    );
    contentDiv.appendChild(paginationDiv);

    table.querySelectorAll('.details-btn').forEach((btn) => {
      btn.addEventListener('click', (e) => {
        const id = (e.target as HTMLButtonElement).dataset.id!;
        store.openModal(`run-detail-${id}`);
      });
    });
  };

  store.subscribe(render);

  store.setRunsLoading(true);
  runsAPI
    .getAll({ limit: store.state.runs.limit })
    .then((response) => store.setRuns(response))
    .catch((err) => store.setState('runs', { error: err.message, loading: false }));

  return page;
}
