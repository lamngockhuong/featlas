import { Link, useLocation, useNavigate, useParams, useSearch } from '@tanstack/react-router'
import { type KeyboardEvent, useState } from 'react'
import { useEntity } from '../api/client'
import type { Hotspot, ScreenFront, Variant } from '../api/types'
import { RoleNames } from '../components/config-labels'
import { EntityBody } from '../components/entity-body'
import { EntityHeader } from '../components/entity-header'
import { EntityLink } from '../components/entity-link'
import { EntityList, refIds, useEntityTitle } from '../components/entity-refs'
import { Loc } from '../components/loc-text'
import { ErrorBox, Loading } from '../components/query-state'
import { Chip, Muted, Section } from '../components/section'
import { HotspotOverlay, Lightbox, type OverlaySpot, ShotFrame } from '../components/shot-frame'
import { SourceList } from '../components/source-list'
import { hotspotFromHash } from '../lib/hotspot'
import { useContentLang, useT } from '../lib/prefs'

export function ScreenPage() {
  const { id } = useParams({ from: '/screens/$id' })
  const q = useEntity('screen', id)
  if (q.isPending) return <Loading />
  if (q.isError) return <ErrorBox error={q.error} retry={() => q.refetch()} />
  return <ScreenView key={id} />
}

function ScreenView() {
  const t = useT()
  const { text } = useContentLang()
  const title = useEntityTitle()
  const { id } = useParams({ from: '/screens/$id' })
  const { v } = useSearch({ from: '/screens/$id' })
  const hash = useLocation({ select: (l) => l.hash })
  const navigate = useNavigate({ from: '/screens/$id' })
  const { data: d } = useEntity('screen', id)
  const [hovered, setHovered] = useState<number | null>(null)
  const [zoom, setZoom] = useState(false)
  if (!d) return null

  const front = d.front as ScreenFront
  const variants: Variant[] = Array.isArray(front.variants) ? front.variants : []
  const variant = variants.find((x) => x.id === v) ?? variants[0]
  const media = variant ? d.media[variant.shot] : undefined
  const hotspots: Hotspot[] = variant?.hotspots ?? []
  const selected = hotspotFromHash(hash)
  const active = hovered ?? selected
  const selectedSpot = hotspots.find((h) => h.n === selected)

  const select = (n: number) =>
    navigate({
      search: (prev) => prev,
      hash: `h${n}`,
      replace: true,
      resetScroll: false,
      hashScrollIntoView: false,
    })
  const chooseVariant = (vid: string) =>
    navigate({
      search: (prev) => ({ ...prev, v: vid === variants[0]?.id ? undefined : vid }),
      hash: '',
      replace: true,
      resetScroll: false,
    })
  const onTabKey = (e: KeyboardEvent, i: number) => {
    const next = e.key === 'ArrowRight' ? i + 1 : e.key === 'ArrowLeft' ? i - 1 : null
    if (next === null) return
    const target = variants[(next + variants.length) % variants.length]
    if (target) {
      e.preventDefault()
      chooseVariant(target.id)
      document.getElementById(`variant-tab-${target.id}`)?.focus()
    }
  }

  const spots: OverlaySpot[] = hotspots.map((h) => ({
    n: h.n,
    label: text(h.label),
    box: h.box,
    to: h.to,
    toTitle: h.to ? title('screen', h.to) : undefined,
  }))
  const overlay = media && (
    <HotspotOverlay
      spots={spots}
      media={media}
      active={active}
      onHover={setHovered}
      onSelect={select}
    />
  )
  const flows = refIds(d.refsTo, 'flow')
  const fromScreens = refIds(d.refsTo, 'screen')
  const captured = front.captured

  return (
    <article className="mx-auto max-w-7xl px-4 py-6 sm:px-6">
      <EntityHeader
        entity={d}
        issues={d.issues}
        meta={
          <>
            {front.ref && <Chip>{front.ref}</Chip>}
            {front.route && (
              <span>
                <span className="text-slate-500">{t('screen.route')}: </span>
                <code className="text-xs">{front.route}</code>
              </span>
            )}
            {d.feature && (
              <span>
                <span className="text-slate-500">{t('entity.feature')}: </span>
                <EntityLink kind="feature" id={d.feature} className="link">
                  {title('feature', d.feature)}
                </EntityLink>
              </span>
            )}
            {d.roles.length > 0 && (
              <span>
                <span className="text-slate-500">{t('entity.roles')}: </span>
                <RoleNames ids={d.roles} />
              </span>
            )}
          </>
        }
      />

      {variants.length > 1 && (
        <div role="tablist" aria-label={t('screen.variants')} className="mt-5 flex flex-wrap gap-1">
          {variants.map((x, i) => {
            const on = x.id === variant?.id
            return (
              <button
                key={x.id}
                id={`variant-tab-${x.id}`}
                type="button"
                role="tab"
                aria-selected={on}
                tabIndex={on ? 0 : -1}
                onClick={() => chooseVariant(x.id)}
                onKeyDown={(e) => onTabKey(e, i)}
                className={`rounded-md px-3 py-1.5 text-sm ${
                  on
                    ? 'bg-slate-900 text-white dark:bg-slate-100 dark:text-slate-900'
                    : 'bg-slate-100 text-slate-700 hover:bg-slate-200 dark:bg-slate-800 dark:text-slate-200'
                }`}
              >
                <Loc v={x.label} fallback={x.id} />
              </button>
            )
          })}
        </div>
      )}

      <div className="mt-4 grid gap-6 lg:grid-cols-[minmax(0,1fr)_20rem]">
        <div>
          <ShotFrame media={media} alt={text(d.title) || d.id} onZoom={() => setZoom(true)}>
            {overlay}
          </ShotFrame>
          {selectedSpot && (
            <div
              aria-live="polite"
              className="mt-3 rounded-md border border-sky-200 bg-sky-50 p-3 text-sm dark:border-sky-900 dark:bg-sky-950/50"
            >
              <p className="font-medium">
                {selectedSpot.n}. <Loc v={selectedSpot.label} />
              </p>
              {selectedSpot.desc && (
                <p className="mt-1 text-slate-700 dark:text-slate-300">
                  <Loc v={selectedSpot.desc} />
                </p>
              )}
            </div>
          )}
          {captured?.at && (
            <p className="mt-2 text-xs text-slate-500">
              {t('screen.captured', { at: captured.at, backend: captured.backend ?? '?' })}
              {captured.viewport ? ` · ${captured.viewport}` : ''}
            </p>
          )}
        </div>

        <aside aria-label={t('screen.callouts')}>
          <h2 className="mb-2 text-sm font-semibold">{t('screen.callouts')}</h2>
          {hotspots.length === 0 ? (
            <Muted>{t('screen.noCallouts')}</Muted>
          ) : (
            <ol className="space-y-1.5">
              {hotspots.map((h) => {
                const on = active === h.n
                return (
                  <li
                    key={h.n}
                    id={`callout-${h.n}`}
                    data-callout={h.n}
                    data-active={on}
                    onMouseEnter={() => setHovered(h.n)}
                    onMouseLeave={() => setHovered(null)}
                    className={`rounded-md border p-2 text-sm transition-colors ${
                      on
                        ? 'border-sky-400 bg-sky-50 dark:border-sky-700 dark:bg-sky-950/50'
                        : 'border-slate-200 dark:border-slate-800'
                    }`}
                  >
                    <button
                      type="button"
                      onClick={() => select(h.n)}
                      onFocus={() => setHovered(h.n)}
                      onBlur={() => setHovered(null)}
                      aria-current={selected === h.n ? 'true' : undefined}
                      className="flex w-full items-start gap-2 text-left"
                    >
                      <span className="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full bg-sky-500 text-[11px] font-bold text-white">
                        {h.n}
                      </span>
                      <span className="font-medium">
                        <Loc v={h.label} />
                        {h.kind && h.kind !== 'control' && (
                          <Chip className="ml-1.5 align-middle">{h.kind}</Chip>
                        )}
                      </span>
                    </button>
                    {h.desc && (
                      <p className="mt-1 pl-7 text-slate-600 dark:text-slate-400">
                        <Loc v={h.desc} />
                      </p>
                    )}
                    {h.to && (
                      <p className="mt-1 pl-7">
                        <Link to="/screens/$id" params={{ id: h.to }} className="link text-xs">
                          → {title('screen', h.to)}
                        </Link>
                      </p>
                    )}
                  </li>
                )
              })}
            </ol>
          )}
        </aside>
      </div>

      {zoom && media && (
        <Lightbox onClose={() => setZoom(false)} label={text(d.title) || d.id}>
          <ShotFrame media={media} alt={text(d.title) || d.id}>
            {overlay}
          </ShotFrame>
        </Lightbox>
      )}

      <EntityBody bodies={d.bodies} />

      <Section title={t('screen.behind')}>
        <dl className="grid gap-x-6 gap-y-4 text-sm sm:grid-cols-2">
          {front.route && (
            <div>
              <dt className="font-medium text-slate-500">{t('screen.route')}</dt>
              <dd>
                <code className="text-xs">{front.route}</code>
              </dd>
            </div>
          )}
          {front.api && front.api.length > 0 && (
            <div>
              <dt className="font-medium text-slate-500">{t('screen.api')}</dt>
              <dd>
                <ul className="space-y-1">
                  {front.api.map((a) => (
                    <li key={`${a.method} ${a.path}`}>
                      <Chip className="mr-1 font-mono">{a.method}</Chip>
                      <code className="text-xs break-all">{a.path}</code>
                      {a.note && (
                        <span className="ml-1 text-slate-500">
                          — <Loc v={a.note} />
                        </span>
                      )}
                    </li>
                  ))}
                </ul>
              </dd>
            </div>
          )}
          {front.tables && front.tables.length > 0 && (
            <div>
              <dt className="font-medium text-slate-500">{t('screen.tables')}</dt>
              <dd className="flex flex-wrap gap-1">
                {front.tables.map((x) => (
                  <Chip key={x} className="font-mono">
                    {x}
                  </Chip>
                ))}
              </dd>
            </div>
          )}
          <div>
            <dt className="font-medium text-slate-500">{t('entity.modules')}</dt>
            <dd>
              <EntityList kind="module" ids={front.modules} />
            </dd>
          </div>
          <div className="sm:col-span-2">
            <dt className="font-medium text-slate-500">{t('entity.code')}</dt>
            <dd>
              <SourceList items={d.sources} />
            </dd>
          </div>
        </dl>
      </Section>

      {front.fields && front.fields.length > 0 && (
        <Section title={t('screen.fields')}>
          <div className="overflow-x-auto">
            <table className="table">
              <thead>
                <tr>
                  <th>{t('screen.label')}</th>
                  <th>{t('screen.meaning')}</th>
                  <th>{t('screen.source')}</th>
                </tr>
              </thead>
              <tbody>
                {front.fields.map((f) => (
                  <tr key={f.label}>
                    <td className="font-medium">{f.label}</td>
                    <td>
                      <Loc v={f.meaning} />
                    </td>
                    <td>{f.source && <code className="text-xs">{f.source}</code>}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Section>
      )}

      <div className="grid gap-x-6 sm:grid-cols-3">
        <Section title={t('screen.next')}>
          {front.links?.length ? (
            <ul className="space-y-1 text-sm">
              {front.links.map((l) => (
                <li key={l.to}>
                  <Link to="/screens/$id" params={{ id: l.to }} className="link">
                    {title('screen', l.to)}
                  </Link>
                  {l.how && (
                    <span className="text-slate-500">
                      {' '}
                      — <Loc v={l.how} />
                    </span>
                  )}
                </li>
              ))}
            </ul>
          ) : (
            <Muted>{t('entity.none')}</Muted>
          )}
        </Section>
        <Section title={t('screen.from')}>
          <EntityList kind="screen" ids={fromScreens} />
        </Section>
        <Section title={t('screen.inFlows')}>
          <EntityList kind="flow" ids={flows} />
        </Section>
      </div>
    </article>
  )
}
