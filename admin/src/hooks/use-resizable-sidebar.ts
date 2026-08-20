import { useCallback, useEffect, useRef, useState, type KeyboardEvent, type PointerEvent } from "react"

const MIN = 64
const MAX = 420
const DEFAULT = 224
const COMPACT = 132
const COLLAPSED = 72

function clamp(n: number) {
  return Math.min(MAX, Math.max(MIN, Math.round(n)))
}

function readStored(key: string) {
  try {
    const n = Number(localStorage.getItem(key))
    if (Number.isFinite(n) && n >= MIN && n <= MAX) return n
  } catch {
    /* ignore */
  }
  return DEFAULT
}

export function useResizableSidebar(storageKey: string) {
  const [width, setWidth] = useState(() => readStored(storageKey))
  const [resizing, setResizing] = useState(false)
  const expandedWidthRef = useRef(width >= COMPACT ? width : DEFAULT)
  const compact = width < COMPACT

  useEffect(() => {
    if (width >= COMPACT) expandedWidthRef.current = width
  }, [width])

  useEffect(() => {
    try {
      localStorage.setItem(storageKey, String(width))
    } catch {
      /* ignore */
    }
  }, [storageKey, width])

  const onResizePointerDown = useCallback((event: PointerEvent<HTMLElement>) => {
    event.preventDefault()
    const handle = event.currentTarget
    handle.setPointerCapture(event.pointerId)
    const startX = event.clientX
    const startW = handle.parentElement?.getBoundingClientRect().width ?? width
    setResizing(true)
    document.body.style.cursor = "col-resize"
    document.body.style.userSelect = "none"

    function onMove(e: globalThis.PointerEvent) {
      setWidth(clamp(startW + (e.clientX - startX)))
    }
    function onUp(e: globalThis.PointerEvent) {
      handle.releasePointerCapture(e.pointerId)
      handle.removeEventListener("pointermove", onMove)
      handle.removeEventListener("pointerup", onUp)
      handle.removeEventListener("pointercancel", onUp)
      setResizing(false)
      document.body.style.cursor = ""
      document.body.style.userSelect = ""
    }
    handle.addEventListener("pointermove", onMove)
    handle.addEventListener("pointerup", onUp)
    handle.addEventListener("pointercancel", onUp)
  }, [width])

  const onResizeKeyDown = useCallback((event: KeyboardEvent<HTMLElement>) => {
    if (event.key === "ArrowLeft") {
      event.preventDefault()
      setWidth((w) => clamp(w - 16))
    }
    if (event.key === "ArrowRight") {
      event.preventDefault()
      setWidth((w) => clamp(w + 16))
    }
  }, [])

  const resetWidth = useCallback(() => setWidth(DEFAULT), [])

  const toggleCollapsed = useCallback(() => {
    setWidth((w) => (w < COMPACT ? clamp(expandedWidthRef.current) : COLLAPSED))
  }, [])

  return {
    width,
    compact,
    resizing,
    collapsed: width <= COLLAPSED,
    onResizePointerDown,
    onResizeKeyDown,
    resetWidth,
    toggleCollapsed,
  }
}
