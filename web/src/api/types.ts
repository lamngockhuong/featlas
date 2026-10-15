// Response shapes of the muninn JSON API, written by hand after the Go
// server types (internal/server/snapshot.go). Frontmatter stays loosely typed
// because the binary passes it through as parsed YAML.

export type Localized = Record<string, string>
export type Kind = 'feature' | 'screen' | 'flow' | 'module'
export const KINDS: Kind[] = ['feature', 'screen', 'flow', 'module']

export type ReviewState = 'draft' | 'changes_requested' | 'modified' | 'outdated' | 'approved'
export const REVIEW_STATES: ReviewState[] = [
  'draft',
  'changes_requested',
  'modified',
  'outdated',
  'approved',
]

export type ReasonKind = 'content' | 'source-changed' | 'source-missing' | 'source-added'

export interface Reason {
  kind: ReasonKind
  path?: string
}

export interface Status {
  state: ReviewState
  reasons?: Reason[]
}

export interface Review {
  status: 'draft' | 'approved' | 'changes_requested'
  by?: string
  at?: string
  note?: string
  contentHash?: string
  sourceHashes?: Record<string, string>
}

export interface Media {
  url: string
  w: number
  h: number
  bytes: number
}

export interface Labelled {
  id: string
  label: Localized
}

export interface Config {
  muninn: number
  project: { name: string; summary?: Localized }
  languages: { default: string; available?: string[] }
  repo?: { root?: string; url?: string; branch?: string; path?: string }
  roles?: Labelled[]
  areas?: Labelled[]
  owners?: Record<string, string[]>
}

export interface Capabilities {
  canReview: boolean
  needsToken: boolean
  readOnlyReason?: string
  static: boolean
  localRepo?: string
}

export interface Project {
  config: Config
  counts: Record<ReviewState, number>
  total: number
  validation: { errors: number; warnings: number }
  capabilities: Capabilities
  version: string
  generation: number
}

export interface Summary {
  key: string
  kind: Kind
  id: string
  title: Localized | string
  summary?: Localized | string
  area?: string
  feature?: string
  roles: string[]
  owners: string[]
  status: Status
  review: Review
  thumb?: Media
  counts?: Record<string, number>
  errors: number
  warnings: number
  file: string
  /** Content hash of the entity without its review block; sent back with a review. */
  revision: string
}

export interface Ref {
  kind: Kind | 'term'
  id: string
  field: string
}

export interface Issue {
  severity: 'error' | 'warning'
  file: string
  entity?: string
  path?: string
  message: string
}

export interface Source {
  path: string
  dir: boolean
  change?: Exclude<ReasonKind, 'content'>
}

export interface Detail extends Summary {
  // biome-ignore lint/suspicious/noExplicitAny: frontmatter is free-form YAML; pages read it through the typed views below
  front: Record<string, any>
  bodies: Record<string, string>
  translations: string[]
  refsTo: Ref[]
  issues: Issue[]
  media: Record<string, Media>
  steps?: { n: number; media?: Media }[]
  sources: Source[]
}

// Typed views of frontmatter, following schema/muninn.schema.json.

export interface Hotspot {
  n: number
  label: Localized
  desc?: Localized
  box: number[]
  kind?: 'control' | 'hidden' | 'link'
  to?: string
}

export interface Variant {
  id: string
  label: Localized
  shot: string
  hotspots?: Hotspot[]
}

export interface ScreenFront {
  ref?: string
  route?: string
  variants: Variant[]
  fields?: { label: string; meaning?: Localized; source?: string }[]
  api?: { method: string; path: string; note?: Localized }[]
  tables?: string[]
  modules?: string[]
  links?: { to: string; how?: Localized }[]
  captured?: { at?: string; branch?: string; backend?: string; viewport?: string }
}

export interface FeatureFront {
  area: string
  roles?: string[]
  summary?: Localized
  screens?: string[]
  flows?: string[]
  modules?: string[]
  terms?: string[]
  specs?: { label: string; url: string }[]
}

export type StepAction =
  | 'click'
  | 'type'
  | 'select'
  | 'check'
  | 'upload'
  | 'run'
  | 'wait'
  | 'result'

export interface FlowStep {
  n: number
  action: StepAction
  label: Localized
  desc?: Localized
  screen?: string
  variant?: string
  shot?: string
  box?: number[]
  sample?: { label: string; value: string; note?: Localized }[]
  run?: { cmd: string; where?: Localized }
}

export interface FlowFront {
  role: string
  goal?: Localized
  start?: string
  end?: string
  steps: FlowStep[]
}

export interface ModuleFront {
  kind: 'backend' | 'frontend' | 'job' | 'package' | 'other'
  paths?: string[]
  endpoints?: { method: string; path: string; roles?: string[]; note?: Localized }[]
  tables?: string[]
  dependsOn?: string[]
}

export interface SearchDoc {
  key: string
  kind: Kind | 'term'
  id: string
  title: Localized
  text: string
  route?: string
}

export interface GraphNode {
  key: string
  kind: Kind
  id: string
  title: Localized | string | null
  state: ReviewState
}

export interface GraphEdge {
  from: string
  to: string
  field: string
}

export interface Graph {
  nodes: GraphNode[]
  edges: GraphEdge[]
}

export interface Validation {
  errors: number
  warnings: number
  issues: Issue[]
}

export interface Term {
  id: string
  term: Localized
  meaning: Localized
  related?: string[]
  refsTo: Ref[]
}

export interface ReviewRequest {
  status: 'approved' | 'changes_requested'
  by: string
  note?: string
  revision?: string
}
