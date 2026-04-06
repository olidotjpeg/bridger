import { useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import type { Image, Tag } from "../../api/images";
import { fetchImageTags, patchImage, createTag } from "../../api/images";
import StarRating from "../stars/StarRating";
import TagEditor from "../tags/TagEditor";
import "./Lightbox.css";

interface LightboxProps {
  images: Image[];
  selectedId: number | null;
  onClose: () => void;
  onNavigate: (id: number) => void;
}

export default function LightBox({ images, selectedId, onClose, onNavigate }: LightboxProps) {
  const queryClient = useQueryClient()
  const currentIndex = images.findIndex((img) => img.id === selectedId);
  const currentImage = images[currentIndex];
  const prevImage = images[currentIndex - 1];
  const nextImage = images[currentIndex + 1];

  const { data: imageTags = [] } = useQuery({
    queryKey: ['imageTags', selectedId],
    queryFn: () => fetchImageTags(selectedId!),
    enabled: selectedId !== null,
  })

  const ratingMutation = useMutation({
    mutationFn: (rating: number) => patchImage(selectedId!, { rating }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['images'] }),
  })

  const tagsMutation = useMutation({
    mutationFn: (tags: number[]) => patchImage(selectedId!, { tags }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['imageTags', selectedId] })
      queryClient.invalidateQueries({ queryKey: ['images'] })
    },
  })

  const createTagMutation = useMutation({
    mutationFn: (name: string) => createTag(name),
    onSuccess: (newTag) => {
      queryClient.invalidateQueries({ queryKey: ['tags'] })
      const updatedIds = [...imageTags.map(t => t.id), newTag.id]
      tagsMutation.mutate(updatedIds)
    },
  })

  function handleAddTag(tag: Tag) {
    tagsMutation.mutate([...imageTags.map(t => t.id), tag.id])
  }

  function handleRemoveTag(tagId: number) {
    tagsMutation.mutate(imageTags.filter(t => t.id !== tagId).map(t => t.id))
  }

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      // Don't fire shortcuts when typing in an input
      if (e.target instanceof HTMLInputElement) return

      if (e.key === "Escape") onClose();
      if (e.key === "ArrowRight" && nextImage) onNavigate(nextImage.id);
      if (e.key === "ArrowLeft" && prevImage) onNavigate(prevImage.id);

      const n = parseInt(e.key)
      if (!isNaN(n) && n >= 0 && n <= 5) {
        ratingMutation.mutate(n)
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [onClose, onNavigate, prevImage, nextImage, selectedId]);

  if (!currentImage) return null;

  const captureDate = currentImage.capture_date
    ? new Date(currentImage.capture_date).toLocaleDateString('en-GB', { day: 'numeric', month: 'short', year: 'numeric' })
    : null;

  const mutating = ratingMutation.isPending || tagsMutation.isPending

  return (
    <div className="lightbox" onClick={onClose}>
      <button className="lightbox-close" onClick={onClose}>×</button>

      {prevImage && (
        <button
          className="lightbox-nav lightbox-nav-prev"
          onClick={e => { e.stopPropagation(); onNavigate(prevImage.id); }}
        >
          ‹
        </button>
      )}

      <div className="lightbox-content" onClick={e => e.stopPropagation()}>
        <img src={`/api/images/${selectedId}/full`} alt={currentImage.filename} />

        <div className="lightbox-meta">
          <span className="lightbox-filename">{currentImage.filename}</span>
          {captureDate && <span className="lightbox-detail">{captureDate}</span>}
          <span className="lightbox-detail">{currentImage.width} × {currentImage.height}</span>
          <span className="lightbox-detail">{currentImage.mime_type}</span>
          <span className="lightbox-counter">{currentIndex + 1} / {images.length}</span>
        </div>

        <div className="lightbox-actions">
          <StarRating
            value={currentImage.rating}
            onChange={rating => ratingMutation.mutate(rating)}
            disabled={mutating}
          />
          <TagEditor
            tags={imageTags}
            onAdd={handleAddTag}
            onRemove={handleRemoveTag}
            onCreateAndAdd={name => createTagMutation.mutate(name)}
            disabled={mutating}
          />
        </div>
      </div>

      {nextImage && (
        <button
          className="lightbox-nav lightbox-nav-next"
          onClick={e => { e.stopPropagation(); onNavigate(nextImage.id); }}
        >
          ›
        </button>
      )}
    </div>
  );
}
