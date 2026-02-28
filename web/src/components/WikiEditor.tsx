import { useEffect, useState, useCallback, useRef } from 'react'
import { WikiPage } from '../lib/api'
import ImagePickerModal from './ImagePickerModal'
import * as Y from 'yjs'
import { WebsocketProvider } from 'y-websocket'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

interface WikiEditorProps {
  page: WikiPage
}

export default function WikiEditor({ page }: WikiEditorProps) {
  const [content, setContent] = useState('')
  const [isPreview, setIsPreview] = useState(false)
  const [syncState, setSyncState] = useState<'connecting' | 'connected' | 'disconnected'>('connecting')
  const [lastSaved, setLastSaved] = useState<Date | null>(null)
  const [showImagePicker, setShowImagePicker] = useState(false)

  const ydocRef = useRef<Y.Doc | null>(null)
  const providerRef = useRef<WebsocketProvider | null>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    // Initialize Yjs document
    const ydoc = new Y.Doc()
    ydocRef.current = ydoc

    const ytext = ydoc.getText('content')

    // Get auth token from localStorage
    const token = localStorage.getItem('token')
    if (!token) {
      console.error('No auth token found')
      setSyncState('disconnected')
      return
    }

    // Determine WebSocket URL
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsHost = window.location.host
    const wsUrl = `${wsProtocol}//${wsHost}/api/wiki/collab`

    // Initialize WebSocket provider
    const provider = new WebsocketProvider(
      wsUrl,
      `wiki-page-${page.id}`,
      ydoc,
      {
        params: { token },
        connect: true,
      }
    )
    providerRef.current = provider

    // Handle connection state
    provider.on('status', ({ status }: { status: string }) => {
      if (status === 'connected') {
        setSyncState('connected')
        setLastSaved(new Date())
      } else if (status === 'disconnected') {
        setSyncState('disconnected')
      } else {
        setSyncState('connecting')
      }
    })

    // Sync Yjs content to React state
    const updateContent = () => {
      setContent(ytext.toString())
    }

    ytext.observe(updateContent)
    updateContent() // Initial content

    // Cleanup on unmount
    return () => {
      ytext.unobserve(updateContent)
      provider.destroy()
      ydoc.destroy()
    }
  }, [page.id])

  const handleContentChange = useCallback((newContent: string) => {
    if (!ydocRef.current) return

    const ytext = ydocRef.current.getText('content')
    const currentContent = ytext.toString()

    // Calculate diff and apply changes
    if (newContent !== currentContent) {
      ydocRef.current.transact(() => {
        ytext.delete(0, currentContent.length)
        ytext.insert(0, newContent)
      })
    }
  }, [])

  const handleTextareaChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newContent = e.target.value
    setContent(newContent)
    handleContentChange(newContent)
  }

  // Handle tab key in textarea
  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Tab') {
      e.preventDefault()
      const textarea = e.currentTarget
      const start = textarea.selectionStart
      const end = textarea.selectionEnd
      const newContent = content.substring(0, start) + '  ' + content.substring(end)
      setContent(newContent)
      handleContentChange(newContent)

      // Move cursor after the inserted spaces
      setTimeout(() => {
        textarea.selectionStart = textarea.selectionEnd = start + 2
      }, 0)
    }
  }

  const insertImageMarkdown = (alt: string, url: string) => {
    const textarea = textareaRef.current
    const markdown = `![${alt}](${url})`

    if (textarea) {
      const start = textarea.selectionStart
      const end = textarea.selectionEnd
      const newContent = content.substring(0, start) + markdown + content.substring(end)
      setContent(newContent)
      handleContentChange(newContent)

      // Move cursor after inserted markdown
      setTimeout(() => {
        textarea.selectionStart = textarea.selectionEnd = start + markdown.length
        textarea.focus()
      }, 0)
    } else {
      // Fallback: append to end
      const newContent = content + (content.endsWith('\n') ? '' : '\n') + markdown + '\n'
      setContent(newContent)
      handleContentChange(newContent)
    }

    setShowImagePicker(false)
  }

  const getSyncStatusColor = () => {
    switch (syncState) {
      case 'connected':
        return 'bg-green-500'
      case 'connecting':
        return 'bg-yellow-500'
      case 'disconnected':
        return 'bg-red-500'
    }
  }

  const getSyncStatusText = () => {
    switch (syncState) {
      case 'connected':
        return 'Connected'
      case 'connecting':
        return 'Connecting...'
      case 'disconnected':
        return 'Disconnected'
    }
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="border-b border-dark-border-subtle bg-dark-bg-secondary px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-dark-text-primary">{page.title}</h1>
            <div className="flex items-center gap-3 mt-2">
              <div className="flex items-center gap-2">
                <div className={`w-2 h-2 rounded-full ${getSyncStatusColor()}`} />
                <span className="text-sm text-dark-text-tertiary">{getSyncStatusText()}</span>
              </div>
              {lastSaved && (
                <>
                  <span className="text-dark-text-tertiary">•</span>
                  <span className="text-sm text-dark-text-tertiary">
                    Last saved {lastSaved.toLocaleTimeString()}
                  </span>
                </>
              )}
            </div>
          </div>

          <div className="flex items-center gap-2">
            {!isPreview && (
              <button
                onClick={() => setShowImagePicker(true)}
                className="px-3 py-2 rounded text-sm font-medium transition-colors bg-dark-bg-tertiary text-dark-text-secondary hover:bg-dark-bg-tertiary/80 flex items-center gap-1.5"
                title="Insert image"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
                Image
              </button>
            )}
            <button
              onClick={() => setIsPreview(false)}
              className={`px-4 py-2 rounded text-sm font-medium transition-colors ${
                !isPreview
                  ? 'bg-dark-accent-primary text-white'
                  : 'bg-dark-bg-tertiary text-dark-text-secondary hover:bg-dark-bg-tertiary/80'
              }`}
            >
              Edit
            </button>
            <button
              onClick={() => setIsPreview(true)}
              className={`px-4 py-2 rounded text-sm font-medium transition-colors ${
                isPreview
                  ? 'bg-dark-accent-primary text-white'
                  : 'bg-dark-bg-tertiary text-dark-text-secondary hover:bg-dark-bg-tertiary/80'
              }`}
            >
              Preview
            </button>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-hidden">
        {isPreview ? (
          <div className="h-full overflow-y-auto px-6 py-4">
            {content.trim() ? (
              <div className="prose prose-invert max-w-none">
                <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
              </div>
            ) : (
              <div className="flex items-center justify-center h-full text-dark-text-tertiary">
                <div className="text-center">
                  <svg className="w-16 h-16 mx-auto mb-4 text-dark-text-tertiary/50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                  </svg>
                  <p className="text-lg">No content to preview</p>
                  <p className="text-sm mt-2">Switch to Edit mode to start writing</p>
                </div>
              </div>
            )}
          </div>
        ) : (
          <textarea
            ref={textareaRef}
            value={content}
            onChange={handleTextareaChange}
            onKeyDown={handleKeyDown}
            placeholder="Start writing in Markdown...

# Heading 1
## Heading 2

**bold** *italic* `code`

- List item
- List item

[Link text](https://example.com)

```code block```"
            className="w-full h-full px-6 py-4 bg-dark-bg-primary text-dark-text-primary resize-none focus:outline-none font-mono text-sm placeholder-dark-text-tertiary/50"
            spellCheck={false}
          />
        )}
      </div>

      {/* Footer helper */}
      {!isPreview && (
        <div className="border-t border-dark-border-subtle bg-dark-bg-secondary px-6 py-2">
          <div className="flex items-center gap-4 text-xs text-dark-text-tertiary">
            <span>Markdown supported</span>
            <span>•</span>
            <span>Changes sync automatically</span>
            <span>•</span>
            <span>Tab to indent</span>
          </div>
        </div>
      )}

      {/* Image Picker Modal */}
      {showImagePicker && (
        <ImagePickerModal
          onSelect={insertImageMarkdown}
          onClose={() => setShowImagePicker(false)}
          wikiPageId={page.id}
          onUploadComplete={() => {}}
        />
      )}
    </div>
  )
}
