import '@testing-library/jest-dom'

// Polyfill ResizeObserver for Headless UI in jsdom
if (typeof globalThis.ResizeObserver === 'undefined') {
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
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
