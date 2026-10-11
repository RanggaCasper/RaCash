import type { ErrorCode } from '#shared/types/schema'

export interface ApiErrorDetail {
  field?: string
  code?: string
  message?: string
  required_permission?: string
}

/** Error terstruktur dari envelope gagal (API_CONTRACT §1.3). */
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: ErrorCode | 'NETWORK_ERROR',
    message: string,
    public readonly details: ApiErrorDetail[] = [],
    public readonly requestId = '',
  ) {
    super(message)
    this.name = 'ApiError'
  }

  /** Detail validasi per field: { "target.user_id": "Wajib diisi." }. */
  fieldErrors(): Record<string, string> {
    const out: Record<string, string> = {}
    for (const d of this.details) if (d.field && d.message) out[d.field] = d.message
    return out
  }
}

/** 403 dari token terbatas: user internal wajib mengaktifkan 2FA (API_CONTRACT §4.7). */
export function isSetupRequired(e: unknown): boolean {
  const err = toApiError(e)
  return err.code === 'PERMISSION_DENIED' && err.details.some(d => d.code === 'two_factor_setup_required')
}

interface FetchLikeError {
  status?: number
  statusCode?: number
  data?: unknown
}

/** Mengubah error $fetch (ofetch) menjadi ApiError. */
export function toApiError(e: unknown): ApiError {
  if (e instanceof ApiError) return e
  const fe = (e ?? {}) as FetchLikeError
  const status = fe.status ?? fe.statusCode ?? 0
  const body = fe.data as { error?: { code?: ErrorCode, message?: string, details?: ApiErrorDetail[] }, request_id?: string } | undefined
  if (body?.error?.code) {
    return new ApiError(status, body.error.code, body.error.message ?? '', body.error.details ?? [], body.request_id ?? '')
  }
  return new ApiError(status, 'NETWORK_ERROR', 'Tidak dapat terhubung ke server. Periksa koneksi Anda.')
}
