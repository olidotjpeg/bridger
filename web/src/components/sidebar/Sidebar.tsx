import './Sidebar.css'
import type { ScanStatus } from '../../api/images'

interface SidebarProps {
  sort: string
  order: string
  minRating: number | undefined
  onSortChange: (sort: string) => void
  onOrderChange: (order: string) => void
  onRatingChange: (rating: number | undefined) => void
  scanStatus?: ScanStatus
  onTriggerScan: () => void
}

const RATING_OPTIONS: { label: string; value: number | undefined }[] = [
  { label: 'All', value: undefined },
  { label: '1+', value: 1 },
  { label: '2+', value: 2 },
  { label: '3+', value: 3 },
  { label: '4+', value: 4 },
  { label: '5', value: 5 },
]

export default function Sidebar({ sort, order, minRating, onSortChange, onOrderChange, onRatingChange, scanStatus, onTriggerScan }: SidebarProps) {
  const isRunning = scanStatus?.running ?? false
  const progress = isRunning && scanStatus && scanStatus.total > 0
    ? Math.round((scanStatus.processed / scanStatus.total) * 100)
    : null

  return (
    <div className="sidebar">
      <div className="sidebar-logo">Bridger</div>

      <div className="sidebar-section">
        <span className="sidebar-label">Sort by</span>
        <select className="sidebar-select" value={sort} onChange={e => onSortChange(e.target.value)}>
          <option value="capture_date">Capture date</option>
          <option value="filename">Filename</option>
          <option value="rating">Rating</option>
        </select>
      </div>

      <div className="sidebar-section">
        <span className="sidebar-label">Order</span>
        <div className="sidebar-toggle">
          <button className={order === 'asc' ? 'active' : ''} onClick={() => onOrderChange('asc')}>↑ Asc</button>
          <button className={order === 'desc' ? 'active' : ''} onClick={() => onOrderChange('desc')}>↓ Desc</button>
        </div>
      </div>

      <div className="sidebar-section">
        <span className="sidebar-label">Min rating</span>
        <div className="sidebar-rating">
          {RATING_OPTIONS.map(opt => (
            <button
              key={String(opt.value)}
              className={minRating === opt.value ? 'active' : ''}
              onClick={() => onRatingChange(opt.value)}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      <div className="sidebar-section sidebar-scan">
        <span className="sidebar-label">Library</span>
        <button
          className={`scan-button ${isRunning ? 'scanning' : ''}`}
          onClick={onTriggerScan}
          disabled={isRunning}
        >
          {isRunning ? 'Scanning…' : 'Scan now'}
        </button>

        {isRunning && scanStatus && (
          <div className="scan-progress">
            <div className="scan-progress-bar">
              <div
                className="scan-progress-fill"
                style={{ width: `${progress ?? 0}%` }}
              />
            </div>
            <span className="scan-progress-text">
              {scanStatus.processed.toLocaleString()} / {scanStatus.total.toLocaleString()}
              {scanStatus.errors > 0 && ` · ${scanStatus.errors} errors`}
            </span>
          </div>
        )}
      </div>
    </div>
  )
}
