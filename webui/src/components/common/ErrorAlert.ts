export function ErrorAlert(message: string, onDismiss?: () => void): HTMLElement {
  const container = document.createElement('div');
  container.className = 'p-notification--negative';
  container.innerHTML = `
    <div class="p-notification__content">
      <h5 class="p-notification__title">Error</h5>
      <p class="p-notification__message">${message}</p>
      ${onDismiss ? '<button class="p-button--link p-notification__action">Dismiss</button>' : ''}
    </div>
  `;

  if (onDismiss) {
    const button = container.querySelector('button');
    if (button) {
      button.addEventListener('click', () => {
        container.remove();
        onDismiss();
      });
    }
  }

  return container;
}
