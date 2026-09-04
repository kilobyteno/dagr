import { CaretDownIcon, FileTextIcon, PlusIcon } from '@phosphor-icons/react'
import { useState } from 'react'

import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { canCreateChildPage, type DocumentNode } from '@/lib/api/documents'
import { useLocale } from '@/lib/i18n'
import { cn } from '@/lib/utils'

function TreeItems({
  nodes,
  activeId,
  depth,
  onSelect,
  onCreateChild,
}: {
  nodes: readonly DocumentNode[]
  activeId: string
  depth: number
  onSelect: (id: string) => void
  onCreateChild: (parentId: string) => void
}) {
  const { t } = useLocale()
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})

  return (
    <SidebarMenu>
      {nodes.map((node) => {
        const hasChildren = node.children.length > 0
        const isCollapsed = collapsed[node.id]
        return (
          <SidebarMenuItem key={node.id}>
            <div
              className="flex h-8 w-full items-center"
              style={{ paddingLeft: `${0.5 + depth * 0.75}rem` }}
            >
              <div
                className={cn(
                  'flex min-w-0 flex-1 items-center rounded-md',
                  'hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
                  activeId === node.id &&
                    'bg-sidebar-accent font-medium text-sidebar-accent-foreground',
                )}
              >
                <span className="flex size-7 shrink-0 items-center justify-center">
                  {hasChildren ? (
                    <button
                      type="button"
                      className="flex size-7 items-center justify-center text-muted-foreground"
                      aria-label={
                        isCollapsed
                          ? t('docs.expandPage')
                          : t('docs.collapsePage')
                      }
                      onClick={() => {
                        setCollapsed((prev) => ({
                          ...prev,
                          [node.id]: !prev[node.id],
                        }))
                      }}
                    >
                      <CaretDownIcon
                        className={cn(
                          'size-3.5',
                          isCollapsed && '-rotate-90',
                        )}
                      />
                    </button>
                  ) : null}
                </span>
                <SidebarMenuButton
                  isActive={false}
                  onClick={() => onSelect(node.id)}
                  className="h-8 min-w-0 flex-1 gap-1.5 px-1 hover:bg-transparent hover:text-inherit active:bg-transparent data-active:bg-transparent data-active:font-normal data-open:hover:bg-transparent"
                >
                  <span className="flex size-4 shrink-0 items-center justify-center overflow-hidden text-[13px] leading-none">
                    {node.icon ? (
                      <span aria-hidden className="block size-4 text-center leading-4">
                        {node.icon}
                      </span>
                    ) : (
                      <FileTextIcon className="size-4 text-muted-foreground" />
                    )}
                  </span>
                  <span className="truncate">{node.title}</span>
                </SidebarMenuButton>
              </div>
              {canCreateChildPage(depth) ? (
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  className="mr-1 shrink-0 text-muted-foreground"
                  aria-label={t('docs.newChildPage')}
                  onClick={() => onCreateChild(node.id)}
                >
                  <PlusIcon />
                </Button>
              ) : null}
            </div>
            {hasChildren && !isCollapsed ? (
              <div className="w-full">
                <TreeItems
                  nodes={node.children}
                  activeId={activeId}
                  depth={depth + 1}
                  onSelect={onSelect}
                  onCreateChild={onCreateChild}
                />
              </div>
            ) : null}
          </SidebarMenuItem>
        )
      })}
    </SidebarMenu>
  )
}

export function DocsTree({
  tree,
  loading,
  activeId,
  onSelect,
  onCreate,
  className,
}: {
  tree: readonly DocumentNode[]
  loading?: boolean
  activeId: string
  onSelect: (id: string) => void
  onCreate: (parentId?: string) => void
  className?: string
}) {
  const { t } = useLocale()

  return (
    <Sidebar
      collapsible="none"
      className={cn('hidden w-[16.5rem]! md:flex', className)}
    >
      <SidebarContent className="overflow-hidden">
        <ScrollArea className="min-h-0 flex-1">
          <div className="flex flex-col gap-3 px-2 py-3">
            <SidebarGroup className="p-0">
              <SidebarGroupContent>
                {loading ? (
                  <p className="px-2 py-1 text-xs text-muted-foreground">
                    {t('docs.loading')}
                  </p>
                ) : tree.length === 0 ? (
                  <p className="px-2 py-1 text-xs text-muted-foreground">
                    {t('docs.empty')}
                  </p>
                ) : (
                  <TreeItems
                    nodes={tree}
                    activeId={activeId}
                    depth={0}
                    onSelect={onSelect}
                    onCreateChild={(parentId) => onCreate(parentId)}
                  />
                )}
                <SidebarMenu>
                  <SidebarMenuItem>
                    <SidebarMenuButton
                      onClick={() => onCreate()}
                      className="px-2 text-muted-foreground"
                      aria-label={t('docs.newPage')}
                    >
                      <PlusIcon />
                      <span className="truncate">{t('docs.newPage')}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>
          </div>
        </ScrollArea>
      </SidebarContent>
    </Sidebar>
  )
}
