import {
  Field,
  FieldDescription,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useServerPublicConfig } from '@/hooks/use-server-public-config'
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
  const serverUrl = resolveServerUrl(value)
  const { unreachable, loading: checkingServer } =
    useServerPublicConfig(serverUrl)

  const setMode = (mode: ServerHostMode) => {
    onChange({ ...value, mode })
  }

  return (
    <Field>
      <FieldLabel>Server</FieldLabel>
      <Tabs
        value={value.mode}
        onValueChange={(next) => {
          if (next === 'cloud' || next === 'selfhosted') setMode(next)
        }}
      >
        <TabsList className="grid h-9 w-full grid-cols-2">
          <TabsTrigger value="cloud">Cloud</TabsTrigger>
          <TabsTrigger value="selfhosted">Self-hosted</TabsTrigger>
        </TabsList>
      </Tabs>

      {value.mode === 'cloud' ? (
        <FieldDescription
          className={unreachable ? 'text-destructive' : undefined}
        >
          {unreachable
            ? 'Could not reach Dagr Cloud. Try again later or use a self-hosted server.'
            : checkingServer
              ? 'Checking Dagr Cloud…'
              : `Connects to ${CLOUD_SERVER_URL}.`}
        </FieldDescription>
      ) : (
        <>
          <Input
            id={serverInputId}
            type="url"
            inputMode="url"
            autoComplete="url"
            placeholder="https://chat.example.com"
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
              ? 'Could not reach this server. Check the URL and that Dagr is running.'
              : checkingServer
                ? 'Checking server…'
                : 'Your self-hosted Dagr API base URL.'}
          </FieldDescription>
        </>
      )}
    </Field>
  )
}
