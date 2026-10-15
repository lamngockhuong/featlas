import type { ReactNode } from 'react'

export function Section({
  title,
  children,
  id,
}: {
  title: ReactNode
  children: ReactNode
  id?: string
}) {
  return (
    <section id={id} className="mt-8">
      <h2 className="mb-3 text-base font-semibold text-slate-800 dark:text-slate-100">{title}</h2>
      {children}
    </section>
  )
}

export function Chip({ children, className = '' }: { children: ReactNode; className?: string }) {
  return (
    <span
      className={`inline-flex items-center rounded bg-slate-100 px-1.5 py-0.5 text-xs text-slate-700 dark:bg-slate-800 dark:text-slate-300 ${className}`}
    >
      {children}
    </span>
  )
}

export function Muted({ children }: { children: ReactNode }) {
  return <p className="text-sm text-slate-500 dark:text-slate-400">{children}</p>
}
