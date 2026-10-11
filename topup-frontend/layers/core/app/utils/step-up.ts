// Koordinator step-up 2FA (RBAC-08): saat API menjawab 403 STEP_UP_REQUIRED, minta
// kode TOTP ke user (lewat modal), tukar dengan step_up_token, lalu ulangi request.
// Fungsi murni tanpa Vue/Nuxt agar bisa di-unit-test.

export interface StepUpGrant {
  token: string
  /** RFC 3339 UTC, dari POST /auth/step-up. */
  expiresAt: string
}

/** Dilempar prompt bila user menutup modal tanpa memasukkan kode. */
export class StepUpCancelled extends Error {
  constructor() {
    super('Verifikasi 2FA dibatalkan.')
    this.name = 'StepUpCancelled'
  }
}

/** Menampilkan modal TOTP. `retry` true bila token sebelumnya baru saja ditolak server. */
export type StepUpPrompt = (opts: { retry: boolean }) => Promise<StepUpGrant>

/** Token dianggap habis sedikit sebelum kedaluwarsa agar tidak dikirim saat hampir mati. */
const SAFETY_MARGIN_MS = 5_000

export class StepUpCoordinator {
  private grant: StepUpGrant | null = null
  private inflight: Promise<string> | null = null

  constructor(
    private readonly prompt: StepUpPrompt,
    private readonly now: () => number = Date.now,
  ) {}

  /** Token tersimpan yang masih berlaku, atau null. */
  cached(): string | null {
    if (!this.grant) return null
    const exp = Date.parse(this.grant.expiresAt)
    if (Number.isNaN(exp) || exp - this.now() <= SAFETY_MARGIN_MS) {
      this.grant = null
      return null
    }
    return this.grant.token
  }

  invalidate() {
    this.grant = null
  }

  /**
   * Token step-up: dari cache bila masih berlaku, kalau tidak minta ke user.
   * Beberapa request yang butuh token bersamaan berbagi satu modal.
   */
  token(opts: { fresh?: boolean } = {}): Promise<string> {
    if (opts.fresh) this.invalidate()
    const hit = this.cached()
    if (hit) return Promise.resolve(hit)
    this.inflight ??= this.prompt({ retry: opts.fresh === true })
      .then((g) => {
        this.grant = g
        return g.token
      })
      .finally(() => {
        this.inflight = null
      })
    return this.inflight
  }

  /**
   * Menjalankan `run`. Bila gagal STEP_UP_REQUIRED, minta token lalu ulangi sekali;
   * bila token itu pun ditolak (mis. milik sesi lain), minta token baru sekali lagi.
   * Dibatalkan user → error STEP_UP_REQUIRED yang asli dilempar ulang.
   */
  async run<T>(
    run: (stepUpToken?: string) => Promise<T>,
    isStepUpRequired: (e: unknown) => boolean,
  ): Promise<T> {
    try {
      return await run()
    }
    catch (first) {
      if (!isStepUpRequired(first)) throw first
      try {
        return await run(await this.token())
      }
      catch (second) {
        if (second instanceof StepUpCancelled) throw first
        if (!isStepUpRequired(second)) throw second
        try {
          return await run(await this.token({ fresh: true }))
        }
        catch (third) {
          if (third instanceof StepUpCancelled) throw second
          throw third
        }
      }
    }
  }
}
