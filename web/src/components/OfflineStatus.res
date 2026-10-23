type snapshot = {state: string, message: string, updateAvailable: bool}

@module("../offline/client.ts") external getOfflineStatus: unit => snapshot = "getOfflineStatus"
@module("../offline/client.ts")
external subscribeOffline: (snapshot => unit) => (unit => unit) = "subscribeOffline"

@react.component
let make = () => {
  let (status, setStatus) = React.useState(getOfflineStatus)

  React.useEffect0(() => {
    Some(subscribeOffline(next => setStatus(_ => next)))
  })

  <div className="flex flex-col gap-1 text-xs" dataTestId="offline-status" role="status" ariaLive=#polite>
    <span className={status.state == "ready" ? "text-emerald-400" : "text-neutral-400"}>
      {React.string(status.message)}
    </span>
    {status.updateAvailable
      ? <span className="text-neutral-400">
          {React.string("Update ready. Finish your work, then close all Go-Pixo tabs and reopen.")}
        </span>
      : React.null}
  </div>
}
