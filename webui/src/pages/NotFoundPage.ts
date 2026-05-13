import { router } from '@/router';

export function NotFoundPage(): HTMLElement {
  const page = document.createElement('div');
  page.className = 'not-found-page';
  page.innerHTML = `
    <div class="p-section">
      <h1 class="p-heading--1">404 - Page Not Found</h1>
      <p>The page you're looking for doesn't exist.</p>
      <button class="p-button--positive" id="go-home">Go to Workflows</button>
    </div>
  `;

  const btn = page.querySelector('#go-home') as HTMLButtonElement;
  btn.addEventListener('click', () => {
    router.navigate('#/workflows');
  });

  return page;
}
