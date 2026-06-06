interface WailsRuntime {
  EventsOn(event: string, callback: () => void): () => void
}

interface WailsGo {
  main: {
    App: {
      PickFolder(): Promise<string>
      SaveConfig(dirs: string[]): Promise<void>
    }
  }
}

declare global {
  interface Window {
    runtime?: WailsRuntime
    go?: WailsGo
  }
}

export {}
