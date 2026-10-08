<script>
  let { name, size = 18 } = $props();
  const paths = {
    search: 'm21 21-4.5-4.5 M19 10.5a8.5 8.5 0 1 1-17 0 8.5 8.5 0 0 1 17 0',
    arrow: 'M5 12h14 m-6-6 6 6-6 6',
    back: 'M19 12H5 m6-6-6 6 6 6',
    chevron: 'm9 5 7 7-7 7',
    close: 'm6 6 12 12 M6 18 18 6',
    copy: 'M9 9h11v11H9z M15 5V2H2v13h3',
    trash: 'M3 6h18 M9 6V3h6v3 M5 6l1 15h12l1-15 M10 10v7 M14 10v7',
    code: 'm8 5-6 7 6 7 m8-14 6 7-6 7 M14 3l-4 18',
    check: 'm5 12 4 4L19 6',
    inbox: 'M4 4h16l2 13v3H2v-3L4 4 M2 15h6l2 3h4l2-3h6',
    alert: 'M12 3 2 21h20L12 3 M12 9v5 M12 17v1',
    moon: 'M21 13A9 9 0 0 1 11 3a9 9 0 1 0 10 10',
    sun: 'M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8 M12 2v2 M12 20v2 M2 12h2 M20 12h2 M5 5l1.5 1.5 M17.5 17.5 19 19 M5 19l1.5-1.5 M17.5 6.5 19 5',
    incoming: 'M18 6 6 18 M6 7v11h11',
    outgoing: 'M6 18 18 6 M7 6h11v11',
    external: 'M14 3h7v7 M21 3 10 14 M10 5H3v16h16v-7',
    book: 'M4 3h14a2 2 0 0 1 2 2v16H6a2 2 0 0 1-2-2V3 M4 17h16 M8 7h8 M8 11h5',
    message: 'M4 4h16v12H9l-5 4V4 M8 8h8 M8 12h5',
    response: 'M4 5h16v14H4z M8 9l3 3-3 3 M13 15h3',
  };
</script>

<svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
  <path d={paths[name] ?? paths.arrow} />
</svg>
