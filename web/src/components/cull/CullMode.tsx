import { useState, useEffect, useRef } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import type { Image } from '../../api/images'
import { patchImage } from '../../api/images'
import StarRating from '../stars/StarRating'
import './CullMode.css'

interface CullModeProps {
  images: Image[]
  onExit: () => void
}

export default function CullMode({ images, onExit }: CullModeProps) {
  const queryClient = useQueryClient()
  const [currentIndex, setCurrentIndex] = useState(0)
  const [ratingOverrides, setRatingOverrides] = useState<Record<number, number>>({})
  const thumbRefs = useRef<(HTMLDivElement | null)[]>([])

  const currentImage = images[currentIndex]

  const ratingMutation = useMutation({
    mutationFn: ({ id, rating }: { id: number; rating: number }) => patchImage(id, { rating }),
    onMutate: ({ id, rating }) => {
      setRatingOverrides(prev => ({ ...prev, [id]: rating }))
    },
    onError: (_, { id }) => {
      setRatingOverrides(prev => {
        const next = { ...prev }
        delete next[id]
        return next
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['images'] })
    },
  })

  function getEffectiveRating(img: Image) {
    return ratingOverrides[img.id] ?? img.rating
  }

  function rate(rating: number) {
    if (!currentImage) return
    // X toggles: reject → unrated, unrated → reject
    const current = getEffectiveRating(currentImage)
    const newRating = rating === -1 && current === -1 ? 0 : rating
    ratingMutation.mutate({ id: currentImage.id, rating: newRating })
  }

  // Scroll active thumbnail into view
  useEffect(() => {
    thumbRefs.current[currentIndex]?.scrollIntoView({
      behavior: 'smooth',
      block: 'nearest',
      inline: 'center',
    })
  }, [currentIndex])

  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if (e.target instanceof HTMLInputElement) return
      if (e.key === 'ArrowLeft') setCurrentIndex(i => Math.max(0, i - 1))
      if (e.key === 'ArrowRight') setCurrentIndex(i => Math.min(images.length - 1, i + 1))
      if (e.key === 'Escape') onExit()
      if (e.key === 'x' || e.key === 'X') rate(-1)
      const n = parseInt(e.key)
      if (!isNaN(n) && n >= 0 && n <= 5) rate(n)
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [images.length, onExit, currentImage?.id])

  if (!currentImage) {
    return (
      <div className="cull-empty">
        <p>No images in this selection.</p>
        <button onClick={onExit}>Back to gallery</button>
      </div>
    )
  }

  const exifParts = [
    currentImage.camera_model,
    currentImage.iso ? `ISO ${currentImage.iso}` : null,
    currentImage.aperture != null ? `f/${currentImage.aperture}` : null,
    currentImage.shutter_speed,
    currentImage.focal_length != null ? `${currentImage.focal_length}mm` : null,
  ].filter(Boolean)

  const captureDate = currentImage.capture_date
    ? new Date(currentImage.capture_date).toLocaleDateString('en-GB', {
        day: 'numeric', month: 'short', year: 'numeric',
      })
    : null

  return (
    <div className="cull-mode">
      <div className="cull-image-area">
        <img
          key={currentImage.id}
          src={`/api/images/${currentImage.id}/full`}
          alt={currentImage.filename}
          className="cull-image"
        />
        <div className="cull-overlay">
          <div className="cull-info">
            <span className="cull-filename">{currentImage.filename}</span>
            <span className="cull-meta">
              {[captureDate, ...exifParts].filter(Boolean).join(' · ')}
            </span>
          </div>
          <div className="cull-controls">
            <StarRating
              value={getEffectiveRating(currentImage)}
              onChange={rate}
              disabled={ratingMutation.isPending}
            />
            <span className="cull-counter">{currentIndex + 1} / {images.length}</span>
          </div>
        </div>
      </div>

      <div className="cull-legend">
        <span><kbd>←</kbd><kbd>→</kbd> navigate</span>
        <span><kbd>1</kbd>–<kbd>5</kbd> rate</span>
        <span><kbd>x</kbd> reject / unreject</span>
        <span><kbd>Esc</kbd> exit</span>
      </div>

      <div className="cull-filmstrip">
        {images.map((img, i) => {
          const r = getEffectiveRating(img)
          return (
            <div
              key={img.id}
              ref={el => { thumbRefs.current[i] = el }}
              className={`cull-thumb${i === currentIndex ? ' active' : ''}`}
              onClick={() => setCurrentIndex(i)}
            >
              <img src={img.thumbnail_path} alt={img.filename} />
              {r === -1 && <span className="cull-thumb-badge reject">✕</span>}
              {r > 0 && <span className="cull-thumb-badge stars">{'★'.repeat(r)}</span>}
            </div>
          )
        })}
      </div>
    </div>
  )
}
