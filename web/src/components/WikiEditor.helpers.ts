// ── Pure helper functions extracted from WikiEditor.tsx ──────────────

// ── Types ────────────────────────────────────────────────────────────

export type ImageInfo = { html: string; url: string; alt: string; caption: string; index: number }
export type DrawInfo = { shortcode: string; id: string; size: string; zoom: string; index: number }
export type SyncState = 'connecting' | 'connected' | 'disconnected'
export type SaveStatus = 'idle' | 'saving' | 'saved' | 'error'

// ── HTML entity helpers ──────────────────────────────────────────────

export function escapeHtml(s: string): string {
  return s.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;')
}

export function unescapeHtml(s: string): string {
  return s.replaceAll('&amp;', '&').replaceAll('&lt;', '<').replaceAll('&gt;', '>').replaceAll('&quot;', '"')
}

// ── Image markup builder ─────────────────────────────────────────────

export function buildImageMarkup(url: string, alt: string, caption: string, size: string): string {
  const sizeStyles: Record<string, string> = {
    s: 'max-width:50%;height:auto;',
    m: 'max-width:75%;height:auto;',
    l: 'width:100%;height:auto;max-width:100%;',
  }
  const imgStyle = sizeStyles[size] || sizeStyles.m

  if (size === 'l' && !caption) {
    return `![${alt}](${url})`
  }
  const captionHtml = caption ? '<figcaption>' + escapeHtml(caption) + '</figcaption>' : ''
  return `<figure style="text-align:center;margin:1.5rem 0"><a href="${url}" data-lightbox="article-images" data-title="${escapeHtml(alt)}"><img src="${url}" alt="${escapeHtml(alt)}" style="${imgStyle}"/></a>${captionHtml}</figure>`
}

// ── Image detection helpers ──────────────────────────────────────────

export function findImagesInContent(content: string): ImageInfo[] {
  const images: ImageInfo[] = []

  const figureRegex = /<figure[^>]*>[\s\S]*?<\/figure>/g
  const imgSrcRegex = /src="([^"]+)"/
  const imgAltRegex = /alt="([^"]*)"/
  const captionRegex = /<figcaption>([\s\S]*?)<\/figcaption>/
  let match
  while ((match = figureRegex.exec(content)) !== null) {
    const figHtml = match[0]
    const srcMatch = imgSrcRegex.exec(figHtml)
    if (!srcMatch) continue
    const altMatch = imgAltRegex.exec(figHtml)
    const capMatch = captionRegex.exec(figHtml)
    images.push({
      html: figHtml,
      url: srcMatch[1],
      alt: unescapeHtml(altMatch?.[1] || ''),
      caption: unescapeHtml(capMatch?.[1] || ''),
      index: match.index,
    })
  }

  const mdRegex = /!\[([^\]]*)\]\(([^)]+)\)/g
  while ((match = mdRegex.exec(content)) !== null) {
    const pos = match.index
    const insideFigure = images.some(img => pos >= img.index && pos < img.index + img.html.length)
    if (insideFigure) continue
    images.push({ html: match[0], url: match[2], alt: match[1], caption: '', index: match.index })
  }

  images.sort((a, b) => a.index - b.index)
  return images
}

export function detectImageSize(html: string): string {
  if (!html.startsWith('<figure')) return 'l'
  if (/max-width:\s*50%/.test(html)) return 's'
  if (/max-width:\s*75%/.test(html)) return 'm'
  return 'l'
}

// ── Draw shortcode helpers ───────────────────────────────────────────

export function findDrawsInContent(content: string): DrawInfo[] {
  const draws: DrawInfo[] = []
  const re = /\[draw:([a-zA-Z0-9_-]+)(?::edit)?(?::([sml]))?(?::z([^\]]+))?\]/g
  let match
  while ((match = re.exec(content)) !== null) {
    draws.push({
      shortcode: match[0],
      id: match[1],
      size: match[2] || 'm',
      zoom: match[3] || 'fit',
      index: match.index,
    })
  }
  draws.sort((a, b) => a.index - b.index)
  return draws
}

export function mapYjsStatus(status: string): SyncState {
  if (status === 'connected') return 'connected'
  if (status === 'disconnected') return 'disconnected'
  return 'connecting'
}

export function findDrawShortcodeAtPosition(text: string, pos: number): string | null {
  const re = /\[draw:([a-zA-Z0-9_-]+)(?::edit)?\]/g
  let match
  while ((match = re.exec(text)) !== null) {
    if (pos >= match.index && pos <= match.index + match[0].length) {
      return match[1]
    }
  }
  return null
}

export function shouldSaveContent(isDirty: boolean, current: string, lastSaved: string): boolean {
  return isDirty && current !== lastSaved
}

// ── Status display helpers ───────────────────────────────────────────

export function getSyncStatusColor(syncState: SyncState): string {
  switch (syncState) {
    case 'connected': return 'bg-green-500'
    case 'connecting': return 'bg-yellow-500'
    case 'disconnected': return 'bg-red-500'
  }
}

export function getSaveStatusColor(saveStatus: SaveStatus, syncState: SyncState): string {
  switch (saveStatus) {
    case 'saving': return 'bg-yellow-500'
    case 'saved': return 'bg-green-500'
    case 'error': return 'bg-red-500'
    default: return getSyncStatusColor(syncState)
  }
}

export function getSaveStatusText(saveStatus: SaveStatus, autoSaveEnabled: boolean): string {
  switch (saveStatus) {
    case 'saving': return 'Saving...'
    case 'saved': return 'Saved'
    case 'error': return 'Save failed'
    default: return autoSaveEnabled ? 'Autosave on' : 'Autosave off'
  }
}

export function getSaveStatusTextColor(saveStatus: SaveStatus): string {
  switch (saveStatus) {
    case 'saving': return 'text-yellow-400'
    case 'saved': return 'text-green-400'
    case 'error': return 'text-red-400'
    default: return 'text-dark-text-tertiary'
  }
}

// ── Draw shortcode builder ───────────────────────────────────────────

export function buildDrawShortcode(id: string, size: string, zoom: string): string {
  const sizeTag = size === 'm' ? '' : ':' + size
  const zoomTag = zoom === 'fit' ? '' : ':z' + zoom
  return `[draw:${id}:edit${sizeTag}${zoomTag}]`
}

// ── Cursor/markup helpers ────────────────────────────────────────────

export function insertMarkupAtCursor(
  textarea: HTMLTextAreaElement | null,
  content: string,
  markup: string,
): { newContent: string; focusPos: number | null } {
  if (textarea) {
    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    return { newContent: content.substring(0, start) + markup + content.substring(end), focusPos: start + markup.length }
  }
  const sep = content.endsWith('\n') ? '' : '\n'
  return { newContent: content + sep + markup + '\n', focusPos: null }
}

// ── Save status helpers ──────────────────────────────────────────────

export function clearSavedStatus(prev: SaveStatus): SaveStatus {
  return prev === 'saved' ? 'idle' : prev
}
