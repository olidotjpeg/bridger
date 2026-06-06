import { useEffect, type RefObject } from 'react'

export function useClickOutside<T extends HTMLElement>(
  ref: RefObject<T | null>,
  callback: () => void,
) {
  useEffect(() => {
    function handleMouseDown(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        callback()
      }
    }
    document.addEventListener('mousedown', handleMouseDown)
    return () => document.removeEventListener('mousedown', handleMouseDown)
  }, [ref, callback])
}
