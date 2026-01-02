open React
open ReactDOM

switch querySelector("#root") {
| Some(element) =>
  let root = ReactDOM.Client.createRoot(element)
  ReactDOM.Client.Root.render(root, <App />)
| None => Js.Console.error("Root element not found")
}
