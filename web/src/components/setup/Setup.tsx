import { useState, useEffect } from 'react'
import { useQuery, useMutation } from '@tanstack/react-query'
import { fetchScanStatus } from '../../api/images'
import { saveConfig, listDirectory } from '../../api/config'
import type { AppConfig, DirListing } from '../../api/config'
import './Setup.css'

type Step = 'welcome' | 'select' | 'confirm' | 'scanning'

interface SetupProps {
  config: AppConfig
  onComplete: () => void
}

export default function Setup({ config, onComplete }: SetupProps) {
  const [step, setStep] = useState<Step>('welcome')
  const [folders, setFolders] = useState<string[]>([])
  const [browserOpen, setBrowserOpen] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [noImagesWarning, setNoImagesWarning] = useState(false)

  const saveMutation = useMutation({
    mutationFn: () => saveConfig(folders),
    onSuccess: () => setStep('scanning'),
    onError: (e: Error) => setSaveError(e.message),
  })

  const { data: scanStatus } = useQuery({
    queryKey: ['scan-status'],
    queryFn: fetchScanStatus,
    enabled: step === 'scanning',
    refetchInterval: step === 'scanning' ? 2000 : false,
  })

  useEffect(() => {
    if (step !== 'scanning' || !scanStatus) return
    if (!scanStatus.running) {
      if (scanStatus.total === 0) {
        setNoImagesWarning(true)
      } else {
        onComplete()
      }
    }
  }, [scanStatus, step, onComplete])

  function addFolder(path: string) {
    setFolders(prev => prev.includes(path) ? prev : [...prev, path])
  }

  function removeFolder(path: string) {
    setFolders(prev => prev.filter(f => f !== path))
  }

  const progress = scanStatus && scanStatus.total > 0
    ? Math.round((scanStatus.processed / scanStatus.total) * 100)
    : null

  return (
    <div className="setup">
      <div className="setup-card">
        <div className="setup-steps">
          {(['welcome', 'select', 'confirm'] as Step[]).map((s, i) => (
            <div
              key={s}
              className={`setup-step-dot ${step === s || (step === 'scanning' && s === 'confirm') ? 'active' : ''} ${
                ['welcome', 'select', 'confirm'].indexOf(step) > i ? 'done' : ''
              }`}
            />
          ))}
        </div>

        {step === 'welcome' && (
          <div className="setup-body">
            <h1 className="setup-title">Bridger</h1>
            <p className="setup-subtitle">
              A fast, local photo culling tool.<br />
              Let's find your photos to get started.
            </p>
            <button className="setup-btn-primary" onClick={() => setStep('select')}>
              Get started →
            </button>
          </div>
        )}

        {step === 'select' && (
          <div className="setup-body">
            <h2 className="setup-heading">Select photo folders</h2>
            <p className="setup-hint">Add one or more folders containing your images.</p>

            <div className="setup-folder-list">
              {folders.length === 0 && (
                <p className="setup-empty">No folders added yet.</p>
              )}
              {folders.map(f => (
                <div key={f} className="setup-folder-row">
                  <span className="setup-folder-path">{f}</span>
                  <button className="setup-folder-remove" onClick={() => removeFolder(f)}>×</button>
                </div>
              ))}
            </div>

            <button className="setup-btn-secondary" onClick={() => setBrowserOpen(true)}>
              + Add folder
            </button>

            <div className="setup-nav">
              <button className="setup-btn-ghost" onClick={() => setStep('welcome')}>← Back</button>
              <button
                className="setup-btn-primary"
                disabled={folders.length === 0}
                onClick={() => setStep('confirm')}
              >
                Next →
              </button>
            </div>
          </div>
        )}

        {step === 'confirm' && (
          <div className="setup-body">
            <h2 className="setup-heading">Ready to scan</h2>

            <div className="setup-confirm-section">
              <span className="setup-confirm-label">Folders</span>
              {folders.map(f => (
                <div key={f} className="setup-folder-row setup-folder-row--readonly">
                  <span className="setup-folder-path">{f}</span>
                </div>
              ))}
            </div>

            <div className="setup-confirm-section">
              <span className="setup-confirm-label">Database</span>
              <span className="setup-confirm-value">{config.db_path}</span>
            </div>

            <div className="setup-confirm-section">
              <span className="setup-confirm-label">Thumbnails</span>
              <span className="setup-confirm-value">{config.thumbs_path}</span>
            </div>

            {saveError && <p className="setup-error">{saveError}</p>}

            <div className="setup-nav">
              <button className="setup-btn-ghost" onClick={() => setStep('select')}>← Back</button>
              <button
                className="setup-btn-primary"
                disabled={saveMutation.isPending}
                onClick={() => { setSaveError(null); saveMutation.mutate() }}
              >
                {saveMutation.isPending ? 'Saving…' : 'Start scanning'}
              </button>
            </div>
          </div>
        )}

        {step === 'scanning' && (
          <div className="setup-body">
            <h2 className="setup-heading">
              {noImagesWarning ? 'No images found' : 'Scanning…'}
            </h2>

            {!noImagesWarning && (
              <>
                <div className="scan-progress-bar">
                  <div
                    className="scan-progress-fill"
                    style={{ width: `${progress ?? 0}%` }}
                  />
                </div>
                <p className="scan-progress-text">
                  {scanStatus
                    ? `${scanStatus.processed.toLocaleString()} / ${scanStatus.total.toLocaleString()} files`
                    : 'Starting…'}
                </p>
              </>
            )}

            {noImagesWarning && (
              <>
                <p className="setup-hint">
                  No supported images were found in the selected folders.
                </p>
                <button className="setup-btn-primary" onClick={onComplete}>
                  Go to gallery anyway
                </button>
              </>
            )}
          </div>
        )}
      </div>

      {browserOpen && (
        <FileBrowser
          onSelect={path => { addFolder(path); setBrowserOpen(false) }}
          onClose={() => setBrowserOpen(false)}
        />
      )}
    </div>
  )
}

