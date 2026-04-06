import { useEffect } from "react";
import type { Image } from "../../api/images";
import "./Lightbox.css";

interface LightboxProps {
  images: Image[];
  selectedId: number | null;
  onClose: () => void;
  onNavigate: (id: number) => void;
}

export default function LightBox({
  images,
  selectedId,
  onClose,
  onNavigate,
}: LightboxProps) {
  const currentIndex = images.findIndex((img) => img.id === selectedId);
  const currentImage = images[currentIndex]

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
      if (e.key === "ArrowRight") {
        const next = images[currentIndex + 1];
        if (next) onNavigate(next.id);
      }
      if (e.key === "ArrowLeft") {
        const prev = images[currentIndex - 1];
        if (prev) onNavigate(prev.id);
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [selectedId, onClose, onNavigate, images, currentIndex]);

  if (!currentImage) return null

  return (
    <div className="lightbox">
      <div className="lightbox-wrapper">
        <img src={`/api/images/${selectedId}/full`} alt={currentImage.filename} />
        <div className="meta">
          <p>Capture date {currentImage.capture_date}</p>
          <p>W/H {currentImage.width} / {currentImage.height}</p>
          <p>Name {currentImage.filename}</p>
          <p>MimeType {currentImage.mime_type}</p>
        </div>
      </div>
    </div>
  );
}
