import { useState } from 'react'
import './StarRating.css'

interface StarRatingProps {
  value: number
  onChange: (rating: number) => void
  disabled?: boolean
}

export default function StarRating({ value, onChange, disabled }: StarRatingProps) {
  const [hovered, setHovered] = useState(0)
  const display = hovered || value

  return (
    <div className="star-rating" onMouseLeave={() => setHovered(0)}>
      {[1, 2, 3, 4, 5].map(n => (
        <button
          key={n}
          className={`star ${n <= display ? 'filled' : ''}`}
          onMouseEnter={() => setHovered(n)}
          onClick={() => onChange(value === n ? 0 : n)}
          disabled={disabled}
          aria-label={`${n} star${n > 1 ? 's' : ''}`}
        >
          ★
        </button>
      ))}
    </div>
  )
}
