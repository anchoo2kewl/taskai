import { useState, useEffect, useRef, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { api, GraphNode, GraphData } from '../lib/api'

// Force simulation constants
const REPULSION = 6000
const SPRING_K = 0.004
const IDEAL_DIST = 160
const CENTER_K = 0.002
const DAMPING = 0.82
const DT = 1.0
const MAX_ITERS = 400

interface NodePos {
  x: number
  y: number
  vx: number
  vy: number
}

export default function KnowledgeGraphPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()

  const [graphData, setGraphData] = useState<GraphData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState<'all' | 'wiki' | 'task'>('all')
  const [searchQuery, setSearchQuery] = useState('')
  const [hoveredNodeId, setHoveredNodeId] = useState<number | null>(null)

  // SVG size measured on mount
  const [svgSize, setSvgSize] = useState({ w: 800, h: 600 })

  // Pan/zoom state
  const [pan, setPan] = useState({ x: 0, y: 0 })
  const [scale, setScale] = useState(1)
  const isPanning = useRef(false)
  const panStart = useRef({ x: 0, y: 0 })

  // Force simulation: positions stored in a ref to avoid re-render thrashing.
  // setRenderTick triggers a re-render each animation frame without re-creating the positions array.
  const posRef = useRef<NodePos[]>([])
  const [, setRenderTick] = useState(0)
  const animFrameRef = useRef<number | null>(null)
  const iterRef = useRef(0)

  const svgRef = useRef<SVGSVGElement>(null)

  // Measure SVG on mount
  useEffect(() => {
    const measure = () => {
      if (svgRef.current) {
        setSvgSize({ w: svgRef.current.clientWidth, h: svgRef.current.clientHeight })
      }
    }
    measure()
    window.addEventListener('resize', measure)
    return () => window.removeEventListener('resize', measure)
  }, [])

  // Load graph data
  useEffect(() => {
    if (!projectId) return
    const load = async () => {
      setLoading(true)
      setError(null)
      try {
        const data = await api.getProjectGraph(parseInt(projectId))
        setGraphData(data)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load graph')
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [projectId])

  // Initialize and run force simulation when graph data changes
  useEffect(() => {
    if (!graphData || graphData.nodes.length === 0) return

    const n = graphData.nodes.length
    // Place nodes in a circle with random jitter
    posRef.current = graphData.nodes.map((_, i) => ({
      x: Math.cos((2 * Math.PI * i) / n) * 200 + (Math.random() - 0.5) * 40,
      y: Math.sin((2 * Math.PI * i) / n) * 200 + (Math.random() - 0.5) * 40,
      vx: 0,
      vy: 0,
    }))
    iterRef.current = 0

    const nodeIndexMap = new Map(graphData.nodes.map((node, i) => [node.id, i]))

    const runFrame = () => {
      if (iterRef.current >= MAX_ITERS) return
      iterRef.current++

      const pos = posRef.current
      const edges = graphData.edges
      const forces: { fx: number; fy: number }[] = pos.map(() => ({ fx: 0, fy: 0 }))

      // Repulsion between all pairs
      for (let i = 0; i < pos.length; i++) {
        for (let j = i + 1; j < pos.length; j++) {
          const dx = pos[i].x - pos[j].x
          const dy = pos[i].y - pos[j].y
          const distSq = dx * dx + dy * dy || 1
          const dist = Math.sqrt(distSq)
          const force = REPULSION / distSq
          const fx = (dx / dist) * force
          const fy = (dy / dist) * force
          forces[i].fx += fx
          forces[i].fy += fy
          forces[j].fx -= fx
          forces[j].fy -= fy
        }
      }

      // Spring attraction along edges
      for (const edge of edges) {
        const si = nodeIndexMap.get(edge.source_node_id)
        const ti = nodeIndexMap.get(edge.target_node_id)
        if (si === undefined || ti === undefined) continue
        const dx = pos[ti].x - pos[si].x
        const dy = pos[ti].y - pos[si].y
        const dist = Math.sqrt(dx * dx + dy * dy) || 1
        const force = SPRING_K * (dist - IDEAL_DIST)
        const fx = (dx / dist) * force
        const fy = (dy / dist) * force
        forces[si].fx += fx
        forces[si].fy += fy
        forces[ti].fx -= fx
        forces[ti].fy -= fy
      }

      // Centering + integrate
      for (let i = 0; i < pos.length; i++) {
        forces[i].fx -= pos[i].x * CENTER_K
        forces[i].fy -= pos[i].y * CENTER_K
        pos[i].vx = (pos[i].vx + forces[i].fx * DT) * DAMPING
        pos[i].vy = (pos[i].vy + forces[i].fy * DT) * DAMPING
        pos[i].x += pos[i].vx
        pos[i].y += pos[i].vy
      }

      setRenderTick(t => t + 1)
      animFrameRef.current = requestAnimationFrame(runFrame)
    }

    if (animFrameRef.current !== null) {
      cancelAnimationFrame(animFrameRef.current)
    }
    animFrameRef.current = requestAnimationFrame(runFrame)

    return () => {
      if (animFrameRef.current !== null) {
        cancelAnimationFrame(animFrameRef.current)
      }
    }
  }, [graphData])

  // Navigate to entity when node is clicked
  const handleNodeClick = useCallback((node: GraphNode) => {
    if (node.entity_type === 'wiki') {
      navigate(`/app/projects/${node.project_id}?tab=wiki&page=${node.entity_id}`)
    } else {
      // task_number is stored in entity_number
      if (node.entity_number != null) {
        navigate(`/app/projects/${node.project_id}/tasks/${node.entity_number}`)
      } else {
        navigate(`/app/projects/${node.project_id}`)
      }
    }
  }, [navigate])

  // Pan/zoom handlers
  const handleWheel = useCallback((e: React.WheelEvent) => {
    e.preventDefault()
    const factor = e.deltaY > 0 ? 0.9 : 1.1
    setScale(s => Math.max(0.1, Math.min(8, s * factor)))
  }, [])

  const handleMouseDown = useCallback((e: React.MouseEvent<SVGSVGElement>) => {
    const tag = (e.target as Element).tagName
    if (tag === 'svg' || tag === 'line') {
      isPanning.current = true
      panStart.current = { x: e.clientX - pan.x, y: e.clientY - pan.y }
      e.preventDefault()
    }
  }, [pan])

  const handleMouseMove = useCallback((e: React.MouseEvent) => {
    if (isPanning.current) {
      setPan({ x: e.clientX - panStart.current.x, y: e.clientY - panStart.current.y })
    }
  }, [])

  const handleMouseUp = useCallback(() => {
    isPanning.current = false
  }, [])

  // Derived display data (filtered)
  const filteredNodes = (graphData?.nodes ?? []).filter(n => {
    if (filter !== 'all' && n.entity_type !== filter) return false
    if (searchQuery && !n.title.toLowerCase().includes(searchQuery.toLowerCase())) return false
    return true
  })
  const filteredNodeIdSet = new Set(filteredNodes.map(n => n.id))
  const filteredEdges = (graphData?.edges ?? []).filter(
    e => filteredNodeIdSet.has(e.source_node_id) && filteredNodeIdSet.has(e.target_node_id)
  )

  // Node index map for position lookup (based on full graphData.nodes)
  const nodeIndexMap = graphData
    ? new Map(graphData.nodes.map((n, i) => [n.id, i]))
    : new Map<number, number>()

  // Connected node IDs for hover highlighting
  const connectedIds = new Set<number>()
  if (hoveredNodeId !== null) {
    for (const e of filteredEdges) {
      if (e.source_node_id === hoveredNodeId) connectedIds.add(e.target_node_id)
      if (e.target_node_id === hoveredNodeId) connectedIds.add(e.source_node_id)
    }
  }

  const centerX = svgSize.w / 2 + pan.x
  const centerY = svgSize.h / 2 + pan.y

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-400" />
      </div>
    )
  }

  if (error) {
    return <div className="p-8 text-center text-red-400">{error}</div>
  }

  return (
    <div className="flex flex-col h-full bg-dark-bg-primary overflow-hidden">
      {/* Top controls bar */}
      <div className="flex items-center gap-3 px-4 py-3 border-b border-dark-border-subtle flex-shrink-0">
        <h2 className="text-sm font-semibold text-dark-text-primary">Knowledge Graph</h2>
        <div className="flex-1" />
        {/* Search */}
        <input
          type="text"
          placeholder="Search nodes..."
          value={searchQuery}
          onChange={e => setSearchQuery(e.target.value)}
          className="bg-dark-bg-secondary border border-dark-border-subtle text-dark-text-primary text-xs rounded-md px-3 py-1.5 w-44 focus:outline-none focus:ring-1 focus:ring-primary-500"
        />
        {/* Filter pills */}
        {(['all', 'wiki', 'task'] as const).map(f => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className={`px-2.5 py-1 text-xs rounded-md font-medium transition-colors ${
              filter === f
                ? 'bg-primary-500 text-white'
                : 'bg-dark-bg-secondary text-dark-text-secondary hover:text-dark-text-primary border border-dark-border-subtle'
            }`}
          >
            {f === 'all' ? 'All' : f === 'wiki' ? 'Wiki' : 'Tasks'}
          </button>
        ))}
        <span className="text-xs text-dark-text-quaternary tabular-nums">
          {filteredNodes.length} nodes · {filteredEdges.length} edges
        </span>
      </div>

      {/* Graph canvas */}
      {filteredNodes.length === 0 ? (
        <div className="flex-1 flex items-center justify-center text-dark-text-tertiary text-sm">
          <div className="text-center space-y-2">
            <div className="text-5xl">🔗</div>
            <p className="font-medium text-dark-text-secondary">No linked content yet</p>
            <p className="text-xs text-dark-text-quaternary max-w-xs">
              Use <code className="bg-dark-bg-secondary px-1 py-0.5 rounded text-primary-400">[[wiki:ID]]</code> or{' '}
              <code className="bg-dark-bg-secondary px-1 py-0.5 rounded text-primary-400">[[task:ID]]</code> in wiki
              pages or task descriptions to create links
            </p>
          </div>
        </div>
      ) : (
        <svg
          ref={svgRef}
          className="flex-1 select-none"
          style={{ cursor: isPanning.current ? 'grabbing' : 'grab' }}
          onWheel={handleWheel}
          onMouseDown={handleMouseDown}
          onMouseMove={handleMouseMove}
          onMouseUp={handleMouseUp}
          onMouseLeave={handleMouseUp}
        >
          <g transform={`translate(${centerX},${centerY}) scale(${scale})`}>
            {/* Edges */}
            {filteredEdges.map(edge => {
              const si = nodeIndexMap.get(edge.source_node_id)
              const ti = nodeIndexMap.get(edge.target_node_id)
              if (si === undefined || ti === undefined) return null
              const sp = posRef.current[si]
              const tp = posRef.current[ti]
              if (!sp || !tp) return null

              const isHighlighted =
                hoveredNodeId === edge.source_node_id || hoveredNodeId === edge.target_node_id
              const isDimmed = hoveredNodeId !== null && !isHighlighted

              return (
                <line
                  key={edge.id}
                  x1={sp.x}
                  y1={sp.y}
                  x2={tp.x}
                  y2={tp.y}
                  stroke={isHighlighted ? '#60a5fa' : '#4b5563'}
                  strokeWidth={isHighlighted ? 2 : 1}
                  opacity={isDimmed ? 0.15 : 0.7}
                />
              )
            })}

            {/* Nodes */}
            {filteredNodes.map(node => {
              const idx = nodeIndexMap.get(node.id)
              if (idx === undefined) return null
              const p = posRef.current[idx]
              if (!p) return null

              const isHovered = hoveredNodeId === node.id
              const isConnected = connectedIds.has(node.id)
              const isDimmed = hoveredNodeId !== null && !isHovered && !isConnected

              const baseColor = node.entity_type === 'wiki' ? '#3b82f6' : '#f97316'
              const brightColor = node.entity_type === 'wiki' ? '#93c5fd' : '#fdba74'
              const radius = isHovered ? 11 : 8
              const displayTitle =
                node.title.length > 22 ? node.title.slice(0, 22) + '…' : node.title

              return (
                <g
                  key={node.id}
                  transform={`translate(${p.x},${p.y})`}
                  style={{ cursor: 'pointer', opacity: isDimmed ? 0.25 : 1 }}
                  onClick={() => handleNodeClick(node)}
                  onMouseEnter={() => setHoveredNodeId(node.id)}
                  onMouseLeave={() => setHoveredNodeId(null)}
                >
                  {/* Glow ring on hover */}
                  {isHovered && (
                    <circle r={radius + 5} fill={baseColor} opacity={0.2} />
                  )}
                  <circle
                    r={radius}
                    fill={isHovered || isConnected ? brightColor : baseColor}
                    stroke={isHovered ? '#ffffff' : isConnected ? '#d1d5db' : 'transparent'}
                    strokeWidth={1.5}
                  />
                  {/* Type badge */}
                  <text
                    textAnchor="middle"
                    dy="4"
                    fontSize="8"
                    fill={isHovered || isConnected ? '#1f2937' : '#ffffff'}
                    fontWeight="600"
                    style={{ pointerEvents: 'none' }}
                  >
                    {node.entity_type === 'wiki' ? 'W' : 'T'}
                  </text>
                  {/* Label */}
                  <text
                    textAnchor="middle"
                    dy={radius + 14}
                    fontSize="11"
                    fill={isDimmed ? '#374151' : isHovered ? '#f9fafb' : '#9ca3af'}
                    style={{ pointerEvents: 'none' }}
                  >
                    {displayTitle}
                  </text>
                </g>
              )
            })}
          </g>
        </svg>
      )}

      {/* Bottom legend */}
      <div className="flex items-center gap-4 px-4 py-2 border-t border-dark-border-subtle flex-shrink-0">
        <div className="flex items-center gap-1.5 text-xs text-dark-text-tertiary">
          <div className="w-3 h-3 rounded-full bg-blue-500 flex-shrink-0" />
          <span>Wiki page</span>
        </div>
        <div className="flex items-center gap-1.5 text-xs text-dark-text-tertiary">
          <div className="w-3 h-3 rounded-full bg-orange-500 flex-shrink-0" />
          <span>Task</span>
        </div>
        <div className="ml-auto text-xs text-dark-text-quaternary">
          Scroll to zoom · Drag to pan · Click to open
        </div>
      </div>
    </div>
  )
}
