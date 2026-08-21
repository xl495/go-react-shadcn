const loaders = new Map<string, Promise<void>>()

export function loadScript(src: string): Promise<void> {
  const hit = loaders.get(src)
  if (hit) return hit
  const p = new Promise<void>((resolve, reject) => {
    if (document.querySelector(`script[src="${src}"]`)) {
      resolve()
      return
    }
    const el = document.createElement("script")
    el.src = src
    el.async = true
    el.onload = () => resolve()
    el.onerror = () => reject(new Error(`failed to load ${src}`))
    document.head.appendChild(el)
  })
  loaders.set(src, p)
  return p
}
