import type { ApiPageMeta } from '#shared/types/schema'

type Method = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

export interface ApiOptions {
  query?: Record<string, unknown>
  body?: unknown
  headers?: Record<string, string>
  /** Jangan tampilkan toast otomatis saat error (mis. form yang menampilkan error sendiri). */
  silent?: boolean
}

interface Envelope<T> {
  success: true
  data: T
  meta?: ApiPageMeta
  request_id: string
}

/**
 * Klien API lewat BFF (/api/**). Path ditulis tanpa prefix /api, mis. '/me',
 * '/orders'. Mengurai envelope, melempar ApiError, dan menampilkan toast
 * standar di client kecuali `silent`. Dipanggil di setup (butuh konteks Nuxt).
 */
export function useApi() {
  // Saat SSR, useRequestFetch meneruskan cookie request asli ke handler BFF.
  const fetcher = import.meta.server ? useRequestFetch() : $fetch
  const toast = import.meta.client ? useToast() : null
  const auth = useAuthStore()
  const stepUp = useStepUpStore()

  async function raw<T>(method: Method, path: string, opts: ApiOptions = {}): Promise<Envelope<T>> {
    const send = (stepUpToken?: string) => fetcher<Envelope<T>>(`/api${path}`, {
      method,
      query: opts.query,
      body: opts.body as Record<string, unknown> | undefined,
      headers: stepUpToken ? { ...opts.headers, 'X-Step-Up-Token': stepUpToken } : opts.headers,
    })
    try {
      // 403 STEP_UP_REQUIRED → modal TOTP → request diulang otomatis (hanya di client).
      return import.meta.client
        ? await stepUp.run(send, e => toApiError(e).code === 'STEP_UP_REQUIRED')
        : await send()
    }
    catch (e) {
      const err = toApiError(e)
      if (err.status === 401) auth.clear()
      if (toast && !opts.silent && err.code !== 'VALIDATION_ERROR' && err.code !== 'STEP_UP_REQUIRED') {
        toast.add({ title: err.message, color: 'error', description: err.requestId ? `ID: ${err.requestId}` : undefined })
      }
      throw err
    }
  }

  async function request<T>(method: Method, path: string, opts?: ApiOptions): Promise<T> {
    return (await raw<T>(method, path, opts)).data
  }

  /** Untuk endpoint berhalaman: mengembalikan data + meta. */
  async function page<T>(path: string, opts?: ApiOptions): Promise<{ data: T[], meta: ApiPageMeta }> {
    const env = await raw<T[]>('GET', path, opts)
    return { data: env.data, meta: env.meta as ApiPageMeta }
  }

  return {
    request,
    page,
    get: <T>(path: string, opts?: ApiOptions) => request<T>('GET', path, opts),
    post: <T>(path: string, body?: unknown, opts?: ApiOptions) => request<T>('POST', path, { ...opts, body }),
    put: <T>(path: string, body?: unknown, opts?: ApiOptions) => request<T>('PUT', path, { ...opts, body }),
    patch: <T>(path: string, body?: unknown, opts?: ApiOptions) => request<T>('PATCH', path, { ...opts, body }),
    del: <T>(path: string, opts?: ApiOptions) => request<T>('DELETE', path, opts),
  }
}
