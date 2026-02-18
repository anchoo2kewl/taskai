// Version information injected at build time
export const version = {
  version: import.meta.env.VITE_VERSION || 'dev',
  gitCommit: import.meta.env.VITE_GIT_COMMIT || 'unknown',
  buildTime: import.meta.env.VITE_BUILD_TIME || 'unknown',
}
