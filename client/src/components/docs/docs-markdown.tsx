import {
  type ReactNode,
  Children,
  cloneElement,
  isValidElement,
} from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import type { Components } from 'react-markdown'

import { MessageBodyWithHandles } from '@/components/chat/user-handle'
import { cn } from '@/lib/utils'

function withEmoji(children: ReactNode): ReactNode {
  return Children.map(children, (child, index) => {
    if (typeof child === 'string') {
      return (
        <MessageBodyWithHandles
          key={`text-${index}`}
          body={child}
          usersByHandle={new Map()}
        />
      )
    }
    if (isValidElement<{ children?: ReactNode }>(child) && child.props.children) {
      return cloneElement(child, {
        children: withEmoji(child.props.children),
      })
    }
    return child
  })
}

function MarkdownChildren({ children }: { children?: ReactNode }) {
  return <>{withEmoji(children)}</>
}

const components: Components = {
  p: ({ children }) => (
    <p>
      <MarkdownChildren>{children}</MarkdownChildren>
    </p>
  ),
  a: ({ href, children }) => (
    <a href={href}>
      <MarkdownChildren>{children}</MarkdownChildren>
    </a>
  ),
  strong: ({ children }) => (
    <strong>
      <MarkdownChildren>{children}</MarkdownChildren>
    </strong>
  ),
  em: ({ children }) => (
    <em>
      <MarkdownChildren>{children}</MarkdownChildren>
    </em>
  ),
  del: ({ children }) => (
    <del>
      <MarkdownChildren>{children}</MarkdownChildren>
    </del>
  ),
  li: ({ children }) => (
    <li>
      <MarkdownChildren>{children}</MarkdownChildren>
    </li>
  ),
  h1: ({ children }) => (
    <h1>
      <MarkdownChildren>{children}</MarkdownChildren>
    </h1>
  ),
  h2: ({ children }) => (
    <h2>
      <MarkdownChildren>{children}</MarkdownChildren>
    </h2>
  ),
  h3: ({ children }) => (
    <h3>
      <MarkdownChildren>{children}</MarkdownChildren>
    </h3>
  ),
  h4: ({ children }) => (
    <h4>
      <MarkdownChildren>{children}</MarkdownChildren>
    </h4>
  ),
  h5: ({ children }) => (
    <h5>
      <MarkdownChildren>{children}</MarkdownChildren>
    </h5>
  ),
  h6: ({ children }) => (
    <h6>
      <MarkdownChildren>{children}</MarkdownChildren>
    </h6>
  ),
  th: ({ children }) => (
    <th>
      <MarkdownChildren>{children}</MarkdownChildren>
    </th>
  ),
  td: ({ children }) => (
    <td>
      <MarkdownChildren>{children}</MarkdownChildren>
    </td>
  ),
}

export function DocsMarkdown({
  body,
  className,
}: {
  body: string
  className?: string
}) {
  return (
    <article
      className={cn(
        'text-sm leading-relaxed text-foreground [&_a]:font-medium [&_a]:text-primary [&_a]:underline [&_blockquote]:border-l-2 [&_blockquote]:border-border [&_blockquote]:pl-3 [&_blockquote]:text-muted-foreground [&_code]:rounded [&_code]:bg-muted [&_code]:px-1 [&_h1]:mb-3 [&_h1]:text-2xl [&_h1]:font-semibold [&_h2]:mb-2 [&_h2]:text-xl [&_h2]:font-semibold [&_h3]:mb-2 [&_h3]:text-lg [&_h3]:font-semibold [&_li]:leading-snug [&_ol]:mb-3 [&_ol]:list-decimal [&_ol]:pl-5 [&_p]:mb-3 [&_pre]:mb-3 [&_pre]:overflow-x-auto [&_pre]:rounded-md [&_pre]:bg-muted [&_pre]:p-3 [&_ul]:mb-3 [&_ul]:list-disc [&_ul]:pl-5',
        className,
      )}
    >
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
        {body}
      </ReactMarkdown>
    </article>
  )
}
