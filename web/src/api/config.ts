export interface AppConfig {
  needs_setup: boolean
  db_path: string
  thumbs_path: string
}

export async function fetchConfig(): Promise<AppConfig> {
  const res = await fetch('/api/config')
  if (!res.ok) throw new Error('Failed to fetch config')
  return res.json()
}

export async function saveConfig(scanDirs: string[]): Promise<void> {
  const res = await fetch('/api/config', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ scan_dirs: scanDirs }),
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.message ?? 'Failed to save config')
  }
}

