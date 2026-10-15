import { useProject } from '../api/client'
import type { Source } from '../api/types'
import { githubUrl, vscodeUrl } from '../lib/code-links'
import { useT } from '../lib/prefs'

interface Item {
  path: string
  dir?: boolean
  change?: Source['change']
}

/** Repository paths with GitHub and VS Code links and per-file change markers. */
export function SourceList({ items }: { items: Item[] }) {
  const t = useT()
  const { data } = useProject()
  if (items.length === 0) return <p className="text-sm text-slate-500">{t('entity.none')}</p>
  const repo = data?.config.repo
  const local = data?.capabilities.localRepo
  return (
    <ul className="space-y-1 text-sm">
      {items.map((s) => {
        const gh = githubUrl(repo, s.path, s.dir ?? s.path.endsWith('/'))
        const vs = vscodeUrl(local, s.path)
        return (
          <li key={s.path} className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <code className="break-all rounded bg-slate-100 px-1.5 py-0.5 text-xs dark:bg-slate-800">
              {s.path}
            </code>
            {gh && (
              <a
                href={gh}
                target="_blank"
                rel="noreferrer"
                className="text-xs text-sky-700 hover:underline dark:text-sky-400"
                aria-label={`${t('code.github')}: ${s.path}`}
              >
                GitHub
              </a>
            )}
            {vs && (
              <a
                href={vs}
                className="text-xs text-sky-700 hover:underline dark:text-sky-400"
                aria-label={`${t('code.vscode')}: ${s.path}`}
              >
                VS Code
              </a>
            )}
            {s.change && (
              <span
                data-change={s.change}
                className="rounded bg-amber-100 px-1.5 py-0.5 text-xs text-amber-800 dark:bg-amber-950 dark:text-amber-200"
              >
                {t(`source.${s.change}`)}
              </span>
            )}
          </li>
        )
      })}
    </ul>
  )
}
