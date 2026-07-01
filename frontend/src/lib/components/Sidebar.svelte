<script lang="ts">
  import Drawer, {
    AppContent,
    Content,
    Header,
    Title,
    Subtitle,
    Scrim,
  } from '@smui/drawer';
  import List, { Item, Text, Graphic, Separator } from '@smui/list';
  import IconButton from '@smui/icon-button';

  type View = 'home' | 'catalog' | 'color-search' | 'compare' | 'mix';

  interface Props {
    currentView: View;
    collapsed: boolean;
    onNavigate: (view: View) => void;
    onToggle: () => void;
  }

  let { currentView, collapsed, onNavigate, onToggle }: Props = $props();

  const navItems: { id: View; label: string; icon: string }[] = [
    { id: 'home', label: 'Início', icon: 'home' },
    { id: 'catalog', label: 'Catálogo', icon: 'inventory_2' },
    { id: 'color-search', label: 'Buscar Cor', icon: 'colorize' },
    { id: 'compare', label: 'Comparar', icon: 'compare_arrows' },
    { id: 'mix', label: 'Mistura', icon: 'science' },
  ];
</script>

<Drawer variant="dismissible" bind:open={() => !collapsed, (v) => { if (v === collapsed) onToggle(); }}>
  <Header>
    <div class="sidebar-logo">
      <div class="logo-icon">
        <span class="material-icons" style="color: white; font-size: 20px;">palette</span>
      </div>
      {#if !collapsed}
        <div>
          <Title class="font-display" style="font-size: 15px; font-weight: 700; color: white;">Paint Match</Title>
          <Subtitle style="font-size: 9px; font-weight: 600; letter-spacing: 0.2em; text-transform: uppercase; color: var(--color-amber-glow);">AI</Subtitle>
        </div>
      {/if}
    </div>
  </Header>

  <Content>
    <List>
      {#each navItems as item}
        {@const isActive = currentView === item.id}
        <Item
          href="javascript:void(0)"
          onclick={() => onNavigate(item.id)}
          activated={isActive}
        >
          <Graphic class="material-icons" style="color: {isActive ? 'var(--color-amber-glow)' : 'var(--color-obsidian-400)'};">{item.icon}</Graphic>
          {#if !collapsed}
            <Text>{item.label}</Text>
            {#if isActive}
              <div class="active-dot"></div>
            {/if}
          {/if}
        </Item>
      {/each}
    </List>
  </Content>

  {#if !collapsed}
    <div class="sidebar-version">v1.0.0</div>
  {/if}

  <div class="sidebar-toggle">
    <IconButton onclick={onToggle} style="color: var(--color-obsidian-500);">
      <span class="material-icons" style="transition: transform 0.3s; transform: rotate({collapsed ? '180deg' : '0deg'});">chevron_left</span>
    </IconButton>
  </div>
</Drawer>

<AppContent>
  <!-- slotted content in App.svelte -->
</AppContent>

<style>
  :global(.smui-drawer) {
    background: var(--color-obsidian-900) !important;
    border-right: 1px solid rgba(255, 255, 255, 0.06);
    transition: width 0.3s ease;
  }

  :global(.smui-drawer--dismissible) {
    width: 220px;
  }

  :global(.smui-drawer--dismissible.collapsed) {
    width: 68px;
  }

  :global(.mdc-drawer-item) {
    border-radius: 12px !important;
    margin: 2px 8px;
  }

  :global(.mdc-drawer-item--activated) {
    background: linear-gradient(135deg, rgba(212, 160, 83, 0.1), rgba(212, 160, 83, 0.03)) !important;
    border: 1px solid rgba(212, 160, 83, 0.12);
  }

  :global(.mdc-drawer-item--activated .mdc-list-item__text) {
    color: white !important;
  }

  .sidebar-logo {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 16px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  }

  .logo-icon {
    width: 36px;
    height: 36px;
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    background: linear-gradient(135deg, var(--color-amber-glow), var(--color-amber-warm));
    box-shadow: 0 4px 12px rgba(212, 160, 83, 0.3);
  }

  .active-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--color-amber-glow);
    margin-left: auto;
    box-shadow: 0 0 6px rgba(212, 160, 83, 0.4);
  }

  .sidebar-version {
    padding: 16px;
    font-size: 10px;
    color: var(--color-obsidian-600);
    font-family: var(--font-mono);
  }

  .sidebar-toggle {
    padding: 10px;
    border-top: 1px solid rgba(255, 255, 255, 0.05);
    display: flex;
    justify-content: center;
  }
</style>
