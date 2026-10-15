import { describe, expect, it } from 'vitest'
import { githubUrl, vscodeUrl } from './code-links'

describe('githubUrl', () => {
  it('builds blob and tree permalinks', () => {
    const repo = { url: 'https://github.com/acme/expense.git', branch: 'main' }
    expect(githubUrl(repo, 'web/a.tsx')).toBe('https://github.com/acme/expense/blob/main/web/a.tsx')
    expect(githubUrl(repo, 'api/claims/', true)).toBe(
      'https://github.com/acme/expense/tree/main/api/claims',
    )
  })

  it('prefixes repo.path when the content root is a subfolder of the repository', () => {
    const repo = { url: 'https://github.com/acme/mono', branch: 'main', path: 'examples/demo/' }
    expect(githubUrl(repo, 'app/app.js')).toBe(
      'https://github.com/acme/mono/blob/main/examples/demo/app/app.js',
    )
  })

  it('returns null without a repository URL', () => {
    expect(githubUrl({}, 'a')).toBeNull()
  })
})

describe('vscodeUrl', () => {
  it('links into the local checkout', () => {
    expect(vscodeUrl('/home/me/acme', 'app/app.js')).toBe('vscode://file/home/me/acme/app/app.js')
    expect(vscodeUrl(undefined, 'a')).toBeNull()
  })
})
