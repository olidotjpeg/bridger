import type { Image } from "../../api/images"
import "./Gallery.css"

interface GalleryListProps {
    images: Image[] | null;
    selectedId: number | null;
    selectedIds: Set<number>;
    onSelectId: (id: number) => void;
    onToggleSelect: (id: number) => void;
}

export default function GalleryList({ images, selectedId, selectedIds, onSelectId, onToggleSelect }: GalleryListProps) {
    if (!images) {
        return "No images arrived"
    }

    function handleClick(e: React.MouseEvent, imageId: number) {
        if (e.shiftKey) {
            onToggleSelect(imageId)
        } else {
            onSelectId(imageId)
        }
    }

    return (
        <ul className="gallery">
            {images.map(image => (
                <li
                    key={image.id}
                    className={[
                        selectedId === image.id ? 'active' : '',
                        selectedIds.has(image.id) ? 'selected' : '',
                    ].filter(Boolean).join(' ')}
                    onClick={e => handleClick(e, image.id)}
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
