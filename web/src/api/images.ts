export interface Image {
  id: number
  filename: string
  capture_date: string
  width: number
  height: number
  mime_type: string
  thumbnail_path: string
}

export interface ImagesResponse {
  data: Image[]
  total: number
  page: number
  limit: number
}

export async function fetchImages(page: number): Promise<ImagesResponse> {
  const res = await fetch(`/api/images?page=${page}&limit=50`)
  if (!res.ok) throw new Error('Failed to fetch images')
  return res.json()
}
