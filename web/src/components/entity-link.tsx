import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import type { Kind } from '../api/types'

const ROUTES = {
  feature: '/features/$id',
  screen: '/screens/$id',
  flow: '/flows/$id',
  module: '/modules/$id',
} as const

export function entityRoute(kind: Kind) {
  return ROUTES[kind]
}

interface Props {
  kind: Kind
  id: string
  children: ReactNode
  className?: string
  hash?: string
}

/** Link to the page of any entity kind. */
export function EntityLink({ kind, id, children, className, hash }: Props) {
  return (
    <Link to={ROUTES[kind]} params={{ id }} hash={hash} className={className}>
      {children}
    </Link>
  )
}
