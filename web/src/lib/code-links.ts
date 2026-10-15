import type { Config } from '../api/types'

/** GitHub-style permalink for a repository-relative path, or null without repo.url. */
export function githubUrl(repo: Config['repo'], path: string, dir = false): string | null {
  if (!repo?.url) return null
  const base = repo.url.replace(/\/+$/, '').replace(/\.git$/, '')
  const branch = repo.branch || 'HEAD'
  // repo.path is where the content's repository root sits inside repo.url.
  const prefix = (repo.path ?? '').replace(/^\/+/, '').replace(/\/+$/, '')
  const clean = [prefix, path.replace(/^\/+/, '').replace(/\/+$/, '')].filter(Boolean).join('/')
  const encoded = clean.split('/').map(encodeURIComponent).join('/')
  return `${base}/${dir ? 'tree' : 'blob'}/${encodeURIComponent(branch)}/${encoded}`
}

/** vscode://file link into the local checkout, or null when the server is not local. */
export function vscodeUrl(localRepo: string | undefined, path: string): string | null {
  if (!localRepo) return null
  const root = localRepo.replaceAll('\\', '/').replace(/\/+$/, '').replace(/^\/+/, '')
  const clean = path.replace(/^\/+/, '')
  return `vscode://file/${encodeURI(`${root}/${clean}`)}`
}
