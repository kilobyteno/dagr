import { contextBridge, ipcRenderer } from 'electron'

contextBridge.exposeInMainWorld('dagr', {
  platform: process.platform,
  invoke: (channel: string, ...args: unknown[]) => ipcRenderer.invoke(channel, ...args),
})

function domReady(condition: DocumentReadyState[] = ['complete', 'interactive']) {
  return new Promise((resolve) => {
    if (condition.includes(document.readyState)) {
      resolve(true)
    } else {
      document.addEventListener('readystatechange', () => {
        if (condition.includes(document.readyState)) {
          resolve(true)
        }
      })
    }
  })
}

const safeDOM = {
  append(parent: HTMLElement, child: HTMLElement) {
    if (![...parent.children].includes(child)) {
      return parent.appendChild(child)
    }
  },
  remove(parent: HTMLElement, child: HTMLElement) {
    if ([...parent.children].includes(child)) {
      return parent.removeChild(child)
    }
  },
}

function useLoading() {
  const prefersDark =
    typeof window !== 'undefined' &&
    window.matchMedia?.('(prefers-color-scheme: dark)').matches

  const styleContent = `
.app-loading-wrap {
  position: fixed;
  inset: 0;
  z-index: 9;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  background: ${prefersDark ? '#1f1a17' : '#faf7f4'};
  color: ${prefersDark ? '#f5f5f5' : '#171717'};
  font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}
.app-loading-wrap::before {
  content: "";
  position: absolute;
  inset: 0;
  background:
    radial-gradient(90% 70% at 50% 20%, rgba(242, 103, 34, ${prefersDark ? '0.28' : '0.32'}), transparent 58%),
    radial-gradient(80% 60% at 80% 90%, rgba(242, 103, 34, ${prefersDark ? '0.14' : '0.16'}), transparent 55%);
  pointer-events: none;
}
.app-loading-inner {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1.15rem;
  animation: dagr-boot-in 500ms ease-out;
}
.app-loading-mark {
  width: 3.5rem;
  height: 3.5rem;
  display: block;
  object-fit: cover;
  border-radius: 0.375rem;
  animation: dagr-boot-mark 2.4s ease-in-out infinite;
  ${prefersDark ? 'filter: invert(1);' : ''}
}
.app-loading-title {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 600;
  letter-spacing: -0.02em;
  line-height: 1.2;
}
.app-loading-label {
  margin: 0;
  font-size: 0.875rem;
  color: ${prefersDark ? 'rgba(245,245,245,0.65)' : 'rgba(23,23,23,0.55)'};
}
.app-loading-bar {
  margin-top: 0.15rem;
  width: 7rem;
  height: 0.125rem;
  border-radius: 999px;
  overflow: hidden;
  background: ${prefersDark ? 'rgba(255,255,255,0.12)' : 'rgba(0,0,0,0.08)'};
}
.app-loading-bar > span {
  display: block;
  width: 50%;
  height: 100%;
  border-radius: 999px;
  background: #f26722;
  animation: dagr-boot-bar 1.2s ease-in-out infinite;
}
@keyframes dagr-boot-in {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}
@keyframes dagr-boot-mark {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(0.96); }
}
@keyframes dagr-boot-bar {
  0% { transform: translateX(-120%); }
  100% { transform: translateX(240%); }
}
`
  const oStyle = document.createElement('style')
  const oDiv = document.createElement('div')
  oStyle.id = 'app-loading-style'
  oStyle.innerHTML = styleContent
  oDiv.className = 'app-loading-wrap'
  oDiv.setAttribute('role', 'status')
  oDiv.setAttribute('aria-live', 'polite')
  oDiv.setAttribute('aria-busy', 'true')
  const iconSrc = new URL('app-icon.png', window.location.href).href
  oDiv.innerHTML = `
    <div class="app-loading-inner">
      <img class="app-loading-mark" src="${iconSrc}" width="56" height="56" alt="" draggable="false" aria-hidden="true" />
      <p class="app-loading-title">Dagr</p>
      <p class="app-loading-label">Starting Dagr…</p>
      <div class="app-loading-bar" aria-hidden="true"><span></span></div>
    </div>
  `

  return {
    appendLoading() {
      safeDOM.append(document.head, oStyle)
      safeDOM.append(document.body, oDiv)
    },
    removeLoading() {
      safeDOM.remove(document.head, oStyle)
      safeDOM.remove(document.body, oDiv)
    },
  }
}

const { appendLoading, removeLoading } = useLoading()
domReady().then(appendLoading)

window.onmessage = (ev) => {
  if (ev.data?.payload === 'removeLoading') removeLoading()
}

setTimeout(removeLoading, 7999)
