import {
  CreditCardIcon,
  GlobeIcon,
  PlugsConnectedIcon,
  SlidersHorizontalIcon,
  UsersIcon,
} from '@phosphor-icons/react'

import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar'
import { useLocale } from '@/lib/i18n'
import { cn } from '@/lib/utils'

export const WORKSPACE_SETTINGS_PAGES = [
  {
    id: 'general',
    icon: SlidersHorizontalIcon,
    labelKey: 'workspace.general',
  },
  {
    id: 'domains',
    icon: GlobeIcon,
    labelKey: 'workspace.domains.nav',
  },
  {
    id: 'people',
    icon: UsersIcon,
    labelKey: 'workspace.people.nav',
  },
  {
    id: 'apps',
    icon: PlugsConnectedIcon,
    labelKey: 'apps.nav',
  },
  {
    id: 'billing',
    icon: CreditCardIcon,
    labelKey: 'workspace.billing.nav',
  },
] as const

export type WorkspaceSettingsPageId =
  (typeof WORKSPACE_SETTINGS_PAGES)[number]['id']

export function canOpenWorkspaceSettings(role: string | undefined) {
  return role === 'owner' || role === 'admin'
}

export function isWorkspaceSettingsPage(
  value: string | undefined,
): value is WorkspaceSettingsPageId {
  return WORKSPACE_SETTINGS_PAGES.some((item) => item.id === value)
}

export function resolveWorkspaceSettingsPage(
  value: string | undefined,
): WorkspaceSettingsPageId {
  return isWorkspaceSettingsPage(value) ? value : 'general'
}

export function workspaceSettingsPageLabel(
  page: WorkspaceSettingsPageId,
  t: ReturnType<typeof useLocale>['t'],
) {
  const item = WORKSPACE_SETTINGS_PAGES.find((entry) => entry.id === page)
  return t(item?.labelKey ?? 'workspace.general')
}

export function WorkspaceSettingsNav({
  activePage = 'general',
  onSelectPage,
  className,
}: {
  activePage?: WorkspaceSettingsPageId
  onSelectPage: (page: WorkspaceSettingsPageId) => void
  className?: string
}) {
  const { t } = useLocale()

  return (
    <Sidebar
      collapsible="none"
      className={cn('hidden w-[16.5rem]! md:flex', className)}
    >
      <SidebarContent className="overflow-hidden">
        <SidebarGroup className="p-0 px-2 py-3">
          <SidebarGroupContent>
            <SidebarMenu>
              {WORKSPACE_SETTINGS_PAGES.map((item) => {
                const Icon = item.icon
                const label = t(item.labelKey)
                const isActive = activePage === item.id
                return (
                  <SidebarMenuItem key={item.id}>
                    <SidebarMenuButton
                      isActive={isActive}
                      onClick={() => onSelectPage(item.id)}
                      className="px-2"
                    >
                      <Icon strokeWidth={2} />
                      <span className="truncate">{label}</span>
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
