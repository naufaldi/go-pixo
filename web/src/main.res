open ReactDOM

@module("./main.ts") external initializeOffline: unit => unit = "initializeOffline"

initializeOffline()

switch querySelector("#root") {
| Some(element) =>
  let root = ReactDOM.Client.createRoot(element)
  ReactDOM.Client.Root.render(root, <App />)
| None => Console.error("Root element not found")
}
