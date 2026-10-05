import { TruthMark } from '../../ui/marks'

/** The legend (surface-spec §5.2): truth states, evidence, kinds. Always one click away. */
export function Legend({ onClose }: { onClose: () => void }) {
  return (
    <div className="legend" role="dialog" aria-label="Legend">
      <div className="legend-head">
        <span className="eyebrow">Legend</span>
        <button onClick={onClose} aria-label="Close legend">
          ×
        </button>
      </div>
      <div className="legend-grid">
        <TruthMark state="proven" />
        <div>
          <strong>Proven</strong> — read from a file, cited
        </div>
        <TruthMark state="interpreted" />
        <div>
          <strong className="interpreted">Interpreted</strong> — model-written, cited
        </div>
        <TruthMark state="unresolved" />
        <div>
          <strong className="legend-unres">Unresolved</strong> — seen, not identified
        </div>
        <div className="legend-rule" />
        <span className="legend-swatch" style={{ borderColor: 'var(--k-slate)' }} />
        <div>Declared — the repository defines it</div>
        <span className="legend-swatch dashed" style={{ borderColor: 'var(--k-violet)' }} />
        <div>Referenced — the repository only names it</div>
        <div className="legend-rule" />
        <span className="legend-swatch filled" style={{ background: 'var(--k-slate-fill)', borderColor: 'var(--k-slate)' }} />
        <div>Container · system · proxy</div>
        <span className="legend-swatch filled" style={{ background: 'var(--k-teal-fill)', borderColor: 'var(--k-teal)' }} />
        <div>Data store · queue</div>
        <span className="legend-swatch filled" style={{ background: 'var(--k-green-fill)', borderColor: 'var(--k-green)' }} />
        <div>Person</div>
        <span className="legend-swatch filled dashed" style={{ background: 'var(--k-violet-fill)', borderColor: 'var(--k-violet)' }} />
        <div>External system</div>
      </div>
    </div>
  )
}
