import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import {
  Background,
  Controls,
  type Edge,
  MarkerType,
  type Node,
  Position,
  ReactFlow,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { useMemo } from 'react'
import { graphQuery } from '../api/client'
import { KINDS, type Kind, type ReviewState } from '../api/types'
import { entityRoute } from '../components/entity-link'
import { ErrorBox, Loading } from '../components/query-state'
import { useContentLang, usePrefs, useT } from '../lib/prefs'

const STATE_COLOR: Record<ReviewState, string> = {
  draft: '#94a3b8',
  changes_requested: '#ef4444',
  modified: '#8b5cf6',
  outdated: '#f59e0b',
  approved: '#10b981',
}

const COL_W = 260
const ROW_H = 72

export default function GraphPage() {
  const t = useT()
  const { text } = useContentLang()
  const { theme } = usePrefs()
  const navigate = useNavigate()
  const q = useQuery(graphQuery)

  const { nodes, edges } = useMemo(() => {
    const g = q.data
    if (!g) return { nodes: [] as Node[], edges: [] as Edge[] }
    // One column per kind; simple and readable for a few hundred nodes.
    const rows: Record<string, number> = {}
    const nodes: Node[] = g.nodes.map((n) => {
      const col = Math.max(0, KINDS.indexOf(n.kind))
      const row = rows[n.kind] ?? 0
      rows[n.kind] = row + 1
      return {
        id: n.key,
        position: { x: col * COL_W, y: row * ROW_H },
        data: { label: `${t(`kind.${n.kind}`)}: ${text(n.title) || n.id}` },
        sourcePosition: Position.Right,
        targetPosition: Position.Left,
        ariaLabel: `${t(`kind.${n.kind}`)} ${text(n.title) || n.id}, ${t(`state.${n.state}`)}`,
        style: {
          borderColor: STATE_COLOR[n.state] ?? STATE_COLOR.draft,
          borderWidth: 2,
          fontSize: 12,
          width: 200,
        },
      }
    })
    const edges: Edge[] = g.edges.map((e) => ({
      id: `${e.from}->${e.to}`,
      source: e.from,
      target: e.to,
      markerEnd: { type: MarkerType.ArrowClosed },
      style: { strokeOpacity: 0.6 },
    }))
    return { nodes, edges }
  }, [q.data, t, text])

  if (q.isPending) return <Loading />
  if (q.isError) return <ErrorBox error={q.error} retry={() => q.refetch()} />

  return (
    <div className="flex h-[calc(100dvh-3.5rem)] flex-col">
      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 px-4 py-3 sm:px-6">
        <h1 className="text-xl font-semibold">{t('nav.graph')}</h1>
        <p className="text-sm text-slate-500">{t('graph.help')}</p>
      </div>
      <div className="min-h-0 flex-1 border-t border-slate-200 dark:border-slate-800">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          fitView
          colorMode={theme}
          nodesConnectable={false}
          onNodeClick={(_, node) => {
            const [kind, ...rest] = node.id.split('/')
            const id = rest.join('/')
            if (kind && (KINDS as string[]).includes(kind)) {
              void navigate({ to: entityRoute(kind as Kind), params: { id } })
            }
          }}
        >
          <Background />
          <Controls />
        </ReactFlow>
      </div>
    </div>
  )
}
