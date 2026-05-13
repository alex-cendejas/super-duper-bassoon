export function LoadingSpinner(message?: string): HTMLElement {
  const container = document.createElement('div');
  container.className = 'loading-spinner';
  container.innerHTML = `
    <div class="spinner-content">
      <div class="spinner"></div>
      ${message ? `<p>${message}</p>` : ''}
    </div>
  `;
  return container;
}
