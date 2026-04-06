import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import GalleryList from './components/gallery/GalleryList'
import LightBox from './components/lightbox/Lightbox'
import Sidebar from './components/sidebar/Sidebar'
import BulkActionBar from './components/bulk/BulkActionBar'
import { fetchImages, patchImage } from './api/images'
import './App.css'

function App() {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())
  const [sort, setSort] = useState('capture_date')
  const [order, setOrder] = useState('desc')
  const [minRating, setMinRating] = useState<number | undefined>(undefined)

  const { data, isLoading, isError } = useQuery({
    queryKey: ['images', page, sort, order, minRating],
    queryFn: () => fetchImages({ page, sort, order, minRating })
  })

  const totalPages = Math.ceil((data?.total ?? 0) / (data?.limit ?? 50))

  const bulkRatingMutation = useMutation({
    mutationFn: async (rating: number) => {
      await Promise.all([...selectedIds].map(id => patchImage(id, { rating })))
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['images'] })
      setSelectedIds(new Set())
    },
  })

  function toggleSelection(id: number) {
    setSelectedIds(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function clearSelection() {
    setSelectedIds(new Set())
  }

  function resetPage() {
    setPage(1)
    setSelectedId(null)
    clearSelection()
  }

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.target instanceof HTMLInputElement) return
      if (selectedIds.size === 0) return
      if (e.key === 'x' || e.key === 'X') bulkRatingMutation.mutate(-1)
      if (e.key === 'Escape' && !selectedId) clearSelection()
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [selectedIds, selectedId])

  return (
    <div className="app">
      <aside className="app-sidebar">
        <Sidebar
          sort={sort}
          order={order}
          minRating={minRating}
          onSortChange={v => { setSort(v); resetPage() }}
          onOrderChange={v => { setOrder(v); resetPage() }}
          onRatingChange={v => { setMinRating(v); resetPage() }}
        />
      </aside>

      <main className="app-main">
        {isLoading && <div className="status-message">Loading...</div>}
        {isError && <div className="status-message error">Failed to load images</div>}
        {data?.data && (
          <div className="gallery-container">
            <GalleryList
              images={data.data}
              selectedId={selectedId}
              selectedIds={selectedIds}
              onSelectId={setSelectedId}
              onToggleSelect={toggleSelection}
            />
          </div>
        )}
        <div className="pagination">
          <button onClick={() => setPage(p => p - 1)} disabled={page === 1}>← Prev</button>
          <span className="page-info">Page {page} of {totalPages}</span>
          <button onClick={() => setPage(p => p + 1)} disabled={page >= totalPages}>Next →</button>
        </div>
      </main>

      {data?.data && selectedId && (
        <LightBox
          images={data.data}
          selectedId={selectedId}
          onClose={() => setSelectedId(null)}
          onNavigate={setSelectedId}
        />
      )}

      {selectedIds.size > 0 && (
        <BulkActionBar
          count={selectedIds.size}
          onSetRating={rating => bulkRatingMutation.mutate(rating)}
          onClear={clearSelection}
        />
      )}
    </div>
  )
}

export default App
