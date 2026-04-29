export interface Image {
  id: number
  filename: string
  capture_date: string
  width: number
  height: number
  rating: number
  mime_type: string
  thumbnail_path: string
}

export interface Tag {
  id: number
  name: string
}

export interface ImagesResponse {
  data: Image[]
  total: number
  page: number
  limit: number
}

export interface ImageParams {
  page: number
  sort?: string
  order?: string
  minRating?: number
}

export async function fetchImages(params: ImageParams): Promise<ImagesResponse> {
  const query = new URLSearchParams({ page: String(params.page), limit: '50' })
  if (params.sort) query.set('sort', params.sort)
  if (params.order) query.set('order', params.order)
  if (params.minRating !== undefined) query.set('rating', String(params.minRating))

  const res = await fetch(`/api/images?${query}`)
  if (!res.ok) throw new Error('Failed to fetch images')
  return res.json()
}

export async function fetchImageTags(id: number): Promise<Tag[]> {
  const res = await fetch(`/api/images/${id}/tags`)
  if (!res.ok) throw new Error('Failed to fetch image tags')
  return res.json()
}

export async function fetchTags(): Promise<Tag[]> {
  const res = await fetch('/api/tags')
  if (!res.ok) throw new Error('Failed to fetch tags')
  return res.json()
}

export async function patchImage(id: number, updates: { rating?: number; tags?: number[] }): Promise<Image> {
  const res = await fetch(`/api/images/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  })
  if (!res.ok) throw new Error('Failed to update image')
  return res.json()
}

export async function createTag(name: string): Promise<Tag> {
  const res = await fetch('/api/tags', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  })
  if (!res.ok) throw new Error('Failed to create tag')
  return res.json()
}

export interface ScanStatus {
  running: boolean
  total: number
  processed: number
  errors: number
}

export async function fetchScanStatus(): Promise<ScanStatus> {
  const res = await fetch('/api/scan/status')
  if (!res.ok) throw new Error('Failed to fetch scan status')
  return res.json()
}

export async function triggerScan(): Promise<void> {
  const res = await fetch('/api/scan', { method: 'POST' })
  if (!res.ok && res.status !== 409) throw new Error('Failed to start scan')
}
