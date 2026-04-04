import { useEffect } from "react";
import type { Image } from "../../App";
import "./Lightbox.css";

interface IProps {
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
}: IProps) {
  const currentIndex = images.findIndex((img) => img.id === selectedId);

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
  }, [selectedId, onClose, onNavigate]);

  return (
    <div className="lightbox">
      <div className="lightbox-wrapper">
        <img src={`api/images/${selectedId}/full`} />
        <div className="meta">
          <p>Capture date {images[currentIndex].capture_date}</p>
          <p>
            W/H {images[currentIndex].width} / {images[currentIndex].height}
          </p>
          <p>Name {images[currentIndex].filename}</p>
          <p>MimeType {images[currentIndex].mime_type}</p>
        </div>
      </div>
    </div>
  );
}
