import { useState } from 'react'
import { ChatCircleIcon, GearIcon, PlusIcon } from '@phosphor-icons/react'

import { WorkspaceIconMark } from '@/components/chat/workspace-icon'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar'
import { useLocale } from '@/lib/i18n'
import { cn } from '@/lib/utils'

export type ModuleRailWorkspace = {
  railKey: string
  id: string
  name: string
  serverUrl: string
  token: string
  serverLabel?: string
  hasIcon?: boolean
  iconUpdatedAt?: string
}

export const SHELL_MODULES = [
  {
    id: 'chat',
    icon: ChatCircleIcon,
    labelKey: 'chat.moduleChat',
  },
  {
    id: 'settings',
    icon: GearIcon,
    labelKey: 'chat.moduleSettings',
  },
] as const

export type ShellModule = (typeof SHELL_MODULES)[number]['id']

function workspaceLabel(
  workspace: ModuleRailWorkspace,
  workspaces: readonly ModuleRailWorkspace[],
) {
  const mixedServers =
    workspaces.length > 1 &&
    workspaces.some((item) => item.serverLabel !== workspace.serverLabel)
  return mixedServers && workspace.serverLabel
    ? `${workspace.name} · ${workspace.serverLabel}`
    : workspace.name
}

function WorkspaceSwitcherMark({
  workspace,
  stacked,
  active,
}: {
  workspace: ModuleRailWorkspace
  stacked?: boolean
  active?: boolean
}) {
  return (
    <span className="relative flex size-8 items-start justify-center overflow-visible">
      {stacked ? (
        <span
          aria-hidden
          className="absolute top-2 left-1/2 size-6 -translate-x-1/2 rounded-sm bg-gray-900/20 dark:bg-white/20"
        />
      ) : null}
      <WorkspaceIconMark
        workspaceId={workspace.id}
        name={workspace.name}
        hasIcon={workspace.hasIcon}
        iconUpdatedAt={workspace.iconUpdatedAt}
        serverUrl={workspace.serverUrl}
        token={workspace.token}
        className={cn(
          'relative z-10 flex items-center justify-center overflow-hidden rounded-md border border-muted-foreground/10 font-semibold',
          stacked ? 'size-7.5 text-[10.5px]' : 'size-8 text-xs',
          workspace.hasIcon ? '' : 'bg-secondary-foreground text-secondary',
          active && 'ring-2 ring-secondary',
        )}
        initialsClassName={cn(
          'flex items-center justify-center',
          stacked ? 'size-7.5' : 'size-8',
        )}
      />
    </span>
  )
}

