import { describe, expect, it, vi } from 'vitest'
import { StepUpCancelled, StepUpCoordinator, type StepUpGrant } from '../../layers/core/app/utils/step-up'
import { ApiError, isSetupRequired, toApiError } from '../../layers/core/app/utils/api-error'

const isStepUp = (e: unknown) => e instanceof ApiError && e.code === 'STEP_UP_REQUIRED'
const stepUpError = () => new ApiError(403, 'STEP_UP_REQUIRED', 'butuh 2FA')

function grantIn(ms: number, token = 'su_1', now = 1_000_000): StepUpGrant {
  return { token, expiresAt: new Date(now + ms).toISOString() }
}

describe('StepUpCoordinator.run', () => {
  it('request biasa tidak memunculkan prompt', async () => {
    const prompt = vi.fn()
    const c = new StepUpCoordinator(prompt)
    await expect(c.run(async () => 'ok', isStepUp)).resolves.toBe('ok')
    expect(prompt).not.toHaveBeenCalled()
  })

  it('error selain STEP_UP_REQUIRED diteruskan apa adanya', async () => {
    const prompt = vi.fn()
    const c = new StepUpCoordinator(prompt)
    const boom = new ApiError(500, 'INTERNAL_ERROR', 'x')
    await expect(c.run(async () => { throw boom }, isStepUp)).rejects.toBe(boom)
    expect(prompt).not.toHaveBeenCalled()
  })

  it('STEP_UP_REQUIRED → prompt → request diulang dengan token', async () => {
    const prompt = vi.fn(async () => grantIn(300_000, 'su_abc'))
    const c = new StepUpCoordinator(prompt, () => 1_000_000)
    const run = vi.fn(async (t?: string) => {
      if (!t) throw stepUpError()
      return `selesai:${t}`
    })
    await expect(c.run(run, isStepUp)).resolves.toBe('selesai:su_abc')
    expect(prompt).toHaveBeenCalledTimes(1)
    expect(run).toHaveBeenNthCalledWith(1)
    expect(run).toHaveBeenNthCalledWith(2, 'su_abc')
  })

  it('token yang masih berlaku dipakai ulang tanpa prompt kedua', async () => {
    const prompt = vi.fn(async () => grantIn(300_000, 'su_abc'))
    const c = new StepUpCoordinator(prompt, () => 1_000_000)
    const run = async (t?: string) => {
      if (!t) throw stepUpError()
      return t
    }
    await c.run(run, isStepUp)
    await c.run(run, isStepUp)
    expect(prompt).toHaveBeenCalledTimes(1)
  })

  it('token kedaluwarsa tidak dipakai; prompt diminta lagi', async () => {
    let now = 1_000_000
    const prompt = vi.fn(async () => grantIn(300_000, `su_${prompt.mock.calls.length}`, now))
    const c = new StepUpCoordinator(prompt, () => now)
    const run = async (t?: string) => {
      if (!t) throw stepUpError()
      return t
    }
    await c.run(run, isStepUp)
    now += 300_000 // lewat masa berlaku
    await c.run(run, isStepUp)
    expect(prompt).toHaveBeenCalledTimes(2)
  })

  it('token ditolak server (sesi lain) → prompt baru dengan retry=true', async () => {
    const calls: boolean[] = []
    const prompt = vi.fn(async ({ retry }: { retry: boolean }) => {
      calls.push(retry)
      return grantIn(300_000, retry ? 'su_baru' : 'su_lama')
    })
    const c = new StepUpCoordinator(prompt, () => 1_000_000)
    const run = async (t?: string) => {
      if (t !== 'su_baru') throw stepUpError()
      return 'ok'
    }
    await expect(c.run(run, isStepUp)).resolves.toBe('ok')
    expect(calls).toEqual([false, true])
  })

  it('dibatalkan user → error STEP_UP_REQUIRED yang asli', async () => {
    const c = new StepUpCoordinator(async () => { throw new StepUpCancelled() })
    const original = stepUpError()
    await expect(c.run(async () => { throw original }, isStepUp)).rejects.toBe(original)
  })

  it('beberapa request bersamaan berbagi satu prompt', async () => {
    let release!: (g: StepUpGrant) => void
    const prompt = vi.fn(() => new Promise<StepUpGrant>((r) => { release = r }))
    const c = new StepUpCoordinator(prompt, () => 1_000_000)
    const run = async (t?: string) => {
      if (!t) throw stepUpError()
      return t
    }
    const all = Promise.all([c.run(run, isStepUp), c.run(run, isStepUp), c.run(run, isStepUp)])
    await new Promise(r => setTimeout(r, 0))
    release(grantIn(300_000, 'su_sama'))
    await expect(all).resolves.toEqual(['su_sama', 'su_sama', 'su_sama'])
    expect(prompt).toHaveBeenCalledTimes(1)
  })
})

describe('isSetupRequired', () => {
  it('mengenali 403 token terbatas', () => {
    const e = new ApiError(403, 'PERMISSION_DENIED', 'x', [{ code: 'two_factor_setup_required' }])
    expect(isSetupRequired(e)).toBe(true)
  })

  it('PERMISSION_DENIED biasa bukan token terbatas', () => {
    expect(isSetupRequired(new ApiError(403, 'PERMISSION_DENIED', 'x', [{ required_permission: 'orders:read' }]))).toBe(false)
    expect(isSetupRequired(toApiError(new Error('jaringan')))).toBe(false)
  })
})
