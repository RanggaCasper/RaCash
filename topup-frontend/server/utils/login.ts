// Penyelesaian login: token dari Go API disimpan ke cookie httpOnly dan
// DIHAPUS dari respons yang dikirim ke browser.
import type { H3Event } from 'h3'

interface LoginData extends Partial<TokenPair> {
  two_factor_required?: boolean
  [k: string]: unknown
}

export async function finishLogin(event: H3Event, res: Response) {
  relayResponseMeta(event, res)
  const body = await readJson(res) as { success?: boolean, data?: LoginData } | null
  if (!res.ok || !body?.data) return body ?? errorEnvelope('INTERNAL_ERROR', 'Terjadi kesalahan pada sistem.')

  const { access_token, refresh_token, access_token_expires_at, refresh_token_expires_at, ...rest } = body.data
  if (access_token && refresh_token) {
    setAuthSession(event, { access_token, refresh_token, access_token_expires_at, refresh_token_expires_at })
  }
  return { ...body, data: rest }
}
