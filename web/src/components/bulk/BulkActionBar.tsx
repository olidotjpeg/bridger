import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { fetchTags } from '../../api/images'
import type { Tag } from '../../api/images'
import './BulkActionBar.css'

interface BulkActionBarProps {
  count: number
  onSetRating: (rating: number) => void
  onAddTag: (tag: Tag) => void
  onClear: () => void
}

export default function BulkActionBar({ count, onSetRating, onAddTag, onClear }: BulkActionBarProps) {
  const [tagInput, setTagInput] = useState('')
  const [tagOpen, setTagOpen] = useState(false)

  const { data: allTags = [] } = useQuery({
    queryKey: ['tags'],
    queryFn: fetchTags,
  })

  const filtered = allTags.filter(t =>
    t.name.toLowerCase().includes(tagInput.toLowerCase())
  )

  function handleTagSelect(tag: Tag) {
    onAddTag(tag)
    setTagInput('')
    setTagOpen(false)
  }

  return (
    <div className="bulk-bar">
      <div className="bulk-bar-left">
        <span className="bulk-count">{count} selected</span>
        <button className="bulk-clear" onClick={onClear}>Clear</button>
      </div>

      <div className="bulk-bar-center">
        <div className="bulk-tag-wrapper">
          <input
            className="bulk-tag-input"
            placeholder="Tag all…"
            value={tagInput}
            onChange={e => { setTagInput(e.target.value); setTagOpen(true) }}
            onFocus={() => setTagOpen(true)}
            onBlur={() => setTimeout(() => setTagOpen(false), 150)}
            onKeyDown={e => {
              e.stopPropagation()
              if (e.key === 'Escape') { setTagInput(''); setTagOpen(false) }
              if (e.key === 'Enter' && filtered.length > 0) handleTagSelect(filtered[0])
            }}
          />
          {tagOpen && filtered.length > 0 && (
            <div className="bulk-tag-dropdown">
              {filtered.map(tag => (
                <button key={tag.id} className="bulk-tag-option" onMouseDown={() => handleTagSelect(tag)}>
                  {tag.name}
                </button>
              ))}
            </div>
          )}
        </div>
      </div>

      <div className="bulk-bar-right">
        <span className="bulk-label">Rate:</span>
        {[1, 2, 3, 4, 5].map(n => (
          <button key={n} className="bulk-star" onClick={() => onSetRating(n)}>
            {'★'.repeat(n)}
          </button>
        ))}
        <button className="bulk-reject" onClick={() => onSetRating(-1)}>
          Reject
        </button>
      </div>
    </div>
  )
}
