import { useState, useEffect } from 'react'
import { useQuery, useMutation } from '@tanstack/react-query'
import { fetchScanStatus } from '../../api/images'
import { saveConfig } from '../../api/config'
import type { AppConfig } from '../../api/config'
import './Setup.css'

type Step = 'welcome' | 'select' | 'confirm' | 'scanning'

interface SetupProps {
  config: AppConfig
  onComplete: () => void
}

export default function Setup({ config, onComplete }: SetupProps) {
  const [step, setStep] = useState<Step>('welcome')
  const [folders, setFolders] = useState<string[]>([])
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

  async function pickFolder() {
    const path: string = await (window as any).go.main.App.PickFolder()
    if (path) {
      setFolders(prev => prev.includes(path) ? prev : [...prev, path])
    }
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

            <button className="setup-btn-secondary" onClick={pickFolder}>
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
    </div>
  )
}
