import { Link } from '@tanstack/react-router'
import { type ReactNode, useEffect, useRef } from 'react'
import type { Media } from '../api/types'
import { contentUrl } from '../api/url'
import { boxToPercent } from '../lib/hotspot'
import { useT } from '../lib/prefs'

/**
 * An image whose box keeps the real aspect ratio from the start, so overlays
 * positioned in percent never wait for the image to load.
 */
export function ShotFrame({
  media,
  alt,
  children,
  onZoom,
  className = '',
}: {
  media: Media | undefined
  alt: string
  children?: ReactNode
  onZoom?: () => void
  className?: string
}) {
  const t = useT()
  if (!media || !(media.w > 0) || !(media.h > 0)) {
    return (
      <div
        className={`flex aspect-video items-center justify-center rounded-md border border-dashed border-slate-300 text-sm text-slate-500 dark:border-slate-700 ${className}`}
      >
        {t('screen.noImage')}
      </div>
    )
  }
  const img = (
    <img
      src={contentUrl(media.url)}
      alt={alt}
      width={media.w}
      height={media.h}
      className="absolute inset-0 h-full w-full select-none"
      draggable={false}
    />
  )
  return (
    <div
      className={`relative w-full overflow-hidden rounded-md border border-slate-200 bg-slate-50 dark:border-slate-700 dark:bg-slate-900 ${className}`}
      style={{ aspectRatio: `${media.w} / ${media.h}` }}
    >
      {onZoom ? (
        <button
          type="button"
          onClick={onZoom}
          aria-label={t('screen.zoom')}
          className="absolute inset-0 cursor-zoom-in"
        >
          {img}
        </button>
      ) : (
        img
      )}
      {children}
    </div>
  )
}

export interface OverlaySpot {
  n: number
  label: string
  box: unknown
  to?: string
  toTitle?: string
}

/** Numbered, clickable hotspot boxes over a ShotFrame. */
export function HotspotOverlay({
  spots,
  media,
  active,
  onHover,
  onSelect,
}: {
  spots: OverlaySpot[]
  media: Media
  active: number | null
  onHover: (n: number | null) => void
  onSelect: (n: number) => void
}) {
  const t = useT()
  return (
    <>
      {spots.map((s) => {
        const pos = boxToPercent(s.box, media.w, media.h)
        if (!pos) return null
        const on = active === s.n
        const style = {
          left: `${pos.left}%`,
          top: `${pos.top}%`,
          width: `${pos.width}%`,
          height: `${pos.height}%`,
        }
        const cls = `group absolute rounded-sm border-2 outline-none transition-colors focus-visible:ring-4 focus-visible:ring-sky-400/60 ${
          on
            ? 'border-sky-500 bg-sky-400/20'
            : 'border-sky-500/70 bg-transparent hover:bg-sky-400/10'
        }`
        const badge = (
          <span
            aria-hidden="true"
            className={`absolute -top-2.5 -left-2.5 flex size-5 items-center justify-center rounded-full text-[11px] font-bold text-white shadow ${
              on ? 'bg-sky-600 ring-2 ring-white' : 'bg-sky-500'
            }`}
          >
            {s.n}
          </span>
        )
        const common = {
          style,
          className: cls,
          'data-hotspot': s.n,
          onMouseEnter: () => onHover(s.n),
          onMouseLeave: () => onHover(null),
          onFocus: () => onHover(s.n),
          onBlur: () => onHover(null),
        }
        if (s.to) {
          const label = `${s.n}. ${s.label} — ${t('screen.goTo', { title: s.toTitle ?? s.to })}`
          return (
            <Link key={s.n} to="/screens/$id" params={{ id: s.to }} aria-label={label} {...common}>
              {badge}
            </Link>
          )
        }
        return (
          <button
            key={s.n}
            type="button"
            aria-label={`${s.n}. ${s.label}`}
            aria-pressed={on}
            onClick={() => onSelect(s.n)}
            {...common}
          >
            {badge}
          </button>
        )
      })}
    </>
  )
}

/** A single highlighted area, used for flow steps. */
export function BoxHighlight({ box, media, label }: { box: unknown; media: Media; label: string }) {
  const pos = boxToPercent(box, media.w, media.h)
  if (!pos) return null
  return (
    <span
      role="img"
      aria-label={label}
      className="pointer-events-none absolute rounded-sm border-2 border-rose-500 bg-rose-400/15 shadow-[0_0_0_9999px_rgba(15,23,42,0.18)]"
      style={{
        left: `${pos.left}%`,
        top: `${pos.top}%`,
        width: `${pos.width}%`,
        height: `${pos.height}%`,
      }}
    />
  )
}

/** Full-screen view of a screenshot; Escape or the close button dismisses it. */
export function Lightbox({
  onClose,
  label,
  children,
}: {
  onClose: () => void
  label: string
  children: ReactNode
}) {
  const t = useT()
  const closeRef = useRef<HTMLButtonElement>(null)
  useEffect(() => {
    const prev = document.activeElement as HTMLElement | null
    closeRef.current?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('keydown', onKey)
      prev?.focus()
    }
  }, [onClose])
  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={label}
      className="fixed inset-0 z-50 flex flex-col bg-slate-950/90 p-4"
    >
      <div className="flex justify-end">
        <button
          ref={closeRef}
          type="button"
          onClick={onClose}
          className="rounded bg-white/10 px-3 py-1 text-sm text-white hover:bg-white/20"
        >
          {t('screen.closeZoom')}
        </button>
      </div>
      <div className="flex min-h-0 flex-1 items-center justify-center overflow-auto py-4">
        <div className="w-full max-w-[1600px]">{children}</div>
      </div>
    </div>
  )
}
