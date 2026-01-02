open React

// Define File type for browser File API
type file

module Web = {
  module File = {
    type t = file
    @get external name: t => string = "name"
  }
}

type state = {
  wasmReady: bool,
  files: array<Web.File.t>,
}

type action =
  | SetWasmReady(bool)
  | AddFiles(array<Web.File.t>)

let reducer = (state, action) => {
  switch action {
  | SetWasmReady(ready) => {...state, wasmReady: ready}
  | AddFiles(newFiles) => {...state, files: Array.concat(state.files, newFiles)}
  }
}

@react.component
let make = () => {
  let (state, dispatch) = React.useReducer(
    reducer,
    {wasmReady: false, files: []},
  )

  React.useEffect0(() => {
    Wasm.initWasm()
      ->Promise.then(_ => {
        dispatch(SetWasmReady(true))
        Promise.resolve()
      })
      ->Promise.catch(_ => {
        dispatch(SetWasmReady(true))
        Promise.resolve()
      })
      ->ignore
    None
  })

  <div className="min-h-screen flex flex-col items-center justify-center p-4">
    <div className="max-w-2xl w-full bg-white rounded-2xl shadow-xl p-8 space-y-8">
      <header className="text-center">
        <h1 className="text-4xl font-black tracking-tight text-slate-900 mb-2">
          {React.string("Go-Pixo")}
        </h1>
        <p className="text-slate-500">
          {React.string("Fast, local, client-side PNG compression powered by Go & WASM.")}
        </p>
      </header>

      <main className="space-y-6">
        <div className="border-2 border-dashed border-slate-200 rounded-xl p-12 text-center hover:border-blue-400 transition-colors cursor-pointer group">
          <p className="text-lg font-medium text-slate-700">
            {React.string("Drop PNG or JPEG files here")}
          </p>
          <p className="text-sm text-slate-400 mt-2">
            {React.string("or paste from clipboard")}
          </p>
        </div>

        {state.files->Array.length > 0
           ? <div className="space-y-2">
               {state.files
                ->Array.map(file =>
                  <div
                    key={Web.File.name(file)}
                    className="p-2 border rounded text-sm">
                    {React.string(Web.File.name(file))}
                  </div>
                )
                ->React.array}
             </div>
           : React.null}

        <div className="text-center text-xs text-slate-500 bg-green-50 p-2 rounded">
          <span className="font-semibold"> {React.string("🔒 Privacy First")} </span>
          {React.string(" - Runs locally on your device. No data sent to servers.")}
        </div>
      </main>

      <footer className="pt-8 border-t border-slate-100 text-center">
        <p className="text-xs text-slate-400">
          {React.string("Inspired by ")}
          <a
            href="https://github.com/leerob/pixo"
            className="underline hover:text-slate-600">
            {React.string("pixo")}
          </a>
          {React.string(". Built with Go, WASM, React, ReScript, and Tailwind CSS v4.")}
        </p>
      </footer>
    </div>
  </div>
}