export function ModuleRail({
  activeModule = 'chat',
  onSelectModule,
  workspaces = [],
  activeRailKey,
  workspace,
  showWorkspaceSwitcher,
  showSettings = true,
  onSelectWorkspace,
  onAddWorkspace,
}: {
  activeModule?: ShellModule
  onSelectModule: (id: ShellModule) => void
  workspaces?: readonly ModuleRailWorkspace[]
  activeRailKey?: string
  workspace?: ModuleRailWorkspace | null
  showWorkspaceSwitcher?: boolean
  showSettings?: boolean
  onSelectWorkspace?: (railKey: string) => void
  onAddWorkspace?: () => void
}) {
  const { t } = useLocale()
  const { isMobile } = useSidebar()
  const [menuOpen, setMenuOpen] = useState(false)
  const current = workspace ?? workspaces.find((item) => item.railKey === activeRailKey) ?? null
  const currentLabel = current ? workspaceLabel(current, workspaces) : ''
  const switcherCollapsed = !showWorkspaceSwitcher
  const stacked = switcherCollapsed && workspaces.length > 1

  return (
    <Sidebar
      collapsible="none"
      className="w-14! min-w-14! max-w-14! shrink-0 overflow-visible border-r pb-28 md:pb-16"
    >
      <div
        className={cn(
          'grid transition-[grid-template-rows,opacity] duration-300 ease-[cubic-bezier(0.22,1,0.36,1)] motion-reduce:transition-none',
          switcherCollapsed && current
            ? 'grid-rows-[1fr] opacity-100'
            : 'grid-rows-[0fr] opacity-0',
        )}
      >
        <div className="overflow-hidden">
          {current ? (
            <SidebarHeader className="flex h-14 items-center justify-center p-0">
              <SidebarMenu className="items-center">
                <SidebarMenuItem className="flex justify-center">
                  <DropdownMenu open={menuOpen} onOpenChange={setMenuOpen}>
                    <DropdownMenuTrigger asChild>
                      <SidebarMenuButton
                        tooltip={{
                          children: currentLabel,
                          hidden: isMobile || menuOpen,
                        }}
                        aria-label={t('chat.workspaceMenu', { name: current.name })}
                        aria-haspopup="menu"
                        aria-expanded={menuOpen}
                        className="size-8! p-0! hover:bg-transparent data-[state=open]:bg-transparent"
                      >
                        <WorkspaceSwitcherMark
                          workspace={current}
                          stacked={stacked}
                          active={menuOpen}
                        />
                        <span className="sr-only">{currentLabel}</span>
                      </SidebarMenuButton>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent
                      side="right"
                      align="start"
                      sideOffset={8}
                      className="min-w-56"
                    >
                      <DropdownMenuLabel>{currentLabel}</DropdownMenuLabel>
                      <DropdownMenuSeparator />
                      {workspaces.map((item) => {
                        const isActive = item.railKey === current.railKey
                        const label = workspaceLabel(item, workspaces)
                        return (
                          <DropdownMenuItem
                            key={item.railKey}
                            onSelect={() => onSelectWorkspace?.(item.railKey)}
                          >
                            <WorkspaceIconMark
                              workspaceId={item.id}
                              name={item.name}
                              hasIcon={item.hasIcon}
                              iconUpdatedAt={item.iconUpdatedAt}
                              serverUrl={item.serverUrl}
                              token={item.token}
                              className={cn(
                                'size-5 shrink-0 overflow-hidden rounded-sm border border-muted-foreground/10 text-[9px] font-semibold',
                                item.hasIcon
                                  ? ''
                                  : isActive
                                    ? 'bg-primary text-primary-foreground'
                                    : 'bg-muted text-muted-foreground',
                              )}
                              initialsClassName="flex size-5 items-center justify-center"
                            />
                            <span className="min-w-0 flex-1 truncate">{label}</span>
                            {isActive ? (
                              <span className="text-xs text-muted-foreground">
                                {t('profile.active')}
                              </span>
                            ) : null}
                          </DropdownMenuItem>
                        )
                      })}
                      <DropdownMenuSeparator />
                      <DropdownMenuItem onSelect={() => onAddWorkspace?.()}>
                        <PlusIcon />
                        {t('chat.addWorkspace')}
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarHeader>
          ) : null}
        </div>
      </div>
      <SidebarContent className="overflow-x-hidden py-2">
        <SidebarGroup className="px-0 py-0">
          <SidebarGroupContent className="px-0">
            <SidebarMenu className="items-center gap-2.5">
              {SHELL_MODULES.filter(
                (item) => item.id !== 'settings' || showSettings,
              ).map((item) => {
                const Icon = item.icon
                const label = t(item.labelKey)
                const isActive = activeModule === item.id
                return (
                  <SidebarMenuItem key={item.id} className="flex justify-center">
                    <SidebarMenuButton
                      tooltip={{ children: label, hidden: isMobile }}
                      aria-label={label}
                      aria-current={isActive ? 'page' : undefined}
                      isActive={isActive}
                      className="size-8! p-0! hover:bg-transparent data-active:bg-transparent"
                      onClick={() => onSelectModule(item.id)}
                    >
                      <span
                        className={cn(
                          'flex size-8 items-center justify-center rounded-md border border-muted-foreground/10 transition-colors',
                          isActive
                            ? 'bg-primary text-primary-foreground'
                            : 'bg-muted text-muted-foreground hover:bg-accent hover:text-accent-foreground',
                        )}
                      >
                        <Icon strokeWidth={2} className="size-5" />
                      </span>
                      <span className="sr-only">{label}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                )
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  )
}
