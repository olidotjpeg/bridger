import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchProjects, deleteProject, type Project } from '../../api/projects'
import ProjectModal from './ProjectModal'
import './ProjectSidebar.css'

interface ProjectSidebarProps {
  selectedProjectId: number | null
  onSelect: (id: number | null) => void
}

export default function ProjectSidebar({ selectedProjectId, onSelect }: ProjectSidebarProps) {
  const queryClient = useQueryClient()
  const [modalOpen, setModalOpen] = useState(false)
  const [editingProject, setEditingProject] = useState<Project | undefined>()

  const { data: projects = [] } = useQuery({
    queryKey: ['projects'],
    queryFn: fetchProjects,
  })

  const deleteMutation = useMutation({
    mutationFn: deleteProject,
    onSuccess: (_, deletedId) => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      if (selectedProjectId === deletedId) onSelect(null)
    },
  })

  function openCreate() {
    setEditingProject(undefined)
    setModalOpen(true)
  }

  function openEdit(project: Project, e: React.MouseEvent) {
    e.stopPropagation()
    setEditingProject(project)
    setModalOpen(true)
  }

  function handleDelete(project: Project, e: React.MouseEvent) {
    e.stopPropagation()
    if (confirm(`Delete project "${project.name}"?\nImages will not be deleted.`)) {
      deleteMutation.mutate(project.id)
    }
  }

  return (
    <>
      <div className="project-sidebar-section">
        <div className="project-sidebar-header">
          <span className="sidebar-label">Projects</span>
          <button className="project-new-btn" onClick={openCreate} title="New project">+</button>
        </div>

        <div className="project-list">
          <button
            className={`project-item ${selectedProjectId === null ? 'active' : ''}`}
            onClick={() => onSelect(null)}
          >
            <span className="project-item-name">All photos</span>
          </button>

          {projects.map(project => (
            <div
              key={project.id}
              className={`project-item-wrap ${selectedProjectId === project.id ? 'active' : ''}`}
              onClick={() => onSelect(project.id)}
            >
              <span className="project-item-name">{project.name}</span>
              <div className="project-item-actions">
                <button
                  className="project-item-action"
                  title="Edit"
                  onClick={e => openEdit(project, e)}
                >
                  ✎
                </button>
                <button
                  className="project-item-action"
                  title="Delete"
                  onClick={e => handleDelete(project, e)}
                >
                  ×
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>

      {modalOpen && (
        <ProjectModal
          project={editingProject}
          onClose={() => setModalOpen(false)}
        />
      )}
    </>
  )
}
