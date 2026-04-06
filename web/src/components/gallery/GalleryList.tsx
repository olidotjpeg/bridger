import type { Image } from "../../api/images"
import "./Gallery.css"

interface GalleryListProps {
    images: Image[] | null;
    onSelectId: (id: number) => void;
}

export default function GalleryList({images, onSelectId}: GalleryListProps) {
    if (!images) {
        return "No images arrived"
    }

    return (
        <ul className="gallery">
            {images.map(image => (
                <li key={image.id} onClick={() => onSelectId(image.id)}>
                    <img src={image.thumbnail_path} alt={image.filename} />
                </li>
            ))}
        </ul>
    )
}