import type { Image } from '../api/images'

export function formatCaptureDate(capture_date: string | null | undefined): string | null {
  if (!capture_date) return null
  return new Date(capture_date).toLocaleDateString('en-GB', {
    day: 'numeric', month: 'short', year: 'numeric',
  })
}

export function buildExifFields(img: Image): { label: string; value: string | number }[] {
  const raw: { label: string; value: string | number | null | undefined }[] = [
    { label: 'Camera', value: img.camera_model },
    { label: 'ISO', value: img.iso },
    { label: 'Aperture', value: img.aperture != null ? `f/${img.aperture}` : null },
    { label: 'Shutter', value: img.shutter_speed },
    { label: 'Focal length', value: img.focal_length != null ? `${img.focal_length} mm` : null },
  ]
  return raw.filter((f): f is { label: string; value: string | number } =>
    f.value != null && f.value !== ''
  )
}

export function buildExifParts(img: Image): string[] {
  return [
    img.camera_model,
    img.iso ? `ISO ${img.iso}` : null,
    img.aperture != null ? `f/${img.aperture}` : null,
    img.shutter_speed,
    img.focal_length != null ? `${img.focal_length}mm` : null,
  ].filter((p): p is string => p != null && p !== '')
}
