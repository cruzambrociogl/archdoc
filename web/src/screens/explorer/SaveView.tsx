import { useState } from 'react'
import type { ViewsResponse } from '../../api'
import { action, useApi } from '../../api'
import type { Route } from '../../route'
import { Eyebrow } from '../../ui/marks'

/**
 * Name the current view and keep it in .archdoc/views.yaml (surface-spec S-7). What it captures is
 * shown before saving, as the exact entry that will be written; commit the file and the team opens
 * the view by name, in the app and in the published site.
 */
export function SaveView(props: { route: Route; level: string; placed: number; onClose: () => void; onSaved: () => void }) {
  const views = useApi<ViewsResponse>('/api/views')
  const [name, setName] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const view = {
    name: name.trim(),
    level: props.level,
    ...(props.route.focus ? { focus: props.route.focus } : {}),
    ...(props.route.q ? { find: props.route.q } : {}),
    ...(props.route.dim === '1' ? { dim: true } : {}),
  }
  const existing = views.data?.views.some((v) => v.name === view.name)
  const yaml = [
    `- name: ${view.name || 'Untitled view'}`,
    `  level: ${view.level}`,
    ...(view.focus ? [`  focus: ${view.focus}`] : []),
    ...(view.find ? [`  find: ${view.find}`] : []),
    ...(view.dim ? ['  dim: true'] : []),
  ].join('\n')

  const save = async () => {
    if (!view.name || !views.data) return
    setBusy(true)
    setError(null)
    try {
      await action('/api/views/save', { view, base: views.data.hash })
      props.onSaved()
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="dialog-scrim" onClick={props.onClose}>
      <div className="dialog" role="dialog" aria-label="Save view" onClick={(e) => e.stopPropagation()}>
        <Eyebrow>Save view</Eyebrow>
        <input
          autoFocus
          className="dialog-input"
          value={name}
          placeholder="Name this view"
          onChange={(e) => setName(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') save()
            if (e.key === 'Escape') props.onClose()
          }}
        />
        <pre className="dialog-yaml">{yaml}</pre>
        <p className="dialog-note">
          {existing ? 'Replaces the view of the same name in ' : 'Appended to '}
          <span className="mono">{views.data?.file ?? '.archdoc/views.yaml'}</span>. Commit it and the view appears under
          Architecture for your team.
          {props.placed > 0 && ' The arrangement is not part of the view: it lives in layout.yaml and applies to every view of this level.'}
        </p>
        {error && <p className="dialog-error">{error}</p>}
        <div className="dialog-actions">
          <button className="tool" onClick={props.onClose}>
            Cancel
          </button>
          <button className="tool on" onClick={save} disabled={!view.name || busy || !views.data}>
            {existing ? 'Replace view' : 'Save view'}
          </button>
        </div>
      </div>
    </div>
  )
}
