let serverEndpoint = "http://127.0.0.1:7244/ingest/3ab5e3cf-eef6-4cfd-8f53-a31c9850c6a9"
let sessionId = "debug-session"

let info = (~hypothesisId, ~location, ~message, ~data=?) => {
  let payload = {
    "sessionId": sessionId,
    "hypothesisId": hypothesisId,
    "location": location,
    "message": message,
    "data": data,
    "timestamp": Date.now(),
  }
  
  let fetchWithBody: (string, 'a) => unit = %raw(`
    (url, payload) => {
      fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      }).catch(() => {})
    }
  `)
  
  fetchWithBody(serverEndpoint, payload)
}
