import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { useEntities, useProject } from '../api/client'
import type { Summary } from '../api/types'
import { useContentLang, useT } from '../lib/prefs'
import { useConfigLabels } from './config-labels'
import { EntityLink } from './entity-link'
import { Loc } from './loc-text'
import { StateDot } from './status-badge'

const itemCls =
  'flex items-center gap-2 rounded px-2 py-1 text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800'
const activeCls = 'bg-slate-100 font-medium text-slate-900 dark:bg-slate-800 dark:text-white'

function NavLink({
  to,
  children,
}: {
  to: '/' | '/review' | '/glossary' | '/graph'
  children: ReactNode
}) {
  return (
    <Link
      to={to}
      className={itemCls}
      activeOptions={{ exact: true, includeSearch: false }}
      activeProps={{ className: activeCls, 'aria-current': 'page' }}
    >
      {children}
    </Link>
  )
}

function Item({ e, depth }: { e: Summary; depth: number }) {
  return (
    <li>
      <EntityLink kind={e.kind} id={e.id} className={`${itemCls} ${depth ? 'pl-6' : ''}`}>
        <StateDot state={e.status.state} />
        <span className="truncate">
          <Loc v={e.title} fallback={e.id} />
        </span>
      </EntityLink>
    </li>
  )
}

/** Table of contents: role → area → feature → screens and flows. */
export function Sidebar() {
  const t = useT()
  const { text } = useContentLang()
  const project = useProject()
  const { data } = useEntities()
  const { areas } = useConfigLabels()
  const all = data ?? []
  const needs = project.data ? project.data.total - (project.data.counts.approved ?? 0) : undefined

  const features = all.filter((e) => e.kind === 'feature')
  const roles = [...(project.data?.config.roles ?? [])]
  const roleless = features.some((f) => f.roles.length === 0)
  const areaOrder = [...areas.keys()]
  for (const f of features) if (f.area && !areaOrder.includes(f.area)) areaOrder.push(f.area)
  areaOrder.push('')

  const tree = (roleId: string | null) =>
    areaOrder
      .map((area) => ({
        area,
        feats: features.filter(
          (f) =>
            (f.area ?? '') === area &&
            (roleId === null ? f.roles.length === 0 : f.roles.includes(roleId)),
        ),
      }))
      .filter((g) => g.feats.length > 0)
      .map((g) => (
        <li key={g.area || 'other'} className="mt-1">
          <p className="px-2 pt-1 text-[11px] font-semibold tracking-wide text-slate-500 uppercase">
            {g.area ? text(areas.get(g.area)) || g.area : t('home.otherArea')}
          </p>
          <ul>
            {g.feats.map((f) => {
              const children = all.filter(
                (c) =>
                  (c.kind === 'screen' || c.kind === 'flow') &&
                  c.feature === f.id &&
                  (roleId === null || c.roles.includes(roleId)),
              )
              return (
                <li key={f.id}>
                  <ul>
                    <Item e={f} depth={0} />
                    {children.map((c) => (
                      <Item key={c.key} e={c} depth={1} />
                    ))}
                  </ul>
                </li>
              )
            })}
          </ul>
        </li>
      ))

  const modules = all.filter((e) => e.kind === 'module')

  return (
    <nav aria-label="Main" className="flex flex-col gap-4 p-3">
      <ul>
        <li>
          <NavLink to="/">{t('nav.home')}</NavLink>
        </li>
        <li>
          <NavLink to="/review">
            {t('nav.review')}
            {needs !== undefined && needs > 0 && (
              <span className="ml-auto rounded-full bg-amber-100 px-1.5 text-xs text-amber-800 tabular-nums dark:bg-amber-900 dark:text-amber-100">
                {needs}
              </span>
            )}
          </NavLink>
        </li>
        <li>
          <NavLink to="/glossary">{t('nav.glossary')}</NavLink>
        </li>
        <li>
          <NavLink to="/graph">{t('nav.graph')}</NavLink>
        </li>
      </ul>
      {roles.map((r) => (
        <details key={r.id} open className="group">
          <summary className="cursor-pointer rounded px-2 py-1 text-sm font-semibold hover:bg-slate-100 dark:hover:bg-slate-800">
            {text(r.label) || r.id}
          </summary>
          <ul>{tree(r.id)}</ul>
        </details>
      ))}
      {roleless && (
        <details open>
          <summary className="cursor-pointer rounded px-2 py-1 text-sm font-semibold hover:bg-slate-100 dark:hover:bg-slate-800">
            {t('nav.noRole')}
          </summary>
          <ul>{tree(null)}</ul>
        </details>
      )}
      {modules.length > 0 && (
        <details open>
          <summary className="cursor-pointer rounded px-2 py-1 text-sm font-semibold hover:bg-slate-100 dark:hover:bg-slate-800">
            {t('nav.modules')}
          </summary>
          <ul>
            {modules.map((m) => (
              <Item key={m.key} e={m} depth={0} />
            ))}
          </ul>
        </details>
      )}
    </nav>
  )
}
