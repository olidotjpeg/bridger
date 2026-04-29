import { useEffect, useRef } from "react"
import type { Image } from "../../api/images"
import "./Gallery.css"

interface GalleryListProps {
    images: Image[] | null;
    selectedId: number | null;
    selectedIds: Set<number>;
    onSelectId: (id: number) => void;
    onToggleSelect: (id: number) => void;
    onRangeSelect: (fromIndex: number, toIndex: number) => void;
    lastSelectedIndex: number | null;
    onSetLastSelectedIndex: (index: number) => void;
}

export default function GalleryList({
    images,
    selectedId,
    selectedIds,
    onSelectId,
    onToggleSelect,
    onRangeSelect,
    lastSelectedIndex,
    onSetLastSelectedIndex,
}: GalleryListProps) {
    const listRef = useRef<HTMLUListElement>(null)

    useEffect(() => {
        function handleKeyDown(e: KeyboardEvent) {
            if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return
            if (selectedId !== null) return
            if (!images || images.length === 0) return

            if (e.key === ' ') {
                e.preventDefault()
                const idx = lastSelectedIndex ?? 0
                const img = images[idx]
                if (img) onToggleSelect(img.id)
            } else if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
                e.preventDefault()
                onSetLastSelectedIndex(Math.min((lastSelectedIndex ?? -1) + 1, images.length - 1))
            } else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
                e.preventDefault()
                onSetLastSelectedIndex(Math.max((lastSelectedIndex ?? images.length) - 1, 0))
            }
        }
        window.addEventListener('keydown', handleKeyDown)
        return () => window.removeEventListener('keydown', handleKeyDown)
    }, [images, selectedId, lastSelectedIndex, onToggleSelect, onSetLastSelectedIndex])

    if (!images) {
        return "No images arrived"
    }

    function handleClick(e: React.MouseEvent, imageId: number, index: number) {
        if (e.shiftKey) {
            if (lastSelectedIndex !== null && lastSelectedIndex !== index) {
                onRangeSelect(lastSelectedIndex, index)
            } else {
                onToggleSelect(imageId)
            }
            onSetLastSelectedIndex(index)
        } else if (e.metaKey || e.ctrlKey) {
            onToggleSelect(imageId)
            onSetLastSelectedIndex(index)
        } else {
            onSetLastSelectedIndex(index)
            onSelectId(imageId)
        }
    }

    return (
        <ul className="gallery" ref={listRef}>
            {images.map((image, index) => (
                <li
                    key={image.id}
                    className={[
                        selectedId === image.id ? 'active' : '',
                        selectedIds.has(image.id) ? 'selected' : '',
                        lastSelectedIndex === index ? 'cursor' : '',
                    ].filter(Boolean).join(' ')}
                    onClick={e => handleClick(e, image.id, index)}
                >
                    <img src={image.thumbnail_path} alt={image.filename} />
                    {selectedIds.has(image.id) && <span className="gallery-check">✓</span>}
                    {image.rating > 0 && (
                        <span className="gallery-rating">{'★'.repeat(image.rating)}</span>
                    )}
                </li>
            ))}
        </ul>
    )
}
