import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import './App.css'
import GalleryList from './components/gallery/GalleryList'
import LightBox from './components/lightbox/Lightbox'

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

async function fetchImages(page: number): Promise<ImagesResponse> {
  const res = await fetch(`/api/images?page=${page}&limit=50`)
  if (!res.ok) throw new Error('Failed to fetch images')
  return res.json()
}

function App() {
  const [page, setPage] = useState(1)
  const [selectedId, setSelectedId] = useState(0)

  const { data, isLoading, isError } = useQuery({
    queryKey: ['images', page],
    queryFn: () => fetchImages(page)
  })

  const totalPages = Math.ceil((data?.total ?? 0) / (data?.limit ?? 50))

  if (isLoading) return <p>Loading....</p>
  if (isError) return <p>Something went wrong womp womp</p>

  return (
    <div>
      {data?.data ? <GalleryList images={data?.data} onSelectId={setSelectedId}></GalleryList> : <></>}

      <button onClick={() => setPage(p => p - 1)} disabled={page === 1}>Prev</button>
      <button onClick={() => setPage(p => p + 1)} disabled={page >= totalPages}>Next</button>

      {data?.data && selectedId ?
        <LightBox
          images={data?.data}
          selectedId={selectedId}
          onClose={() => setSelectedId(0)}
          onNavigate={setSelectedId}
        />
       : <></>
      }
    </div>
  )
}

export default App
