import { kindStyle } from './kinds'

/** A kind's icon on its tint. Referenced elements — the repository only names them — are dashed. */
export function KindTile({ kind, referenced, size = 30 }: { kind: string; referenced?: boolean; size?: number }) {
  const k = kindStyle(kind)
  const icon = Math.round(size * 0.55)
  return (
    <span className={`kind-tile tile-${k.hue} ${referenced ? 'dashed' : ''}`} style={{ width: size, height: size }} title={k.label}>
      <svg width={icon} height={icon} viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinejoin="round" strokeLinecap="round">
        <path d={k.icon} />
      </svg>
    </span>
  )
}
