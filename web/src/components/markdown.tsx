import ReactMarkdown, { type Components } from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { contentUrl } from '../api/url'
import { MermaidDiagram } from './mermaid-diagram'

const components: Components = {
  code({ className, children, ...rest }) {
    const text = String(children ?? '')
    if (/\blanguage-mermaid\b/.test(className ?? '')) {
      return <MermaidDiagram code={text.replace(/\n$/, '')} />
    }
    return (
      <code className={className} {...rest}>
        {children}
      </code>
    )
  },
  pre({ children }) {
    return <div className="md-pre">{children}</div>
  },
  img({ src, alt }) {
    const url = typeof src === 'string' ? contentUrl(src) : undefined
    return <img src={url} alt={alt ?? ''} loading="lazy" className="max-w-full" />
  },
  a({ href, children }) {
    const external = href ? /^[a-z][a-z0-9+.-]*:/i.test(href) : false
    return (
      <a href={href} {...(external ? { target: '_blank', rel: 'noreferrer' } : {})}>
        {children}
      </a>
    )
  },
}

/** Renders an entity body; ```mermaid blocks become diagrams. Raw HTML is not rendered. */
export function Markdown({ text }: { text: string }) {
  return (
    <div className="md">
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
        {text}
      </ReactMarkdown>
    </div>
  )
}
