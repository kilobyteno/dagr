import {
  ArrowsClockwiseIcon,
  ImageIcon,
  PaletteIcon,
  ShieldCheckIcon,
  TranslateIcon,
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

export const APP_SETTINGS_PAGES = [
  {
    id: 'appearance',
    icon: PaletteIcon,
    labelKey: 'settings.appearance.title',
  },
  {
    id: 'language',
    icon: TranslateIcon,
    labelKey: 'settings.language.title',
  },
  {
    id: 'media',
    icon: ImageIcon,
    labelKey: 'settings.media.title',
  },
  {
    id: 'updates',
    icon: ArrowsClockwiseIcon,
    labelKey: 'settings.updates.title',
  },
  {
    id: 'trusted-domains',
    icon: ShieldCheckIcon,
    labelKey: 'settings.trustedDomains.nav',
  },
] as const

export type AppSettingsPageId = (typeof APP_SETTINGS_PAGES)[number]['id']

export function isAppSettingsPage(
  value: string | undefined,
): value is AppSettingsPageId {
  return APP_SETTINGS_PAGES.some((item) => item.id === value)
}

export function resolveAppSettingsPage(
  value: string | undefined,
): AppSettingsPageId {
  return isAppSettingsPage(value) ? value : 'appearance'
}

export function appSettingsPageLabel(
  page: AppSettingsPageId,
  t: ReturnType<typeof useLocale>['t'],
) {
  const item = APP_SETTINGS_PAGES.find((entry) => entry.id === page)
  return t(item?.labelKey ?? 'settings.appearance.title')
}

export function AppSettingsNav({
  activePage = 'appearance',
  onSelectPage,
  className,
}: {
  activePage?: AppSettingsPageId
  onSelectPage: (page: AppSettingsPageId) => void
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
              {APP_SETTINGS_PAGES.map((item) => {
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
