<script lang="ts">
  interface Props {
    name:
      | 'home' | 'grid' | 'pipette' | 'swap' | 'flask' | 'chevron-left'
      | 'search' | 'close' | 'info' | 'search-off' | 'building' | 'layers'
      | 'palette' | 'flag' | 'wheel' | 'book' | 'chevron-down'
      | 'box' | 'plus' | 'trash' | 'edit' | 'upload' | 'download' | 'check' | 'droplet'
      | 'undo' | 'redo' | 'crosshair' | 'save' | 'image' | 'maximize' | 'menu'
      | 'sidebar' | 'pin' | 'zoom-in' | 'zoom-out';
    size?: number;
  }

  let { name, size = 18 }: Props = $props();
</script>

<svg
  width={size}
  height={size}
  viewBox="0 0 24 24"
  fill="none"
  stroke="currentColor"
  stroke-width="1.6"
  stroke-linecap="round"
  stroke-linejoin="round"
  aria-hidden="true"
>
  {#if name === 'home'}
    <path d="M4 11.5 12 4l8 7.5" />
    <path d="M6 10v9a1 1 0 0 0 1 1h3v-6h4v6h3a1 1 0 0 0 1-1v-9" />
  {:else if name === 'grid'}
    <rect x="4" y="4" width="7" height="7" rx="1.2" />
    <rect x="13" y="4" width="7" height="7" rx="1.2" />
    <rect x="4" y="13" width="7" height="7" rx="1.2" />
    <rect x="13" y="13" width="7" height="7" rx="1.2" />
  {:else if name === 'pipette'}
    <path d="M14.8 3.4 20.6 9.2 16.9 12.9 11.1 7.1z" />
    <path d="M12.4 8.4 5.1 15.7l-1.3 3.9 3.9-1.3 7.3-7.3" />
    <path d="M3.5 20.5l1.3-1.3" />
  {:else if name === 'swap'}
    <path d="M4 8h13.5" />
    <path d="M14 4.5 17.5 8 14 11.5" />
    <path d="M20 16H6.5" />
    <path d="M10 12.5 6.5 16 10 19.5" />
  {:else if name === 'flask'}
    <path d="M10 3h4" />
    <path d="M10.5 3.5v5.4L5.6 17.2a2 2 0 0 0 1.75 3h9.3a2 2 0 0 0 1.75-3l-4.9-8.3V3.5" />
    <path d="M8.2 14.8h7.6" />
  {:else if name === 'chevron-left'}
    <path d="M14.5 5 8 12l6.5 7" />
  {:else if name === 'search'}
    <circle cx="10.5" cy="10.5" r="6.2" />
    <path d="M19 19l-4.3-4.3" />
  {:else if name === 'close'}
    <path d="M5 5l14 14" />
    <path d="M19 5 5 19" />
  {:else if name === 'info'}
    <circle cx="12" cy="12" r="8.5" />
    <path d="M12 11v5.3" />
    <circle cx="12" cy="7.9" r="0.9" fill="currentColor" stroke="none" />
  {:else if name === 'search-off'}
    <circle cx="10.5" cy="10.5" r="6.2" />
    <path d="M19 19l-4.3-4.3" />
    <path d="M4 4l16 16" />
  {:else if name === 'building'}
    <rect x="5" y="3" width="10" height="18" rx="1" />
    <path d="M9 7h2M9 11h2M9 15h2" />
    <path d="M15 10.5h4V21h-4" />
  {:else if name === 'layers'}
    <path d="M12 3 3 8l9 5 9-5-9-5z" />
    <path d="M3 12l9 5 9-5" />
    <path d="M3 16l9 5 9-5" />
  {:else if name === 'palette'}
    <path d="M12 3a9 8 0 1 0 3.3 15.5c.9-.4 1-1.6.2-2.2-.9-.7-.4-2.1.8-2.1H18a3 3 0 0 0 3-3.2C20.7 6.4 16.9 3 12 3z" />
    <circle cx="8.2" cy="11" r="1.1" fill="currentColor" stroke="none" />
    <circle cx="12" cy="7.6" r="1.1" fill="currentColor" stroke="none" />
    <circle cx="15.8" cy="11" r="1.1" fill="currentColor" stroke="none" />
  {:else if name === 'flag'}
    <path d="M6 21V4" />
    <path d="M6 4.5h11l-2.8 3.5L17 11.5H6" />
  {:else if name === 'wheel'}
    <circle cx="12" cy="12" r="8.5" />
    <path d="M12 3.5v17" />
    <path d="M4.64 7.75 19.36 16.25" />
    <path d="M4.64 16.25 19.36 7.75" />
    <circle cx="12" cy="12" r="2.1" fill="currentColor" stroke="none" />
  {:else if name === 'book'}
    <path d="M4 5.5A1.5 1.5 0 0 1 5.5 4H11v15H5.5A1.5 1.5 0 0 0 4 20.5z" />
    <path d="M20 5.5A1.5 1.5 0 0 0 18.5 4H13v15h5.5a1.5 1.5 0 0 1 1.5 1.5z" />
  {:else if name === 'chevron-down'}
    <path d="M5 9.5 12 16l7-6.5" />
  {:else if name === 'box'}
    <path d="M12 3 4 7v10l8 4 8-4V7z" />
    <path d="M4 7l8 4 8-4" />
    <path d="M12 11v10" />
  {:else if name === 'plus'}
    <path d="M12 5v14" />
    <path d="M5 12h14" />
  {:else if name === 'trash'}
    <path d="M4 6.5h16" />
    <path d="M9 6.5V4.5h6v2" />
    <path d="M6.5 6.5 7.3 20a1 1 0 0 0 1 .9h7.4a1 1 0 0 0 1-.9l.8-13.5" />
    <path d="M10 10.5v6M14 10.5v6" />
  {:else if name === 'edit'}
    <path d="M4 20h4L18.5 9.5a2 2 0 0 0-2.8-2.8L5 17.2z" />
    <path d="M14.5 5.5l4 4" />
  {:else if name === 'upload'}
    <path d="M4 15v3a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-3" />
    <path d="M12 16V4" />
    <path d="M7.5 8.5 12 4l4.5 4.5" />
  {:else if name === 'download'}
    <path d="M4 15v3a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-3" />
    <path d="M12 4v12" />
    <path d="M7.5 11.5 12 16l4.5-4.5" />
  {:else if name === 'check'}
    <path d="M5 12.5 10 17.5 19 6.5" />
  {:else if name === 'droplet'}
    <path d="M12 2.5c-2.5 4-6.5 8-6.5 11.7a6.5 6.5 0 1 0 13 0C18.5 10.5 14.5 6.5 12 2.5Z" />
  {:else if name === 'undo'}
    <path d="M3 7v6h6" />
    <path d="M3 13a9 9 0 0 1 15.4-6.4L21 9" />
  {:else if name === 'redo'}
    <path d="M21 7v6h-6" />
    <path d="M21 13a9 9 0 0 0-15.4-6.4L3 9" />
  {:else if name === 'crosshair'}
    <circle cx="12" cy="12" r="7" />
    <path d="M12 2v3M12 19v3M2 12h3M19 12h3" />
  {:else if name === 'save'}
    <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z" />
    <path d="M17 21v-8H7v8" />
    <path d="M7 3v5h8" />
  {:else if name === 'image'}
    <rect x="3" y="3" width="18" height="18" rx="2" />
    <circle cx="8.5" cy="8.5" r="1.5" fill="currentColor" stroke="none" />
    <path d="m21 15-5-5L5 21" />
  {:else if name === 'maximize'}
    <path d="M8 3H5a2 2 0 0 0-2 2v3" />
    <path d="M21 8V5a2 2 0 0 0-2-2h-3" />
    <path d="M3 16v3a2 2 0 0 0 2 2h3" />
    <path d="M16 21h3a2 2 0 0 0 2-2v-3" />
  {:else if name === 'menu'}
    <path d="M4 6h16M4 12h16M4 18h16" />
  {:else if name === 'sidebar'}
    <rect x="3" y="3" width="18" height="18" rx="2" />
    <path d="M15 3v18" />
  {:else if name === 'pin'}
    <path d="M12 17v5" />
    <path d="M9 10.5 3 21h18L15 10.5" />
    <path d="M12 3a4 4 0 0 0-4 4c0 2.5 4 3.5 4 6.5 0-3 4-4 4-6.5a4 4 0 0 0-4-4z" />
  {:else if name === 'zoom-in'}
    <circle cx="10.5" cy="10.5" r="6.2" />
    <path d="M19 19l-4.3-4.3" />
    <path d="M10.5 7.5v6M7.5 10.5h6" />
  {:else if name === 'zoom-out'}
    <circle cx="10.5" cy="10.5" r="6.2" />
    <path d="M19 19l-4.3-4.3" />
    <path d="M7.5 10.5h6" />
  {/if}
</svg>
