import { createMemoryHistory } from '@tanstack/react-router'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Project, Summary } from '../api/types'
import { App, makeQueryClient } from '../app'

const project: Project = {
  config: {
    muninn: 1,
    project: { name: 'Test project' },
    languages: { default: 'en', available: ['en'] },
    areas: [{ id: 'claims', label: { en: 'Claims' } }],
  },
  counts: { draft: 1, changes_requested: 0, modified: 0, outdated: 1, approved: 0 },
  total: 2,
  validation: { errors: 0, warnings: 0 },
  capabilities: { canReview: true, needsToken: false, static: false },
  version: 'test',
  generation: 1,
}

const summary = (over: Partial<Summary>): Summary => ({
  key: 'screen/claim-list',
  kind: 'screen',
  id: 'claim-list',
  title: { en: 'My claims' },
  area: 'claims',
  roles: [],
  owners: ['Emma'],
  status: { state: 'draft' },
  review: { status: 'draft' },
  errors: 0,
  warnings: 0,
  file: 'screens/claim-list/screen.md',
  revision: 'sha256:rev-claim-list',
  ...over,
})

const entities: Summary[] = [
  summary({}),
  summary({
    key: 'flow/submit-claim',
    kind: 'flow',
    id: 'submit-claim',
    title: { en: 'Submit a claim' },
    status: { state: 'outdated', reasons: [{ kind: 'source-changed', path: 'app/claim.html' }] },
    review: { status: 'approved', by: 'Liam', at: '2026-10-01T00:00:00Z' },
  }),
]

type FetchArgs = [string, RequestInit | undefined]
let fetchMock: ReturnType<typeof vi.fn<(...a: FetchArgs) => Promise<Response>>>

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })

beforeEach(() => {
  fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
    if (init?.method === 'POST') {
      return json({ ...entities[0], status: { state: 'approved' }, front: {}, bodies: {} })
    }
    if (url === '/api/project') return json(project)
    if (url === '/api/entities') return json(entities)
    if (url === '/api/validation') return json({ errors: 0, warnings: 0, issues: [] })
    return json({ error: 'not found' }, 404)
  })
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => {
  vi.unstubAllGlobals()
})

function renderReview() {
  const history = createMemoryHistory({ initialEntries: ['/review'] })
  render(<App history={history} queryClient={makeQueryClient()} />)
}

function posts() {
  return fetchMock.mock.calls.filter(([, init]) => init?.method === 'POST')
}

async function rowFor(title: string) {
  const link = await screen.findByRole('link', { name: title }, { timeout: 5000 })
  const row = link.closest('[data-testid="review-row"]')
  if (!(row instanceof HTMLElement)) throw new Error(`no review row for ${title}`)
  return row
}

describe('review queue', () => {
  it('lists entities that need review with their outdated reasons', async () => {
    renderReview()
    const row = await rowFor('Submit a claim')
    expect(within(row).getByText('app/claim.html')).toBeInTheDocument()
    expect(within(row).getByText(/changed since approval/)).toBeInTheDocument()
  })

  it('posts an approval for the clicked entity with the stored reviewer name', async () => {
    localStorage.setItem('muninn.reviewer', 'Ana')
    const user = userEvent.setup()
    renderReview()
    const row = await rowFor('My claims')
    await user.click(within(row).getByRole('button', { name: 'Approve' }))

    await waitFor(() => expect(posts()).toHaveLength(1))
    const [url, init] = posts()[0] as FetchArgs
    expect(url).toBe('/api/entities/screen/claim-list/review')
    expect(new Headers(init?.headers).get('Content-Type')).toBe('application/json')
    expect(JSON.parse(String(init?.body))).toEqual({
      status: 'approved',
      by: 'Ana',
      note: '',
      revision: 'sha256:rev-claim-list',
    })
  })

  it('asks for the reviewer name on the first review and remembers it', async () => {
    const user = userEvent.setup()
    renderReview()
    const row = await rowFor('My claims')
    await user.click(within(row).getByRole('button', { name: 'Approve' }))

    const dialog = await screen.findByRole('dialog', { name: 'Who is reviewing?' })
    expect(posts()).toHaveLength(0)
    await user.type(within(dialog).getByLabelText('Your name'), 'Bao')
    await user.click(within(dialog).getByRole('button', { name: 'Continue' }))

    await waitFor(() => expect(posts()).toHaveLength(1))
    expect(JSON.parse(String(posts()[0]?.[1]?.body))).toMatchObject({ by: 'Bao' })
    expect(localStorage.getItem('muninn.reviewer')).toBe('Bao')
  })

  it('requires a note before requesting changes', async () => {
    localStorage.setItem('muninn.reviewer', 'Ana')
    const user = userEvent.setup()
    renderReview()
    const row = await rowFor('My claims')
    await user.click(within(row).getByRole('button', { name: 'Request changes' }))
    await user.click(within(row).getByRole('button', { name: 'Send' }))
    expect(within(row).getByText('A note is required when requesting changes.')).toBeVisible()
    expect(posts()).toHaveLength(0)

    await user.type(within(row).getByLabelText('Note'), 'Hotspot 2 is wrong')
    await user.click(within(row).getByRole('button', { name: 'Send' }))
    await waitFor(() => expect(posts()).toHaveLength(1))
    expect(JSON.parse(String(posts()[0]?.[1]?.body))).toEqual({
      status: 'changes_requested',
      by: 'Ana',
      note: 'Hotspot 2 is wrong',
      revision: 'sha256:rev-claim-list',
    })
  })
})
