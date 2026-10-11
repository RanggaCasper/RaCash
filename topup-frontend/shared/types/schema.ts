// Helper tipe di atas hasil generate openapi-typescript (api.d.ts).
// Jangan menulis tipe respons API secara manual (RULES §6.2).
import type { components } from './api'

export type Schemas = components['schemas']

/** Schema<'Me'> → tipe objek Me dari openapi.yaml. */
export type Schema<K extends keyof Schemas> = Schemas[K]

export type ApiPageMeta = Schemas['PageMeta']
export type ErrorCode = Schemas['ErrorCode']
