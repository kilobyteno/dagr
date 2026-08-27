import {
  Field,
  FieldDescription,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useServerPublicConfig } from '@/hooks/use-server-public-config'
import { useLocale } from '@/lib/i18n'
import {
  CLOUD_SERVER_URL,
  resolveServerUrl,
  type ServerHostMode,
  type StoredServerHost,
} from '@/lib/server-host'

export function ServerHostPicker({
  value,
  onChange,
  serverInputId,
}: {
  value: StoredServerHost
  onChange: (next: StoredServerHost) => void
  serverInputId: string
}) {
  const { t } = useLocale()
  const serverUrl = resolveServerUrl(value)
  const { unreachable, loading: checkingServer } =
    useServerPublicConfig(serverUrl)

  const setMode = (mode: ServerHostMode) => {
    onChange({ ...value, mode })
  }

  return (
    <Field>
      <FieldLabel>{t('auth.server.label')}</FieldLabel>
      <Tabs
        value={value.mode}
        onValueChange={(next) => {
          if (next === 'cloud' || next === 'selfhosted') setMode(next)
        }}
      >
        <TabsList className="grid h-9 w-full grid-cols-2">
          <TabsTrigger value="cloud">{t('auth.server.cloud')}</TabsTrigger>
          <TabsTrigger value="selfhosted">{t('auth.server.selfHosted')}</TabsTrigger>
        </TabsList>
      </Tabs>

      {value.mode === 'cloud' ? (
        <FieldDescription
          className={unreachable ? 'text-destructive' : undefined}
        >
          {unreachable
            ? t('auth.server.cloudUnreachable')
            : checkingServer
              ? t('auth.server.cloudChecking')
              : t('auth.server.cloudConnects', { url: CLOUD_SERVER_URL })}
        </FieldDescription>
      ) : (
        <>
          <Input
            id={serverInputId}
            type="url"
            inputMode="url"
            autoComplete="url"
            placeholder={t('auth.server.selfHostedPlaceholder')}
            value={value.selfHostedUrl}
            onChange={(event) =>
              onChange({ ...value, selfHostedUrl: event.target.value })
            }
            required
          />
          <FieldDescription
            className={unreachable ? 'text-destructive' : undefined}
          >
            {unreachable
              ? t('auth.server.selfHostedUnreachable')
              : checkingServer
                ? t('auth.server.selfHostedChecking')
                : t('auth.server.selfHostedHint')}
          </FieldDescription>
        </>
      )}
    </Field>
  )
}
