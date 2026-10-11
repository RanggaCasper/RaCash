import type { Schema } from '#shared/types/schema'

export type LoginOutcome =
  | { status: 'ok' }
  | { status: 'two_factor', challengeToken: string }
  | { status: 'two_factor_setup' }

/** Status login & aksi auth. Token dikelola BFF; client hanya melihat profil & permission. */
export function useAuth() {
  const auth = useAuthStore()
  const perms = usePermissionsStore()
  const api = useApi()

  const user = computed(() => auth.user)
  const loggedIn = computed(() => auth.user !== null || auth.setupRequired)

  /** Memuat profil + permission. false bila belum login/sesi habis. */
  async function fetchSession(): Promise<boolean> {
    try {
      const [me, p] = await Promise.all([
        api.get<Schema<'Me'>>('/me', { silent: true }),
        api.get<Schema<'MyPermissions'>>('/me/permissions', { silent: true }),
      ])
      auth.setUser(me)
      perms.set(p)
      return true
    }
    catch (e) {
      // Token terbatas (staf internal belum punya 2FA): sesi ada, tetapi profil ditolak.
      if (isSetupRequired(e)) {
        auth.setSetupRequired()
        perms.clear()
        return true
      }
      auth.clear()
      perms.clear()
      return false
    }
  }

  async function handleLogin(result: Schema<'LoginResult'>): Promise<LoginOutcome> {
    if (result.two_factor_required && result.challenge_token) {
      return { status: 'two_factor', challengeToken: result.challenge_token }
    }
    if (result.two_factor_setup_required) {
      auth.setSetupRequired()
      return { status: 'two_factor_setup' }
    }
    await fetchSession()
    return { status: 'ok' }
  }

  async function login(email: string, password: string, captchaToken?: string): Promise<LoginOutcome> {
    const result = await api.post<Schema<'LoginResult'>>('/auth/login', { email, password }, {
      silent: true,
      headers: captchaToken ? { 'X-Captcha-Token': captchaToken } : undefined,
    })
    return handleLogin(result)
  }

  async function verifyTwoFactor(challengeToken: string, code: string): Promise<LoginOutcome> {
    const result = await api.post<Schema<'LoginResult'>>('/auth/login/2fa', { challenge_token: challengeToken, code }, { silent: true })
    return handleLogin(result)
  }

  async function logout() {
    try {
      await api.post('/auth/logout', undefined, { silent: true })
    }
    finally {
      auth.clear()
      perms.clear()
      await navigateTo('/masuk')
    }
  }

  return { user, loggedIn, fetchSession, login, verifyTwoFactor, logout }
}
