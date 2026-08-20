import {
  type ReactNode,
  Children,
  cloneElement,
  isValidElement,
} from 'react'
import ReactMarkdown from 'react-markdown'
import remarkBreaks from 'remark-breaks'
import remarkGfm from 'remark-gfm'
import type { Components } from 'react-markdown'

import { SafeExternalLink } from '@/components/chat/trusted-domains'
import {
  MessageBodyWithHandles,
  type ChatUserRef,
} from '@/components/chat/user-handle'
import { isEmojiOnlyMessage } from '@/lib/emoji'
import { cn } from '@/lib/utils'

function withMentions(
  children: ReactNode,
  usersByHandle: Map<string, ChatUserRef>,
  serverUrl?: string,
  token?: string,
): ReactNode {
  return Children.map(children, (child, index) => {
    if (typeof child === 'string') {
      return (
        <MessageBodyWithHandles
          key={`text-${index}`}
          body={child}
          usersByHandle={usersByHandle}
          serverUrl={serverUrl}
          token={token}
        />
      )
    }
    if (isValidElement<{ children?: ReactNode }>(child) && child.props.children) {
      return cloneElement(child, {
        children: withMentions(
          child.props.children,
          usersByHandle,
          serverUrl,
          token,
        ),
      })
    }
    return child
  })
}

function MarkdownChildren({
  children,
  usersByHandle,
  serverUrl,
  token,
}: {
  children?: ReactNode
  usersByHandle: Map<string, ChatUserRef>
  serverUrl?: string
  token?: string
}) {
  return <>{withMentions(children, usersByHandle, serverUrl, token)}</>
}

function safeHref(href?: string) {
  if (!href) return undefined
  const value = href.trim()
  if (!value) return undefined
  const lower = value.toLowerCase()
  if (
    lower.startsWith('http://') ||
    lower.startsWith('https://') ||
    lower.startsWith('mailto:')
  ) {
    return value
  }
  return undefined
}

export function MessageMarkdown({
  body,
  usersByHandle,
  className,
  serverUrl,
  token,
}: {
  body: string
  usersByHandle: Map<string, ChatUserRef>
  className?: string
  serverUrl?: string
  token?: string
}) {
  const mentionProps = { usersByHandle, serverUrl, token }
  const emojiOnly = isEmojiOnlyMessage(body)
  const components: Components = {
    p: ({ children }) => (
      <p className="mb-1 last:mb-0">
        <MarkdownChildren {...mentionProps}>{children}</MarkdownChildren>
      </p>
    ),
    a: ({ href, children }) => {
      const safe = safeHref(href)
      if (!safe) {
        return (
          <span>
            <MarkdownChildren {...mentionProps}>{children}</MarkdownChildren>
          </span>
        )
      }
      return (
        <SafeExternalLink
          href={safe}
          className="font-medium text-primary underline underline-offset-2"
        >
          <MarkdownChildren {...mentionProps}>{children}</MarkdownChildren>
        </SafeExternalLink>
      )
    },
    strong: ({ children }) => (
      <strong className="font-semibold text-foreground">
        <MarkdownChildren {...mentionProps}>{children}</MarkdownChildren>
      </strong>
    ),
    em: ({ children }) => (
      <em>
        <MarkdownChildren {...mentionProps}>{children}</MarkdownChildren>
      </em>
    ),
    del: ({ children }) => (
      <del className="text-muted-foreground">
        <MarkdownChildren {...mentionProps}>{children}</MarkdownChildren>
      </del>
    ),
    code: ({ className: codeClassName, children }) => {
      const isBlock = Boolean(codeClassName)
      if (isBlock) {
        return <code className={codeClassName}>{children}</code>
      }
      return (
        <code className="rounded bg-muted px-1 py-0.5 font-mono text-[0.85em] text-foreground">
          {children}
        </code>
      )
    },
    pre: ({ children }) => (
      <pre className="mb-2 overflow-x-auto rounded-md bg-muted p-3 font-mono text-[0.85em] text-foreground last:mb-0">
        {children}
      </pre>
    ),
    ul: ({ children }) => (
      <ul className="mb-1 list-disc pl-4 last:mb-0">{children}</ul>
    ),
    ol: ({ children }) => (
      <ol className="mb-1 list-decimal pl-4 last:mb-0">{children}</ol>
    ),
    li: ({ children }) => (
      <li className="leading-snug [&>p]:mb-0">
        <MarkdownChildren {...mentionProps}>{children}</MarkdownChildren>
      </li>
    ),
    blockquote: ({ children }) => (
      <blockquote className="mb-2 border-l-2 border-border pl-3 text-muted-foreground last:mb-0">
        {children}
      </blockquote>
    ),
    h1: ({ children }) => (
      <p className="mb-2 text-base font-semibold text-foreground last:mb-0">
        <MarkdownChildren {...mentionProps}>{children}</MarkdownChildren>
      </p>
    ),
    h2: ({ children }) => (
      <p className="mb-2 text-sm font-semibold text-foreground last:mb-0">
        <MarkdownChildren {...mentionProps}>{children}</MarkdownChildren>
      </p>
    ),
    h3: ({ children }) => (
      <p className="mb-2 text-sm font-semibold text-foreground last:mb-0">
        <MarkdownChildren {...mentionProps}>{children}</MarkdownChildren>
      </p>
    ),
    h4: ({ children }) => (
      <p className="mb-2 text-sm font-medium text-foreground last:mb-0">
        <MarkdownChildren {...mentionProps}>{children}</MarkdownChildren>
      </p>
    ),
    h5: ({ children }) => (
      <p className="mb-2 text-sm font-medium text-foreground last:mb-0">
        <MarkdownChildren {...mentionProps}>{children}</MarkdownChildren>
      </p>
    ),
    h6: ({ children }) => (
      <p className="mb-2 text-sm font-medium text-foreground last:mb-0">
        <MarkdownChildren {...mentionProps}>{children}</MarkdownChildren>
      </p>
    ),
    hr: () => <hr className="my-3 border-border" />,
    table: ({ children }) => (
      <div className="mb-2 overflow-x-auto last:mb-0">
        <table className="w-full border-collapse text-left text-sm">{children}</table>
      </div>
    ),
    thead: ({ children }) => <thead className="border-b border-border">{children}</thead>,
    th: ({ children }) => (
      <th className="px-2 py-1 font-medium text-foreground">
        <MarkdownChildren {...mentionProps}>{children}</MarkdownChildren>
      </th>
    ),
    td: ({ children }) => (
      <td className="border-t border-border px-2 py-1">
        <MarkdownChildren {...mentionProps}>{children}</MarkdownChildren>
      </td>
    ),
    img: () => null,
  }

  return (
    <div
      className={cn(
        'text-sm text-muted-foreground break-words',
        emojiOnly && 'text-[1.75rem] leading-none',
        className,
      )}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkBreaks]}
        components={components}
      >
        {body}
      </ReactMarkdown>
    </div>
  )
}
