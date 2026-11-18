/** The keyboard shortcut sheet (surface-spec §6.13), opened with ?. */
export function Shortcuts({ onClose }: { onClose: () => void }) {
  const groups: [string, [string, string[]][]][] = [
    [
      'Everywhere',
      [
        ['Search, or ask', ['⌘', 'K']],
        ['Close search, a menu, the find box', ['Esc']],
        ['These keys', ['?']],
      ],
    ],
    [
      'On a diagram',
      [
        ['Find a part', ['/']],
        ['Select the first match', ['↵']],
        ['Clear the find box', ['Esc']],
        ['Pan', ['drag the canvas']],
        ['Arrange', ['drag a box, or a boundary by its name']],
        ['Undo a move', ['⌘', 'Z']],
        ['Redo it', ['⇧', '⌘', 'Z']],
        ['Lock the arrangement', ['the padlock, by the zoom']],
      ],
    ],
  ]
  return (
    <div className="dialog-scrim" onClick={onClose}>
      <div className="dialog shortcuts" role="dialog" aria-label="Keyboard shortcuts" onClick={(e) => e.stopPropagation()}>
        <div className="shortcuts-head">
          <span className="eyebrow">Keyboard shortcuts</span>
          <button className="text-link" onClick={onClose}>
            Close
          </button>
        </div>
        <div className="shortcuts-grid">
          {groups.map(([g, keys]) => (
            <div key={g}>
              <div className="shortcuts-group">{g}</div>
              {keys.map(([d, ks]) => (
                <div key={d} className="shortcut">
                  <span>{d}</span>
                  <span className="shortcut-keys">
                    {ks.map((k) => (
                      <kbd key={k}>{k}</kbd>
                    ))}
                  </span>
                </div>
              ))}
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
