import { useEffect, useMemo, useRef } from 'react'
import { Background, BackgroundVariant, ReactFlow, ReactFlowProvider } from '@xyflow/react'
import '@xyflow/react/dist/base.css'
import type { SceneResponse } from '../../api'
import { Markers, edgeTypes, nodeTypes } from './parts'
import type { Highlight } from './scene'
import { findParts, toFlow } from './scene'

const none = new Set<string>()

/**
 * An archdoc view's diagram (platform-surface §17.4): a scene the engine laid out, read only, with
 * a highlight over it — the same boxes in the same places as in Architecture, with what the view
 * shows lit, tagged or hatched. Find a part lights what matches instead; where boxes can be picked,
 * Enter picks the first match.
 */
export function ViewDiagram(props: {
  scene: SceneResponse
  highlight?: Highlight
  /** Outlined: the view's subject, where it has one. */
  subject?: string
  find: string
  onFind: (q: string) => void
  /** Where a box can be picked — One part's subject — what picking does. */
  onPick?: (id: string) => void
}) {
  return (
    <ReactFlowProvider>
      <Diagram {...props} />
    </ReactFlowProvider>
  )
}

function Diagram(props: Parameters<typeof ViewDiagram>[0]) {
  const findRef = useRef<HTMLInputElement>(null)
  const found = useMemo(() => findParts(props.scene.model.nodes ?? [], props.find), [props.scene.model.nodes, props.find])
  const { nodes, edges } = useMemo(
    () =>
      toFlow(props.scene, props.scene.layout, {
        selected: props.subject ?? null,
        focus: false,
        find: props.find,
        placed: none,
        isNew: none,
        // While finding, what matches is what shows.
        highlight: found ? undefined : props.highlight,
      }),
    [props.scene, props.subject, props.find, props.highlight, found],
  )

  // "/" moves to Find a part, unless something is being typed.
  useEffect(() => {
    const on = (e: KeyboardEvent) => {
      if (e.key !== '/' || e.metaKey || e.ctrlKey || e.altKey) return
      if ((e.target as HTMLElement).closest('input, select, textarea, [contenteditable]')) return
      e.preventDefault()
      findRef.current?.focus()
    }
    window.addEventListener('keydown', on)
    return () => window.removeEventListener('keydown', on)
  }, [])

  const first = found ? (props.scene.model.nodes ?? []).find((n) => found.lit.has(n.id) && n.kind !== 'actor') : undefined

  return (
    <div className={`explorer-canvas view-diagram ${props.onPick ? 'pickable' : ''}`}>
      <Markers />
      <div className="canvas-tools">
        <label className="find">
          <span>/</span>
          <input
            ref={findRef}
            value={props.find}
            aria-label="Find a part"
            placeholder="Find a part"
            onChange={(e) => props.onFind(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && first && props.onPick) props.onPick(first.id)
              if (e.key === 'Escape') {
                props.onFind('')
                e.currentTarget.blur()
                e.stopPropagation()
              }
            }}
          />
          {found && (
            <span className="find-count" aria-live="polite">
              {found.lit.size ? `${found.lit.size} found` : 'none found'}
            </span>
          )}
        </label>
      </div>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        onNodeClick={props.onPick ? (_, n) => n.type === 'element' && props.onPick!(n.id) : undefined}
        nodesDraggable={false}
        nodesConnectable={false}
        elementsSelectable={false}
        zoomOnScroll={false}
        preventScrolling={false}
        zoomOnDoubleClick={false}
        fitView
        fitViewOptions={{ padding: 0.06 }}
        minZoom={0.1}
        maxZoom={1.6}
        proOptions={{ hideAttribution: true }}
      >
        <Background variant={BackgroundVariant.Dots} gap={18} size={1.2} color="var(--canvas-dot)" />
      </ReactFlow>
    </div>
  )
}
