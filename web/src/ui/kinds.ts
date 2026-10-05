// What each element kind looks like (surface-spec §6.3, "Surface Foundations" 6.3). Kind is hue:
// fill tint, stroke, and a 3px cap in the stroke colour. Technology is text beside the icon,
// never a vendor logo — the icon says what a thing is, not who made it.

export interface KindStyle {
  label: string
  hue: 'slate' | 'teal' | 'green' | 'violet' | 'grey'
  icon: string
}

// 16-unit grid, 1.4 stroke.
export const icons = {
  person: 'M8 7a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5z M3 14c0-3 2.2-5 5-5s5 2 5 5',
  system: 'M2 3h12v10H2z M5 6h6 M5 9h4',
  external: 'M8 2a6 6 0 1 0 0 12A6 6 0 0 0 8 2z M2 8h12 M8 2c2 2 2 10 0 12 M8 2c-2 2-2 10 0 12',
  container: 'M2 3h12v10H2z M2 6h12',
  proxy: 'M2 3h12v10H2z M2 6h12 M5 9.5h6 M9 8l2 1.5-2 1.5',
  component: 'M5 3h9v10H5z M3 5h4v2H3z M3 9h4v2H3z',
  datastore: 'M3 4c0-1.1 2.2-2 5-2s5 .9 5 2v8c0 1.1-2.2 2-5 2s-5-.9-5-2z M3 4c0 1.1 2.2 2 5 2s5-.9 5-2',
  queue: 'M2 5h12v6H2z M5 5v6 M8 5v6 M11 5v6',
  entry: 'M1.5 8h8 M6.5 5l3 3-3 3 M10.5 2.5h3.5v11h-3.5',
  entity: 'M2 3h12v10H2z M2 6.5h12 M6 6.5V13',
  flow: 'M2 3h5v3H2z M9 10h5v3H9z M4.5 6v5.5H9',
  dependency: 'M2 5l6-3 6 3v6l-6 3-6-3z M2 5l6 3 6-3 M8 8v6',
}

const kinds: Record<string, KindStyle> = {
  actor: { label: 'Person', hue: 'green', icon: icons.person },
  system: { label: 'Software system', hue: 'slate', icon: icons.system },
  external: { label: 'External system', hue: 'violet', icon: icons.external },
  application: { label: 'Container', hue: 'slate', icon: icons.container },
  component: { label: 'Component', hue: 'slate', icon: icons.component },
  proxy: { label: 'Proxy', hue: 'slate', icon: icons.proxy },
  datastore: { label: 'Data store', hue: 'teal', icon: icons.datastore },
  queue: { label: 'Queue', hue: 'teal', icon: icons.queue },
}

export function kindStyle(kind: string): KindStyle {
  return kinds[kind] ?? { label: kind, hue: 'grey', icon: icons.container }
}
