/**
 * Graph Link utilities for [[wiki:ID|Name]] and [[task:ID|Name]] syntax.
 *
 * This module provides preprocessing to convert the knowledge graph link
 * syntax into custom markdown URLs that can be styled and navigated.
 *
 * Supported formats:
 *   [[wiki:123]]             → displays as "Wiki #123"
 *   [[wiki:123|My Page]]     → displays as "My Page"
 *   [[task:456]]             → displays as "Task #456"
 *   [[task:456|Fix the Bug]] → displays as "Fix the Bug"
 */

const LINK_RE = /\[\[(wiki|task):(\d+)(?:\|([^\]]*))?\]\]/g

/** Preprocesses content to convert [[wiki:ID]] / [[task:ID]] into inline markdown links. */
export function preprocessGraphLinks(content: string): string {
  return content.replace(LINK_RE, (_match, type, id, label) => {
    const displayLabel = label?.trim() || `${type === 'wiki' ? 'Wiki' : 'Task'} #${id}`
    // Use a custom URL scheme so the ReactMarkdown link component can identify them.
    return `[${displayLabel}](graph-link://${type}/${id})`
  })
}

/** Parses a graph-link:// URL into its components. Returns null for non-graph links. */
export function parseGraphLinkUrl(href: string): { type: 'wiki' | 'task'; id: number } | null {
  if (!href.startsWith('graph-link://')) return null
  const rest = href.slice('graph-link://'.length) // e.g. "wiki/123"
  const slash = rest.indexOf('/')
  if (slash < 0) return null
  const type = rest.slice(0, slash) as 'wiki' | 'task'
  const id = parseInt(rest.slice(slash + 1), 10)
  if ((type !== 'wiki' && type !== 'task') || isNaN(id)) return null
  return { type, id }
}
