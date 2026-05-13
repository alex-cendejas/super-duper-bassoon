import { router } from '@/router';

export function Sidebar(): HTMLElement {
  const sidebar = document.createElement('aside');
  sidebar.className = 'p-side-navigation';
  sidebar.innerHTML = `
    <nav class="p-side-navigation__nav">
      <ul class="p-side-navigation__list">
        <li class="p-side-navigation__item">
          <a class="p-side-navigation__link" href="#/workflows">
            <span class="p-side-navigation__label">Workflows</span>
          </a>
        </li>
        <li class="p-side-navigation__item">
          <a class="p-side-navigation__link" href="#/runs">
            <span class="p-side-navigation__label">Runs</span>
          </a>
        </li>
        <li class="p-side-navigation__item">
          <a class="p-side-navigation__link" href="#/health">
            <span class="p-side-navigation__label">Health</span>
          </a>
        </li>
        <li class="p-side-navigation__item">
          <a class="p-side-navigation__link" href="#/clients">
            <span class="p-side-navigation__label">Clients</span>
          </a>
        </li>
        <li class="p-side-navigation__item">
          <a class="p-side-navigation__link" href="#/alerts">
            <span class="p-side-navigation__label">Alerts</span>
          </a>
        </li>
        <li class="p-side-navigation__item">
          <a class="p-side-navigation__link" href="#/bans">
            <span class="p-side-navigation__label">Bans</span>
          </a>
        </li>
      </ul>
    </nav>
  `;

  router.subscribe((path) => {
    const links = sidebar.querySelectorAll('.p-side-navigation__link');
    links.forEach((link) => {
      const href = (link as HTMLAnchorElement).getAttribute('href');
      const isActive = path.startsWith(href || '');
      if (isActive) {
        link.classList.add('is-active');
      } else {
        link.classList.remove('is-active');
      }
    });
  });

  return sidebar;
}
