import '@testing-library/jest-dom'

// Polyfill localStorage/sessionStorage — jsdom@25+ throws SecurityError on opaque
// origins (e.g. about:blank) when storage APIs are accessed.  Provide a simple
// in-memory implementation so tests can call getItem/setItem/clear freely.
class MemoryStorage {
  private _store: Record<string, string> = {}
  get length() { return Object.keys(this._store).length }
  key(index: number) { return Object.keys(this._store)[index] ?? null }
  getItem(key: string) { return Object.prototype.hasOwnProperty.call(this._store, key) ? this._store[key] : null }
  setItem(key: string, value: string) { this._store[key] = String(value) }
  removeItem(key: string) { delete this._store[key] }
  clear() { this._store = {} }
}

Object.defineProperty(globalThis, 'localStorage', {
  value: new MemoryStorage(),
  writable: true,
  configurable: true,
})
Object.defineProperty(globalThis, 'sessionStorage', {
  value: new MemoryStorage(),
  writable: true,
  configurable: true,
})

// Polyfill ResizeObserver for Headless UI in jsdom
globalThis.ResizeObserver ??= class ResizeObserver {
  // No-op stubs required for jsdom environment
  observe() { /* no-op */ }
  unobserve() { /* no-op */ }
  disconnect() { /* no-op */ }
}

// Mock import.meta.env for tests
Object.defineProperty(import.meta, 'env', {
  value: {
    VITE_API_URL: 'http://localhost:8080',
    PROD: false,
    DEV: true,
    MODE: 'test',
  },
})
