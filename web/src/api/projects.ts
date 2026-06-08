export interface Project {
  id: number
  name: string
  dirs: string[]
  created_at: string
}

export async function fetchProjects(): Promise<Project[]> {
  const res = await fetch('/api/projects')
  if (!res.ok) throw new Error('Failed to fetch projects')
  return res.json()
}

export async function createProject(name: string, dirs: string[]): Promise<Project> {
  const res = await fetch('/api/projects', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, dirs }),
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.message ?? 'Failed to create project')
  }
  return res.json()
}

export async function updateProject(id: number, name?: string, dirs?: string[]): Promise<Project> {
  const res = await fetch(`/api/projects/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, dirs }),
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.message ?? 'Failed to update project')
  }
  return res.json()
}

export async function deleteProject(id: number): Promise<void> {
  const res = await fetch(`/api/projects/${id}`, { method: 'DELETE' })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.message ?? 'Failed to delete project')
  }
}
