import { useState, useRef, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { fetchTags } from '../../api/images'
import type { Tag } from '../../api/images'
import './TagEditor.css'

interface TagEditorProps {
  tags: Tag[]
  onAdd: (tag: Tag) => void
  onRemove: (tagId: number) => void
  onCreateAndAdd: (name: string) => void
  disabled?: boolean
}

export default function TagEditor({ tags, onAdd, onRemove, onCreateAndAdd, disabled }: TagEditorProps) {
  const [input, setInput] = useState('')
  const [open, setOpen] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  const { data: allTags = [] } = useQuery({
    queryKey: ['tags'],
    queryFn: fetchTags,
  })

  const currentIds = new Set(tags.map(t => t.id))
  const filtered = allTags.filter(
    t => !currentIds.has(t.id) && t.name.toLowerCase().includes(input.toLowerCase())
  )
  const exactMatch = allTags.some(t => t.name.toLowerCase() === input.toLowerCase())
  const showCreate = input.trim().length > 0 && !exactMatch

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  function handleSelect(tag: Tag) {
    onAdd(tag)
    setInput('')
    inputRef.current?.focus()
  }

  function handleCreate() {
    const name = input.trim()
    if (!name) return
    onCreateAndAdd(name)
    setInput('')
    inputRef.current?.focus()
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault()
      if (filtered.length > 0 && !showCreate) handleSelect(filtered[0])
      else if (showCreate) handleCreate()
    }
    if (e.key === 'Escape') {
      setOpen(false)
      setInput('')
    }
    e.stopPropagation() // prevent lightbox keyboard shortcuts
  }

  return (
    <div className="tag-editor" ref={containerRef}>
      <div className="tag-list">
        {tags.map(tag => (
          <span key={tag.id} className="tag-chip">
            {tag.name}
            <button
              className="tag-remove"
              onClick={() => onRemove(tag.id)}
              disabled={disabled}
              aria-label={`Remove ${tag.name}`}
            >
              ×
            </button>
          </span>
        ))}

        <div className="tag-input-wrapper">
          <input
            ref={inputRef}
            className="tag-input"
            placeholder="Add tag…"
            value={input}
            onChange={e => { setInput(e.target.value); setOpen(true) }}
            onFocus={() => setOpen(true)}
            onKeyDown={handleKeyDown}
            disabled={disabled}
          />

          {open && (filtered.length > 0 || showCreate) && (
            <div className="tag-dropdown">
              {filtered.map(tag => (
                <button key={tag.id} className="tag-option" onMouseDown={() => handleSelect(tag)}>
                  {tag.name}
                </button>
              ))}
              {showCreate && (
                <button className="tag-option tag-option-create" onMouseDown={handleCreate}>
                  Create "{input.trim()}"
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
