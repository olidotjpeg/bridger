import { useState, useEffect, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { fetchDates } from '../../api/images'
import './CullPicker.css'

interface CullPickerProps {
  totalImages: number
  onStart: (dateFrom?: string, dateTo?: string) => void
  onCancel: () => void
}

const MONTH_NAMES = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
]
const DAY_LABELS = ['Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa', 'Su']

function toDateStr(d: Date): string {
  return d.toISOString().slice(0, 10)
}

function getCalendarDays(year: number, month: number): (Date | null)[] {
  const first = new Date(year, month, 1)
  const last = new Date(year, month + 1, 0)
  const offset = (first.getDay() + 6) % 7 // Monday = 0
  const days: (Date | null)[] = []
  for (let i = 0; i < offset; i++) days.push(null)
  for (let d = 1; d <= last.getDate(); d++) days.push(new Date(year, month, d))
  while (days.length % 7 !== 0) days.push(null)
  return days
}

export default function CullPicker({ totalImages, onStart, onCancel }: CullPickerProps) {
  const { data: dates = [] } = useQuery({ queryKey: ['dates'], queryFn: fetchDates })

  const dateSet = useMemo(() => new Set(dates.map(d => d.date)), [dates])

  const [viewYear, setViewYear] = useState(new Date().getFullYear())
  const [viewMonth, setViewMonth] = useState(new Date().getMonth())
  const [seeded, setSeeded] = useState(false)

  // Jump to most recent month with photos once data loads
  useEffect(() => {
    if (!seeded && dates.length > 0) {
      setViewYear(parseInt(dates[0].date.slice(0, 4)))
      setViewMonth(parseInt(dates[0].date.slice(5, 7)) - 1)
      setSeeded(true)
    }
  }, [dates, seeded])

  const [anchor, setAnchor] = useState<string | null>(null)
  const [tip, setTip] = useState<string | null>(null)
  const [hoverDate, setHoverDate] = useState<string | null>(null)

  // Canonical [from, to] from the two clicked dates
  const from = anchor && tip ? (anchor <= tip ? anchor : tip) : anchor
  const to = anchor && tip ? (anchor >= tip ? anchor : tip) : anchor

  // Preview range while hovering before second click
  const previewFrom = anchor && !tip && hoverDate
    ? (anchor <= hoverDate ? anchor : hoverDate) : null
  const previewTo = anchor && !tip && hoverDate
    ? (anchor >= hoverDate ? anchor : hoverDate) : null

  const selectedCount = useMemo(() => {
    if (!from) return 0
    const end = to ?? from
    return dates.filter(d => d.date >= from && d.date <= end).reduce((s, d) => s + d.count, 0)
  }, [dates, from, to])

  function handleDayClick(dateStr: string) {
    if (!anchor || (anchor && tip)) {
      setAnchor(dateStr)
      setTip(null)
    } else {
      setTip(dateStr)
    }
  }

  function prevMonth() {
    if (viewMonth === 0) { setViewYear(y => y - 1); setViewMonth(11) }
    else setViewMonth(m => m - 1)
  }
  function nextMonth() {
    if (viewMonth === 11) { setViewYear(y => y + 1); setViewMonth(0) }
    else setViewMonth(m => m + 1)
  }

  function dayClass(dateStr: string): string {
    const cls = ['cal-day']
    if (dateSet.has(dateStr)) cls.push('has-photos')

    if (from && to) {
      if (dateStr === from || dateStr === to) cls.push('sel-edge')
      else if (dateStr > from && dateStr < to) cls.push('in-range')
    } else if (from && dateStr === from) {
      cls.push('sel-edge')
    }

    if (previewFrom && previewTo) {
      if (dateStr === previewFrom || dateStr === previewTo) cls.push('preview-edge')
      else if (dateStr > previewFrom && dateStr < previewTo) cls.push('preview-range')
    }

    return cls.join(' ')
  }

  function formatRange(): string {
    if (!from) return ''
    const fmt = (s: string) =>
      new Date(s + 'T00:00:00').toLocaleDateString('en-GB', { day: 'numeric', month: 'short', year: 'numeric' })
    if (!to || to === from) return fmt(from)
    return `${fmt(from)} – ${fmt(to)}`
  }

  const calDays = getCalendarDays(viewYear, viewMonth)

  return (
    <div className="cull-picker-overlay" onClick={onCancel}>
      <div className="cull-picker" onClick={e => e.stopPropagation()}>
        <div className="cull-picker-header">
          <h2>Select a shoot</h2>
          <p>Pick a date or range, or cull the entire library.</p>
        </div>

        <div className="cal">
          <div className="cal-nav">
            <button className="cal-nav-btn" onClick={prevMonth}>‹</button>
            <span className="cal-month-label">{MONTH_NAMES[viewMonth]} {viewYear}</span>
            <button className="cal-nav-btn" onClick={nextMonth}>›</button>
          </div>

          <div className="cal-grid">
            {DAY_LABELS.map(d => <div key={d} className="cal-dow">{d}</div>)}
            {calDays.map((day, i) => {
              if (!day) return <div key={i} className="cal-day empty" />
              const ds = toDateStr(day)
              return (
                <div
                  key={ds}
                  className={dayClass(ds)}
                  onClick={() => handleDayClick(ds)}
                  onMouseEnter={() => setHoverDate(ds)}
                  onMouseLeave={() => setHoverDate(null)}
                >
                  <span className="cal-day-num">{day.getDate()}</span>
                  {dateSet.has(ds) && <span className="cal-dot" />}
                </div>
              )
            })}
          </div>
        </div>

        {from && (
          <div className="cull-picker-selection">
            <span className="cull-sel-range">{formatRange()}</span>
            <span className="cull-sel-count">{selectedCount.toLocaleString()} photos</span>
          </div>
        )}

        <div className="cull-picker-actions">
          <button className="cull-btn-library" onClick={() => onStart(undefined, undefined)}>
            Entire library · {totalImages.toLocaleString()} photos
          </button>
          <button
            className="cull-btn-start"
            disabled={!from}
            onClick={() => from && onStart(from, to ?? from)}
          >
            Start culling →
          </button>
        </div>
      </div>
    </div>
  )
}
