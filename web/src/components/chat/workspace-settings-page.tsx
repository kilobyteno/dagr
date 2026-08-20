import { WorkspaceBillingSection } from '@/components/chat/workspace-billing-section'
import { WorkspaceDomainsPage } from '@/components/chat/workspace-domains-page'
import { WorkspaceGeneralPage } from '@/components/chat/workspace-general-page'
import { WorkspaceIconMark } from '@/components/chat/workspace-icon'
import { WorkspacePeopleSection } from '@/components/chat/workspace-people-section'
import {
  WORKSPACE_SETTINGS_PAGES,
  workspaceSettingsPageLabel,
  type WorkspaceSettingsPageId,
} from '@/components/chat/workspace-settings-nav'
import { Button } from '@/components/ui/button'
import { ButtonGroup } from '@/components/ui/button-group'
import { ScrollArea } from '@/components/ui/scroll-area'
import { type ApiWorkspace } from '@/lib/api/workspaces'
import { useLocale } from '@/lib/i18n'
import { cn } from '@/lib/utils'

export function WorkspaceSettingsPage({
  page,
  onSelectPage,
  workspace,
  serverUrl,
  token,
  canManage,
  currentUserId,
  onRenamed,
  onIconChanged,
  onDeleted,
  onLeftWorkspace,
  billingRefreshToken = 0,
}: {
  page: WorkspaceSettingsPageId
  onSelectPage: (page: WorkspaceSettingsPageId) => void
  workspace: ApiWorkspace
  serverUrl: string
  token: string
  canManage: boolean
  currentUserId?: string
  onRenamed: (workspace: ApiWorkspace) => void
  onIconChanged?: (workspace: ApiWorkspace) => void
  onDeleted: (workspaceId: string) => void
  onLeftWorkspace?: () => void
  billingRefreshToken?: number
}) {
  const { t } = useLocale()

  if (!canManage) return null

  return (
    <div className="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-background">
      <header className="flex h-14 shrink-0 items-center gap-2 border-b px-3">
        <div className="hidden min-w-0 flex-1 flex-col gap-0.5 md:flex">
          <h1 className="truncate text-sm font-semibold">
            {workspaceSettingsPageLabel(page, t)}
          </h1>
          <p className="truncate text-xs text-muted-foreground">{workspace.name}</p>
        </div>
        <ButtonGroup
          aria-label={t('workspace.settings')}
          className="min-w-0 flex-1 md:hidden [&>button]:flex-1"
        >
          {WORKSPACE_SETTINGS_PAGES.map((item) => {
            const selected = page === item.id
            return (
              <Button
                key={item.id}
                type="button"
                size="sm"
                variant={selected ? 'default' : 'outline'}
                aria-pressed={selected}
                onClick={() => onSelectPage(item.id)}
                className={cn(!selected && 'bg-background')}
              >
                {t(item.labelKey)}
              </Button>
            )
          })}
        </ButtonGroup>
        <WorkspaceIconMark
          workspaceId={workspace.id}
          name={workspace.name}
          hasIcon={workspace.hasIcon}
          iconUpdatedAt={workspace.iconUpdatedAt}
          serverUrl={serverUrl}
          token={token}
          className="size-8 overflow-hidden rounded-md object-cover"
          initialsClassName="flex size-8 items-center justify-center rounded-md bg-primary text-xs font-semibold text-primary-foreground"
        />
      </header>

      <ScrollArea className="min-h-0 flex-1">
        <div className="mx-auto flex w-full max-w-xl flex-col gap-8 p-6">
          {page === 'general' ? (
            <WorkspaceGeneralPage
              workspace={workspace}
              serverUrl={serverUrl}
              token={token}
              canManage={canManage}
              onRenamed={onRenamed}
              onIconChanged={onIconChanged}
              onDeleted={onDeleted}
            />
          ) : null}
          {page === 'domains' ? (
            <WorkspaceDomainsPage
              workspaceId={workspace.id}
              serverUrl={serverUrl}
              token={token}
              canManage={canManage}
            />
          ) : null}
          {page === 'people' ? (
            <WorkspacePeopleSection
              workspaceId={workspace.id}
              workspaceName={workspace.name}
              serverUrl={serverUrl}
              token={token}
              canManage={canManage}
              currentUserId={currentUserId}
              currentUserRole={workspace.role}
              onLeftWorkspace={onLeftWorkspace}
            />
          ) : null}
          {page === 'billing' ? (
            <WorkspaceBillingSection
              workspaceId={workspace.id}
              serverUrl={serverUrl}
              token={token}
              canManage={canManage}
              refreshToken={billingRefreshToken}
            />
          ) : null}
        </div>
      </ScrollArea>
    </div>
  )
}
