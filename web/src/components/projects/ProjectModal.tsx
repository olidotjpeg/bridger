import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { createProject, updateProject, type Project } from '../../api/projects'
import './ProjectModal.css'

interface ProjectModalProps {
  project?: Project
  onClose: () => void
}

export default function ProjectModal({ project, onClose }: ProjectModalProps) {
  const queryClient = useQueryClient()
  const [name, setName] = useState(project?.name ?? '')
  const [dirs, setDirs] = useState<string[]>(project?.dirs ?? [])
  const [error, setError] = useState<string | null>(null)

  const isEdit = !!project

  const mutation = useMutation({
    mutationFn: () =>
      isEdit
        ? updateProject(project.id, name, dirs)
        : createProject(name, dirs),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      onClose()
    },
    onError: (e: Error) => setError(e.message),
  })

  async function pickFolder() {
    const path: string = await window.go!.main.App.PickFolder()
    if (path) {
      setDirs(prev => prev.includes(path) ? prev : [...prev, path])
    }
  }

  function removeDir(dir: string) {
    setDirs(prev => prev.filter(d => d !== dir))
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim()) { setError('Name is required'); return }
    setError(null)
    mutation.mutate()
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-card" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <span className="modal-title">{isEdit ? 'Edit project' : 'New project'}</span>
          <button className="modal-close" onClick={onClose}>×</button>
        </div>

        <form className="modal-body" onSubmit={handleSubmit}>
          <div className="modal-field">
            <label className="modal-label">Name</label>
            <input
              className="modal-input"
              type="text"
              placeholder="e.g. Wedding 2024"
              value={name}
              onChange={e => setName(e.target.value)}
              autoFocus
            />
          </div>

          <div className="modal-field">
            <label className="modal-label">Folders</label>
            <div className="modal-folder-list">
              {dirs.length === 0 && (
                <p className="modal-empty">No folders added yet.</p>
              )}
              {dirs.map(d => (
                <div key={d} className="modal-folder-row">
                  <span className="modal-folder-path">{d}</span>
                  <button type="button" className="modal-folder-remove" onClick={() => removeDir(d)}>×</button>
                </div>
              ))}
            </div>
            <button type="button" className="modal-btn-secondary" onClick={pickFolder}>
              + Add folder
            </button>
          </div>

          {error && <p className="modal-error">{error}</p>}

          <div className="modal-actions">
            <button type="button" className="modal-btn-ghost" onClick={onClose}>Cancel</button>
            <button type="submit" className="modal-btn-primary" disabled={mutation.isPending}>
              {mutation.isPending ? 'Saving…' : isEdit ? 'Save' : 'Create'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
