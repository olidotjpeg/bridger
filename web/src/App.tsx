import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import './App.css'
import GalleryList from './components/gallery/GalleryList'
import LightBox from './components/lightbox/Lightbox'
import { fetchImages } from './api/images'

function App() {
  const [page, setPage] = useState(1)
  const [selectedId, setSelectedId] = useState<number | null>(null)

  const { data, isLoading, isError } = useQuery({
    queryKey: ['images', page],
    queryFn: () => fetchImages(page)
  })

  const totalPages = Math.ceil((data?.total ?? 0) / (data?.limit ?? 50))

  if (isLoading) return <p>Loading....</p>
  if (isError) return <p>Something went wrong womp womp</p>

  return (
    <div>
      {data?.data && <GalleryList images={data.data} onSelectId={setSelectedId} />}

      <button onClick={() => setPage(p => p - 1)} disabled={page === 1}>Prev</button>
      <button onClick={() => setPage(p => p + 1)} disabled={page >= totalPages}>Next</button>

      {data?.data && selectedId && (
        <LightBox
          images={data.data}
          selectedId={selectedId}
          onClose={() => setSelectedId(null)}
          onNavigate={setSelectedId}
        />
      )}
    </div>
  )
}

export default App
