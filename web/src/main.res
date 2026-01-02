open React
open ReactDOM

switch querySelector("#root") {
| Some(element) => createRoot(element)->render(<App />)
| None => Js.Console.error("Root element not found")
}
