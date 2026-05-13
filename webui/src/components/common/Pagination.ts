export interface PaginationOptions {
  page: number;
  limit: number;
  total: number;
  onPageChange: (page: number) => void;
  onLimitChange: (limit: number) => void;
}

export function Pagination(options: PaginationOptions): HTMLElement {
  const container = document.createElement('div');
  container.className = 'pagination';
  const totalPages = Math.ceil(options.total / options.limit);

  const prevDisabled = options.page === 1;
  const nextDisabled = options.page >= totalPages;

  container.innerHTML = `
    <div class="pagination__controls">
      <button class="p-button--small ${prevDisabled ? 'is-disabled' : ''}" ${prevDisabled ? 'disabled' : ''}>
        ← Previous
      </button>
      <span class="pagination__info">Page ${options.page} of ${totalPages}</span>
      <button class="p-button--small ${nextDisabled ? 'is-disabled' : ''}" ${nextDisabled ? 'disabled' : ''}>
        Next →
      </button>
    </div>
    <div class="pagination__limit">
      <label>Per page:
        <select>
          <option value="10" ${options.limit === 10 ? 'selected' : ''}>10</option>
          <option value="25" ${options.limit === 25 ? 'selected' : ''}>25</option>
          <option value="50" ${options.limit === 50 ? 'selected' : ''}>50</option>
          <option value="100" ${options.limit === 100 ? 'selected' : ''}>100</option>
        </select>
      </label>
    </div>
  `;

  const prevBtn = container.querySelector('button:first-of-type') as HTMLButtonElement;
  const nextBtn = container.querySelector('button:last-of-type') as HTMLButtonElement;
  const select = container.querySelector('select') as HTMLSelectElement;

  prevBtn.addEventListener('click', () => options.onPageChange(options.page - 1));
  nextBtn.addEventListener('click', () => options.onPageChange(options.page + 1));
  select.addEventListener('change', (e) => options.onLimitChange(parseInt((e.target as HTMLSelectElement).value)));

  return container;
}
