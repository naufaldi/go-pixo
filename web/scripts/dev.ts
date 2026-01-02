type Child = ReturnType<typeof Bun.spawn>

const webRoot = new URL("..", import.meta.url).pathname

const logPrefix = (label: string) => `[dev:${label}]`

const spawn = (label: string, cmd: string[], cwd = webRoot, inherit = true): Child => {
  const proc = Bun.spawn(cmd, {
    cwd,
    stdin: "inherit",
    stdout: inherit ? "inherit" : "pipe",
    stderr: inherit ? "inherit" : "pipe",
    env: process.env,
  })

  if (proc.pid == null) {
    console.error(`${logPrefix(label)} failed to start`)
    process.exitCode = 1
  } else {
    console.log(`${logPrefix(label)} started (pid ${proc.pid})`)
  }

  return proc
}

const kill = (proc: Child, label: string) => {
  try {
    if (!proc.killed) {
      proc.kill()
      console.log(`${logPrefix(label)} stopped`)
    }
  } catch {
    // ignore
  }
}

const pipeStream = async (label: string, stream: ReadableStream | null, capture: string[]) => {
  if (!stream) return
  const reader = stream.getReader()
  const decoder = new TextDecoder()
  while (true) {
    const { value, done } = await reader.read()
    if (done) break
    if (value) {
      const text = decoder.decode(value)
      capture.push(text)
      process.stderr.write(text)
    }
  }
}

const isAlreadyRunningMessage = (text: string) =>
  text.includes("already running") || text.includes("A ReScript build is already running")

const vite = spawn("vite", ["bun", "x", "vite"])

// ReScript can only have one watcher; if your editor already runs it, keep Vite up.
let rescript: Child | null = null
let manageRescriptLifecycle = false
const rescriptOutput: string[] = []

rescript = spawn("rescript", ["bun", "run", "rescript:watch"], webRoot, false)
manageRescriptLifecycle = true
void pipeStream("rescript:stdout", rescript.stdout, rescriptOutput)
void pipeStream("rescript:stderr", rescript.stderr, rescriptOutput)

const shutdown = (code: number) => {
  kill(vite, "vite")
  if (rescript && manageRescriptLifecycle) kill(rescript, "rescript")
  process.exit(code)
}

const onSignal = (signal: NodeJS.Signals) => {
  console.log(`${logPrefix("signal")} ${signal}`)
  shutdown(0)
}

process.on("SIGINT", onSignal)
process.on("SIGTERM", onSignal)

const waitForExit = async (label: string, proc: Child) => {
  const exitCode = await proc.exited
  if (exitCode === 0) return
  console.error(`${logPrefix(label)} exited with code ${exitCode}`)
  shutdown(typeof exitCode === "number" ? exitCode : 1)
}

const waitRescript = async () => {
  if (!rescript) return
  const exitCode = await rescript.exited

  if (exitCode === 0) return

  const combined = rescriptOutput.join("")
  if (isAlreadyRunningMessage(combined)) {
    console.warn(`${logPrefix("rescript")} watcher already running elsewhere; keeping Vite up`)
    manageRescriptLifecycle = false
    return
  }

  console.error(`${logPrefix("rescript")} exited with code ${exitCode}`)
  shutdown(typeof exitCode === "number" ? exitCode : 1)
}

await Promise.race([waitRescript(), waitForExit("vite", vite)])
