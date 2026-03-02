import {
  escapeHtml,
  unescapeHtml,
  buildImageMarkup,
  findImagesInContent,
  detectImageSize,
  findDrawsInContent,
  mapYjsStatus,
  findDrawShortcodeAtPosition,
  shouldSaveContent,
  getSyncStatusColor,
  getSaveStatusColor,
  getSaveStatusText,
  getSaveStatusTextColor,
  buildDrawShortcode,
  insertMarkupAtCursor,
  clearSavedStatus,
} from './WikiEditor.helpers'

describe('WikiEditor helpers', () => {
  // ── escapeHtml / unescapeHtml ─────────────────────────────────

  describe('escapeHtml', () => {
    it('escapes ampersands', () => {
      expect(escapeHtml('a & b')).toBe('a &amp; b')
    })

    it('escapes angle brackets', () => {
      expect(escapeHtml('<div>')).toBe('&lt;div&gt;')
    })

    it('escapes double quotes', () => {
      expect(escapeHtml('"hello"')).toBe('&quot;hello&quot;')
    })

    it('escapes all special chars together', () => {
      expect(escapeHtml('<a href="x">&')).toBe('&lt;a href=&quot;x&quot;&gt;&amp;')
    })

    it('returns plain text unchanged', () => {
      expect(escapeHtml('hello world')).toBe('hello world')
    })
  })

  describe('unescapeHtml', () => {
    it('unescapes ampersands', () => {
      expect(unescapeHtml('a &amp; b')).toBe('a & b')
    })

    it('unescapes angle brackets', () => {
      expect(unescapeHtml('&lt;div&gt;')).toBe('<div>')
    })

    it('unescapes double quotes', () => {
      expect(unescapeHtml('&quot;hello&quot;')).toBe('"hello"')
    })

    it('is the inverse of escapeHtml', () => {
      const original = '<a href="test">&foo'
      expect(unescapeHtml(escapeHtml(original))).toBe(original)
    })
  })

  // ── buildImageMarkup ──────────────────────────────────────────

  describe('buildImageMarkup', () => {
    it('returns markdown for large size with no caption', () => {
      const result = buildImageMarkup('http://img.com/a.png', 'My Image', '', 'l')
      expect(result).toBe('![My Image](http://img.com/a.png)')
    })

    it('returns figure HTML for large size with caption', () => {
      const result = buildImageMarkup('http://img.com/a.png', 'Alt', 'Caption text', 'l')
      expect(result).toContain('<figure')
      expect(result).toContain('<figcaption>Caption text</figcaption>')
      expect(result).toContain('width:100%')
    })

    it('returns figure HTML for medium size', () => {
      const result = buildImageMarkup('http://img.com/a.png', 'Alt', '', 'm')
      expect(result).toContain('<figure')
      expect(result).toContain('max-width:75%')
    })

    it('returns figure HTML for small size', () => {
      const result = buildImageMarkup('http://img.com/a.png', 'Alt', '', 's')
      expect(result).toContain('<figure')
      expect(result).toContain('max-width:50%')
    })

    it('escapes alt text in HTML output', () => {
      const result = buildImageMarkup('http://img.com/a.png', '<script>', '', 'm')
      expect(result).toContain('&lt;script&gt;')
      expect(result).not.toContain('<script>')
    })

    it('escapes caption in HTML output', () => {
      const result = buildImageMarkup('http://img.com/a.png', 'Alt', '<b>bold</b>', 'm')
      expect(result).toContain('&lt;b&gt;bold&lt;/b&gt;')
    })

    it('includes lightbox data attributes', () => {
      const result = buildImageMarkup('http://img.com/a.png', 'Alt', '', 'm')
      expect(result).toContain('data-lightbox="article-images"')
      expect(result).toContain('data-title="Alt"')
    })
  })

  // ── findImagesInContent ───────────────────────────────────────

  describe('findImagesInContent', () => {
    it('finds markdown images', () => {
      const content = 'Hello ![Alt text](http://img.com/pic.png) world'
      const images = findImagesInContent(content)
      expect(images).toHaveLength(1)
      expect(images[0].url).toBe('http://img.com/pic.png')
      expect(images[0].alt).toBe('Alt text')
      expect(images[0].caption).toBe('')
    })

    it('finds figure elements', () => {
      const content = '<figure style="text-align:center"><a href="url"><img src="http://img.com/pic.png" alt="My Alt" style="max-width:75%"/></a><figcaption>Caption</figcaption></figure>'
      const images = findImagesInContent(content)
      expect(images).toHaveLength(1)
      expect(images[0].url).toBe('http://img.com/pic.png')
      expect(images[0].alt).toBe('My Alt')
      expect(images[0].caption).toBe('Caption')
    })

    it('finds multiple images', () => {
      const content = '![First](http://a.com/1.png)\n\n![Second](http://a.com/2.png)'
      const images = findImagesInContent(content)
      expect(images).toHaveLength(2)
      expect(images[0].alt).toBe('First')
      expect(images[1].alt).toBe('Second')
    })

    it('returns empty array for no images', () => {
      expect(findImagesInContent('Just some text')).toHaveLength(0)
    })

    it('does not double-count images inside figures', () => {
      const content = '<figure><a href="url"><img src="http://img.com/pic.png" alt="Alt"/></a></figure>'
      const images = findImagesInContent(content)
      expect(images).toHaveLength(1)
    })

    it('sorts images by position', () => {
      const content = '![B](http://b.png) some text ![A](http://a.png)'
      const images = findImagesInContent(content)
      expect(images[0].alt).toBe('B')
      expect(images[1].alt).toBe('A')
    })
  })

  // ── detectImageSize ───────────────────────────────────────────

  describe('detectImageSize', () => {
    it('returns l for plain markdown', () => {
      expect(detectImageSize('![Alt](url)')).toBe('l')
    })

    it('returns s for 50% width figure', () => {
      expect(detectImageSize('<figure><img style="max-width: 50%; height:auto;"/></figure>')).toBe('s')
    })

    it('returns m for 75% width figure', () => {
      expect(detectImageSize('<figure><img style="max-width: 75%; height:auto;"/></figure>')).toBe('m')
    })

    it('returns l for figure without size styles', () => {
      expect(detectImageSize('<figure><img style="width:100%"/></figure>')).toBe('l')
    })
  })

  // ── findDrawsInContent ────────────────────────────────────────

  describe('findDrawsInContent', () => {
    it('finds basic draw shortcode', () => {
      const draws = findDrawsInContent('Some text [draw:abc123:edit] more text')
      expect(draws).toHaveLength(1)
      expect(draws[0].id).toBe('abc123')
      expect(draws[0].size).toBe('m')
      expect(draws[0].zoom).toBe('fit')
    })

    it('finds draw shortcode with size', () => {
      const draws = findDrawsInContent('[draw:abc:edit:s]')
      expect(draws).toHaveLength(1)
      expect(draws[0].size).toBe('s')
    })

    it('finds draw shortcode with zoom', () => {
      const draws = findDrawsInContent('[draw:abc:edit:m:z150%]')
      expect(draws).toHaveLength(1)
      expect(draws[0].zoom).toBe('150%')
    })

    it('finds multiple draws', () => {
      const content = '[draw:a:edit]\n[draw:b:edit:l]\n[draw:c:edit:s:z200%]'
      const draws = findDrawsInContent(content)
      expect(draws).toHaveLength(3)
      expect(draws[0].id).toBe('a')
      expect(draws[1].id).toBe('b')
      expect(draws[2].id).toBe('c')
    })

    it('returns empty for no draws', () => {
      expect(findDrawsInContent('No draws here')).toHaveLength(0)
    })
  })

  // ── mapYjsStatus ──────────────────────────────────────────────

  describe('mapYjsStatus', () => {
    it('maps connected', () => {
      expect(mapYjsStatus('connected')).toBe('connected')
    })

    it('maps disconnected', () => {
      expect(mapYjsStatus('disconnected')).toBe('disconnected')
    })

    it('maps connecting', () => {
      expect(mapYjsStatus('connecting')).toBe('connecting')
    })

    it('maps unknown to connecting', () => {
      expect(mapYjsStatus('whatever')).toBe('connecting')
    })
  })

  // ── findDrawShortcodeAtPosition ───────────────────────────────

  describe('findDrawShortcodeAtPosition', () => {
    it('returns draw id when cursor is inside shortcode', () => {
      const text = 'Hello [draw:abc123:edit] world'
      expect(findDrawShortcodeAtPosition(text, 10)).toBe('abc123')
    })

    it('returns null when cursor is outside shortcode', () => {
      const text = 'Hello [draw:abc123:edit] world'
      expect(findDrawShortcodeAtPosition(text, 2)).toBeNull()
    })

    it('returns null for no shortcodes', () => {
      expect(findDrawShortcodeAtPosition('no draws', 3)).toBeNull()
    })

    it('handles cursor at start of shortcode', () => {
      const text = '[draw:abc:edit]'
      expect(findDrawShortcodeAtPosition(text, 0)).toBe('abc')
    })

    it('handles cursor at end of shortcode', () => {
      const text = '[draw:abc:edit]'
      expect(findDrawShortcodeAtPosition(text, 15)).toBe('abc')
    })
  })

  // ── shouldSaveContent ─────────────────────────────────────────

  describe('shouldSaveContent', () => {
    it('returns true when dirty and content changed', () => {
      expect(shouldSaveContent(true, 'new', 'old')).toBe(true)
    })

    it('returns false when not dirty', () => {
      expect(shouldSaveContent(false, 'new', 'old')).toBe(false)
    })

    it('returns false when content unchanged', () => {
      expect(shouldSaveContent(true, 'same', 'same')).toBe(false)
    })

    it('returns false when not dirty and content unchanged', () => {
      expect(shouldSaveContent(false, 'same', 'same')).toBe(false)
    })
  })

  // ── Status color/text helpers ─────────────────────────────────

  describe('getSyncStatusColor', () => {
    it('returns green for connected', () => {
      expect(getSyncStatusColor('connected')).toBe('bg-green-500')
    })

    it('returns yellow for connecting', () => {
      expect(getSyncStatusColor('connecting')).toBe('bg-yellow-500')
    })

    it('returns red for disconnected', () => {
      expect(getSyncStatusColor('disconnected')).toBe('bg-red-500')
    })
  })

  describe('getSaveStatusColor', () => {
    it('returns yellow for saving', () => {
      expect(getSaveStatusColor('saving', 'connected')).toBe('bg-yellow-500')
    })

    it('returns green for saved', () => {
      expect(getSaveStatusColor('saved', 'connected')).toBe('bg-green-500')
    })

    it('returns red for error', () => {
      expect(getSaveStatusColor('error', 'connected')).toBe('bg-red-500')
    })

    it('falls back to sync color for idle', () => {
      expect(getSaveStatusColor('idle', 'connected')).toBe('bg-green-500')
      expect(getSaveStatusColor('idle', 'disconnected')).toBe('bg-red-500')
    })
  })

  describe('getSaveStatusText', () => {
    it('returns Saving... for saving', () => {
      expect(getSaveStatusText('saving', true)).toBe('Saving...')
    })

    it('returns Saved for saved', () => {
      expect(getSaveStatusText('saved', true)).toBe('Saved')
    })

    it('returns Save failed for error', () => {
      expect(getSaveStatusText('error', true)).toBe('Save failed')
    })

    it('returns autosave state for idle', () => {
      expect(getSaveStatusText('idle', true)).toBe('Autosave on')
      expect(getSaveStatusText('idle', false)).toBe('Autosave off')
    })
  })

  describe('getSaveStatusTextColor', () => {
    it('returns yellow for saving', () => {
      expect(getSaveStatusTextColor('saving')).toBe('text-yellow-400')
    })

    it('returns green for saved', () => {
      expect(getSaveStatusTextColor('saved')).toBe('text-green-400')
    })

    it('returns red for error', () => {
      expect(getSaveStatusTextColor('error')).toBe('text-red-400')
    })

    it('returns tertiary for idle', () => {
      expect(getSaveStatusTextColor('idle')).toBe('text-dark-text-tertiary')
    })
  })

  // ── buildDrawShortcode ────────────────────────────────────────

  describe('buildDrawShortcode', () => {
    it('builds default shortcode (m size, fit zoom)', () => {
      expect(buildDrawShortcode('abc', 'm', 'fit')).toBe('[draw:abc:edit]')
    })

    it('includes size when not m', () => {
      expect(buildDrawShortcode('abc', 's', 'fit')).toBe('[draw:abc:edit:s]')
      expect(buildDrawShortcode('abc', 'l', 'fit')).toBe('[draw:abc:edit:l]')
    })

    it('includes zoom when not fit', () => {
      expect(buildDrawShortcode('abc', 'm', '150%')).toBe('[draw:abc:edit:z150%]')
    })

    it('includes both size and zoom', () => {
      expect(buildDrawShortcode('abc', 'l', '200%')).toBe('[draw:abc:edit:l:z200%]')
    })
  })

  // ── insertMarkupAtCursor ──────────────────────────────────────

  describe('insertMarkupAtCursor', () => {
    it('inserts at cursor when textarea is provided', () => {
      const textarea = {
        selectionStart: 5,
        selectionEnd: 5,
      } as HTMLTextAreaElement
      const result = insertMarkupAtCursor(textarea, 'Hello world', '**bold**')
      expect(result.newContent).toBe('Hello**bold** world')
      expect(result.focusPos).toBe(13)
    })

    it('replaces selection when range is selected', () => {
      const textarea = {
        selectionStart: 5,
        selectionEnd: 11,
      } as HTMLTextAreaElement
      const result = insertMarkupAtCursor(textarea, 'Hello world!', '**bold**')
      expect(result.newContent).toBe('Hello**bold**!')
      expect(result.focusPos).toBe(13)
    })

    it('appends with newline when no textarea', () => {
      const result = insertMarkupAtCursor(null, 'Hello', 'markup')
      expect(result.newContent).toBe('Hello\nmarkup\n')
      expect(result.focusPos).toBeNull()
    })

    it('skips extra newline when content ends with one', () => {
      const result = insertMarkupAtCursor(null, 'Hello\n', 'markup')
      expect(result.newContent).toBe('Hello\nmarkup\n')
    })
  })

  // ── clearSavedStatus ──────────────────────────────────────────

  describe('clearSavedStatus', () => {
    it('clears saved to idle', () => {
      expect(clearSavedStatus('saved')).toBe('idle')
    })

    it('keeps saving as is', () => {
      expect(clearSavedStatus('saving')).toBe('saving')
    })

    it('keeps error as is', () => {
      expect(clearSavedStatus('error')).toBe('error')
    })

    it('keeps idle as idle', () => {
      expect(clearSavedStatus('idle')).toBe('idle')
    })
  })
})
