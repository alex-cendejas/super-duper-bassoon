export interface ConfirmOptions {
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  onConfirm: () => void;
  onCancel?: () => void;
}

export function ConfirmDialog(options: ConfirmOptions): HTMLElement {
  const container = document.createElement('div');
  container.className = 'p-modal';
  container.innerHTML = `
    <div class="p-modal__dialog" role="dialog">
      <header class="p-modal__header">
        <h2 class="p-modal__title">${options.title}</h2>
        <button class="p-modal__close-button" aria-label="Close"></button>
      </header>
      <div class="p-modal__body">
        <p>${options.message}</p>
      </div>
      <footer class="p-modal__footer">
        <button class="p-button--base p-button--neutral cancel-btn">${options.cancelText || 'Cancel'}</button>
        <button class="p-button--positive confirm-btn">${options.confirmText || 'Confirm'}</button>
      </footer>
    </div>
  `;

  const confirmBtn = container.querySelector('.confirm-btn') as HTMLButtonElement;
  const cancelBtn = container.querySelector('.cancel-btn') as HTMLButtonElement;
  const closeBtn = container.querySelector('.p-modal__close-button') as HTMLButtonElement;

  const handleClose = () => {
    container.remove();
    options.onCancel?.();
  };

  confirmBtn.addEventListener('click', () => {
    container.remove();
    options.onConfirm();
  });

  cancelBtn.addEventListener('click', handleClose);
  closeBtn.addEventListener('click', handleClose);

  document.body.appendChild(container);
  return container;
}
