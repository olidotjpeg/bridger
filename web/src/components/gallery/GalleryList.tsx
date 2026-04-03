import type { Image } from "../../App"
import "./Gallery.css"

interface IProps {
    images: Image[] | null;
    onSelectId: (id: number) => void;
}

export default function GalleryList({images, onSelectId}: IProps) {
    if (!images) {
        return "No images arrived"
    }

    return (
        <ul className="gallery">
            {images.map(image => (
                <li key={image.id} onClick={() => onSelectId(image.id)}>
                    <img src={image.thumbnail_path} />
                </li>
            ))}
        </ul>
    )
}