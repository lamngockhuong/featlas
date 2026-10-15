// Starts muninn against a temporary copy of examples/demo (reviews write
// files, so the repository copy is never served) and the Vite dev server in
// front of it. Returns the teardown that stops both and removes the copy.
import { type ChildProcess, execFileSync, spawn } from 'node:child_process'
import { cpSync, mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import { API_PORT, WEB_PORT } from './ports'

const webDir = resolve(import.meta.dirname, '..')
const repoDir = resolve(webDir, '..')

async function waitFor(url: string, proc: ChildProcess, timeoutMs = 90_000): Promise<void> {
  const until = Date.now() + timeoutMs
  while (Date.now() < until) {
    if (proc.exitCode !== null) throw new Error(`${url}: process exited with ${proc.exitCode}`)
    try {
      const res = await fetch(url)
      if (res.ok) return
    } catch {
      // not listening yet
    }
    await new Promise((r) => setTimeout(r, 300))
  }
  throw new Error(`timed out waiting for ${url}`)
}

function stop(proc: ChildProcess | undefined): void {
  if (!proc?.pid || proc.exitCode !== null) return
  try {
    // Negative pid: the whole process group (pnpm/vite spawn children).
    process.kill(-proc.pid, 'SIGTERM')
  } catch {
    proc.kill('SIGTERM')
  }
}

export default async function globalSetup() {
  const tmp = mkdtempSync(join(tmpdir(), 'muninn-e2e-'))
  cpSync(join(repoDir, 'examples', 'demo'), join(tmp, 'demo'), { recursive: true })
  const bin = join(tmp, process.platform === 'win32' ? 'muninn.exe' : 'muninn')
  execFileSync('go', ['build', '-o', bin, './cmd/muninn'], { cwd: repoDir, stdio: 'inherit' })

  const procs: ChildProcess[] = []
  const teardown = async () => {
    for (const p of procs) stop(p)
    rmSync(tmp, { recursive: true, force: true })
  }
  try {
    const api = spawn(
      bin,
      ['serve', '--dir', join(tmp, 'demo', 'muninn'), '--port', String(API_PORT)],
      { stdio: ['ignore', 'inherit', 'inherit'], detached: true },
    )
    procs.push(api)
    await waitFor(`http://127.0.0.1:${API_PORT}/api/project`, api)

    const web = spawn('pnpm', ['exec', 'vite', '--host', '127.0.0.1'], {
      cwd: webDir,
      stdio: ['ignore', 'inherit', 'inherit'],
      detached: true,
      env: {
        ...process.env,
        MUNINN_API: `http://127.0.0.1:${API_PORT}`,
        MUNINN_WEB_PORT: String(WEB_PORT),
      },
    })
    procs.push(web)
    await waitFor(`http://127.0.0.1:${WEB_PORT}/api/project`, web)
  } catch (err) {
    await teardown()
    throw err
  }
  return teardown
}