interface FileBrowserProps {
  onSelect: (path: string) => void
  onClose: () => void
}

function FileBrowser({ onSelect, onClose }: FileBrowserProps) {
  const [currentPath, setCurrentPath] = useState<string | undefined>(undefined)

  const { data: listing, isLoading, error } = useQuery<DirListing>({
    queryKey: ['fs-list', currentPath],
    queryFn: () => listDirectory(currentPath),
  })

  const breadcrumbs = listing
    ? listing.path.split('/').filter(Boolean).reduce<{ label: string; path: string }[]>((acc, part) => {
        const prev = acc[acc.length - 1]?.path ?? ''
        return [...acc, { label: part, path: `${prev}/${part}` }]
      }, [])
    : []

  return (
    <div className="fb-overlay" onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div className="fb-modal">
        <div className="fb-header">
          <span className="fb-title">Select a folder</span>
          <button className="fb-close" onClick={onClose}>×</button>
        </div>

        <div className="fb-breadcrumb">
          <button className="fb-crumb" onClick={() => setCurrentPath(undefined)}>/</button>
          {breadcrumbs.map(crumb => (
            <span key={crumb.path} className="fb-crumb-wrap">
              <span className="fb-crumb-sep">/</span>
              <button className="fb-crumb" onClick={() => setCurrentPath(crumb.path)}>
                {crumb.label}
              </button>
            </span>
          ))}
        </div>

        <div className="fb-entries">
          {isLoading && <p className="fb-empty">Loading…</p>}
          {error && <p className="fb-empty fb-error">Could not read directory.</p>}
          {listing && listing.entries.length === 0 && (
            <p className="fb-empty">No subfolders here.</p>
          )}
          {listing?.parent && (
            <button className="fb-entry fb-entry--up" onClick={() => setCurrentPath(listing.parent)}>
              ↑ ..
            </button>
          )}
          {listing?.entries.map(entry => (
            <button
              key={entry.path}
              className="fb-entry"
              onClick={() => setCurrentPath(entry.path)}
            >
              <span className="fb-entry-icon">▸</span>
              {entry.name}
            </button>
          ))}
        </div>

        <div className="fb-footer">
          <span className="fb-current-path">{listing?.path ?? '—'}</span>
          <button
            className="setup-btn-primary"
            disabled={!listing}
            onClick={() => listing && onSelect(listing.path)}
          >
            Select this folder
          </button>
        </div>
      </div>
    </div>
  )
}
