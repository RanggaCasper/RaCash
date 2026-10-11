/**
 * Cookie penanda sesi (non-httpOnly, tanpa isi rahasia) yang diset BFF bersama
 * cookie token. Dipakai client untuk melewati panggilan /me saat jelas belum login.
 * Harus sama dengan SESSION_HINT_COOKIE di server/utils/session.ts.
 */
export const SESSION_HINT = 'rc_session'
