import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import GalleryList from './components/gallery/GalleryList'
import LightBox from './components/lightbox/Lightbox'
import Sidebar from './components/sidebar/Sidebar'
import BulkActionBar from './components/bulk/BulkActionBar'
import Setup from './components/setup/Setup'
import { fetchImages, fetchScanStatus, triggerScan, patchImage, fetchImageTags } from './api/images'
import { fetchConfig } from './api/config'
import type { Tag } from './api/images'
import './App.css'

function App() {
  const queryClient = useQueryClient()

  const { data: appConfig, isLoading: configLoading } = useQuery({
    queryKey: ['config'],
    queryFn: fetchConfig,
    staleTime: Infinity,
  })

  function handleSetupComplete() {
    queryClient.invalidateQueries({ queryKey: ['config'] })
    queryClient.invalidateQueries({ queryKey: ['images'] })
  }

  const [page, setPage] = useState(1)
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())
  const [lastSelectedIndex, setLastSelectedIndex] = useState<number | null>(null)
  const [sort, setSort] = useState('capture_date')
  const [order, setOrder] = useState('desc')
  const [minRating, setMinRating] = useState<number | undefined>(undefined)

  const { data, isLoading, isError } = useQuery({
    queryKey: ['images', page, sort, order, minRating],
    queryFn: () => fetchImages({ page, sort, order, minRating })
  })

  const { data: scanStatus } = useQuery({
    queryKey: ['scan-status'],
    queryFn: fetchScanStatus,
    refetchInterval: (query) => query.state.data?.running ? 2000 : false,
  })

  const scanMutation = useMutation({
    mutationFn: triggerScan,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['scan-status'] })
    },
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

  const bulkTagMutation = useMutation({
    mutationFn: async (tag: Tag) => {
      // For each selected image, fetch its current tags, then add the new one
      await Promise.all([...selectedIds].map(async (id) => {
        const currentTags = await fetchImageTags(id)
        const tagIds = currentTags.map(t => t.id)
        if (!tagIds.includes(tag.id)) {
          await patchImage(id, { tags: [...tagIds, tag.id] })
        }
      }))
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['images'] })
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

  function rangeSelect(fromIndex: number, toIndex: number) {
    if (!data?.data) return
    const start = Math.min(fromIndex, toIndex)
    const end = Math.max(fromIndex, toIndex)
    setSelectedIds(prev => {
      const next = new Set(prev)
      for (let i = start; i <= end; i++) {
        const img = data.data[i]
        if (img) next.add(img.id)
      }
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

  // Invalidate images query once scan finishes (running flips from true → false)
  const wasRunning = scanStatus?.running
  useEffect(() => {
    if (wasRunning === false) {
      queryClient.invalidateQueries({ queryKey: ['images'] })
    }
  }, [wasRunning])

  if (configLoading) return <div className="status-message">Starting…</div>
  if (appConfig?.needs_setup) return <Setup config={appConfig} onComplete={handleSetupComplete} />

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
          scanStatus={scanStatus}
          onTriggerScan={() => scanMutation.mutate()}
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
              onRangeSelect={rangeSelect}
              lastSelectedIndex={lastSelectedIndex}
              onSetLastSelectedIndex={setLastSelectedIndex}
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
          onAddTag={tag => bulkTagMutation.mutate(tag)}
          onClear={clearSelection}
        />
      )}
    </div>
  )
}

export default App
