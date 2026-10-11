declare module '#app' {
  interface PageMeta {
    /** Permission yang wajib dimiliki untuk membuka halaman (middleware permission.global). */
    permission?: string
  }
}

declare module 'vue' {
  interface ComponentCustomProperties {
    vCan: import('vue').Directive<HTMLElement, string>
  }
}

export {}
