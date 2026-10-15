// Dedicated ports so e2e runs never collide with a developer's `pnpm dev`
// (5174) and `muninn serve` (7078).
export const API_PORT = Number(process.env.MUNINN_E2E_API_PORT) || 7079
export const WEB_PORT = Number(process.env.MUNINN_E2E_WEB_PORT) || 5175
