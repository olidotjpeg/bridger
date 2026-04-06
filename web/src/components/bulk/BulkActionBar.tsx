import './BulkActionBar.css'

interface BulkActionBarProps {
  count: number
  onSetRating: (rating: number) => void
  onClear: () => void
}

export default function BulkActionBar({ count, onSetRating, onClear }: BulkActionBarProps) {
  return (
    <div className="bulk-bar">
      <div className="bulk-bar-left">
        <span className="bulk-count">{count} selected</span>
        <button className="bulk-clear" onClick={onClear}>Clear</button>
      </div>
      <div className="bulk-bar-right">
        <span className="bulk-label">Rate:</span>
        {[1, 2, 3, 4, 5].map(n => (
          <button key={n} className="bulk-star" onClick={() => onSetRating(n)}>
            {'★'.repeat(n)}
          </button>
        ))}
        <button className="bulk-reject" onClick={() => onSetRating(-1)}>
          Reject
        </button>
      </div>
    </div>
  )
}
