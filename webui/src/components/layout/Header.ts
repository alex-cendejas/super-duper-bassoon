import { store } from '@/store';

export function Header(): HTMLElement {
  const header = document.createElement('header');
  header.className = 'p-navigation';
  header.innerHTML = `
    <div class="p-navigation__banner">
      <div class="p-navigation__logo">
        <a href="/">
          <img src="/logo.png" alt="Logo" class="logo-img" />
          <span class="site-name">super-duper-bassoon</span>
        </a>
      </div>
      <nav class="p-navigation__nav">
        <ul class="p-navigation__items">
          <li class="p-navigation__item">
            <a href="#/workflows">Workflows</a>
          </li>
          <li class="p-navigation__item">
            <a href="#/runs">Runs</a>
          </li>
          <li class="p-navigation__item">
            <a href="#/health">Health</a>
          </li>
          <li class="p-navigation__item">
            <a href="#/clients">Clients</a>
          </li>
          <li class="p-navigation__item">
            <a href="#/alerts">Alerts</a>
          </li>
          <li class="p-navigation__item">
            <a href="#/bans">Bans</a>
          </li>
        </ul>
      </nav>
      <div class="p-navigation__aside">
        <button id="toggle-sidebar" class="p-button--base p-button--neutral">Menu</button>
      </div>
    </div>
  `;

  const toggleBtn = header.querySelector('#toggle-sidebar') as HTMLButtonElement;
  toggleBtn?.addEventListener('click', () => {
    store.toggleSidebar();
  });

  return header;
}
