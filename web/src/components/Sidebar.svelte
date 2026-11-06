<script lang="ts">
  let {
    currentHash,
    open = false,
    onClose,
  }: {
    currentHash: string;
    open?: boolean;
    onClose?: () => void;
  } = $props();

  const navGroups = [
    {
      label: 'System',
      items: [
        { href: '#/', icon: '⬛', label: 'Dashboard' },
        { href: '#/plank', icon: '🪵', label: 'P.L.A.N.K.' },
        { href: '#/services', icon: '⚙', label: 'Services' },
        { href: '#/processes', icon: '◈', label: 'Processes' },
        { href: '#/disks', icon: '◫', label: 'Disks' },
        { href: '#/users', icon: '◉', label: 'Users' },
        { href: '#/logs', icon: '≡', label: 'Logs' },
      ],
    },
    {
      label: 'Network',
      items: [
        { href: '#/ports', icon: '◎', label: 'Ports & Sockets' },
        { href: '#/network', icon: '⟁', label: 'Interfaces' },
        { href: '#/firewall', icon: '⬡', label: 'Firewall' },
      ],
    },
    {
      label: 'Applications',
      items: [
        { href: '#/containers', icon: '▣', label: 'Containers' },
        { href: '#/databases', icon: '⬠', label: 'Databases' },
        { href: '#/webservers', icon: '◫', label: 'Webservers' },
        { href: '#/packages', icon: '◫', label: 'Packages' },
        { href: '#/files', icon: '◱', label: 'Files' },
        { href: '#/cron', icon: '◷', label: 'Cron' },
      ],
    },
    {
      label: 'Automation',
      items: [
        { href: '#/ansible', icon: '▶', label: 'Ansible' },
        { href: '#/puppet', icon: '◆', label: 'Puppet' },
      ],
    },
    {
      label: 'Tools',
      items: [
        { href: '#/migration', icon: '⇨', label: 'Migration' },
        { href: '#/terminal', icon: '▮', label: 'Terminal' },
        { href: '#/ai', icon: '◍', label: 'AI Assistant' },
      ],
    },
  ];

  function navigate() {
    onClose?.();
  }
</script>

<nav class="sidebar" class:open aria-label="Main navigation">
  <div class="logo">
    <a href="#/" class="logo-link" onclick={navigate}>
      <span class="logo-word">monk</span><span class="logo-accent">os</span>
    </a>
    <button type="button" class="sidebar-close" onclick={() => onClose?.()} aria-label="Close menu">✕</button>
  </div>

  {#each navGroups as group}
    <div class="nav-group">
      <span class="nav-group-label">{group.label}</span>
      {#each group.items as item}
        <a
          href={item.href}
          class="nav-item"
          class:active={currentHash === item.href}
          onclick={navigate}
        >
          <span class="nav-icon" aria-hidden="true">{item.icon}</span>
          <span class="nav-label">{item.label}</span>
        </a>
      {/each}
    </div>
  {/each}

  <div class="sidebar-footer">
    <a
      href="#/settings"
      class="nav-item"
      class:active={currentHash === '#/settings'}
      onclick={navigate}
    >
      <span class="nav-icon" aria-hidden="true">◈</span>
      <span class="nav-label">Settings</span>
    </a>
    <div class="version-tag">monkos</div>
  </div>
</nav>

<style>
  .sidebar {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow-y: auto;
    overflow-x: hidden;
  }

  .logo {
    padding: 0 0.75rem 0 1rem;
    height: 52px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
    border-bottom: 1px solid var(--border-subtle);
    flex-shrink: 0;
  }
  .logo-link {
    display: flex;
    align-items: center;
    text-decoration: none;
  }
  .logo-word {
    font-family: var(--font-mono);
    font-size: 1.45rem;
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: -0.03em;
  }
  .logo-accent {
    font-family: var(--font-mono);
    font-size: 1.45rem;
    font-weight: 700;
    color: var(--accent);
    letter-spacing: -0.03em;
  }
  .sidebar-close {
    display: none;
    background: none;
    border: 1px solid var(--border-default);
    color: var(--text-secondary);
    border-radius: var(--r-sm);
    width: 2rem;
    height: 2rem;
    font-size: 0.95rem;
    cursor: pointer;
    line-height: 1;
  }
  .sidebar-close:hover {
    color: var(--text-primary);
    border-color: var(--accent);
  }

  .nav-group {
    padding: 0.75rem 0 0.25rem;
    flex-shrink: 0;
  }
  .nav-group-label {
    display: block;
    font-size: 0.65rem;
    font-weight: 500;
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.1em;
    padding: 0 0.875rem 0.375rem;
  }
  .nav-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 0.875rem;
    color: var(--nav-item);
    font-size: 0.9rem;
    font-weight: 400;
    font-family: var(--font-mono);
    text-decoration: none;
    transition: color 0.12s, background 0.12s;
    position: relative;
  }
  .nav-item:hover {
    color: var(--nav-item-hover);
    background: var(--accent-dim);
  }
  .nav-item.active {
    color: var(--accent);
    background: var(--accent-dim);
    font-weight: 500;
  }
  .nav-item.active::before {
    content: '';
    position: absolute;
    left: 0;
    top: 0;
    bottom: 0;
    width: 2px;
    background: var(--accent);
    border-radius: 0 2px 2px 0;
  }
  .nav-icon {
    font-size: var(--icon-md);
    width: 1.35rem;
    text-align: center;
    flex-shrink: 0;
    opacity: 0.75;
    line-height: 1;
  }
  .nav-item:hover .nav-icon,
  .nav-item.active .nav-icon {
    opacity: 1;
  }
  .nav-label {
    flex: 1;
  }

  .sidebar-footer {
    margin-top: auto;
    padding: 0.5rem 0;
    border-top: 1px solid var(--border-subtle);
    flex-shrink: 0;
  }
  .version-tag {
    font-family: var(--font-mono);
    font-size: 0.6rem;
    color: var(--text-tertiary);
    padding: 0.375rem 0.875rem;
    letter-spacing: 0.08em;
  }

  @media (max-width: 900px) {
    .sidebar-close {
      display: inline-flex;
      align-items: center;
      justify-content: center;
    }
  }
</style>
