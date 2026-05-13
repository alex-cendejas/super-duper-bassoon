import { store } from '@/store';

export function Header(): HTMLElement {
  const header = document.createElement('header');
  header.className = 'p-navigation';
  header.innerHTML = `
    <div class="p-navigation__banner">
      <button id="toggle-sidebar" class="p-navigation__toggle" aria-label="Toggle sidebar">
        <svg class="toggle-icon toggle-icon--menu" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <line x1="3" y1="6" x2="21" y2="6"/>
          <line x1="3" y1="12" x2="21" y2="12"/>
          <line x1="3" y1="18" x2="21" y2="18"/>
        </svg>
        <svg class="toggle-icon toggle-icon--close" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <line x1="18" y1="6" x2="6" y2="18"/>
          <line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>
      <div class="p-navigation__logo">
        <a href="/">
          <img src="/logo.png" alt="Logo" class="logo-img" />
          <span class="site-name">super-duper-bassoon</span>
        </a>
      </div>
    </div>
  `;

  const toggleBtn = header.querySelector('#toggle-sidebar') as HTMLButtonElement;
  toggleBtn?.addEventListener('click', () => {
    store.toggleSidebar();
  });

  store.subscribe((state) => {
    if (state.ui.sidebarOpen) {
      toggleBtn?.classList.add('is-open');
    } else {
      toggleBtn?.classList.remove('is-open');
    }
  });

  return header;
}
