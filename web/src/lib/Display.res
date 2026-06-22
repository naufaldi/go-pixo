let formatSize = (bytes: int): string => {
  if bytes >= 1_000_000 {
    let mb = Math.round(Int.toFloat(bytes) /. 1000000.0 *. 10.0) /. 10.0
    Float.toString(mb) ++ " MB"
  } else if bytes >= 1000 {
    let kb = Math.round(Int.toFloat(bytes) /. 1000.0 *. 10.0) /. 10.0
    Float.toString(kb) ++ " KB"
  } else {
    Int.toString(bytes) ++ " bytes"
  }
}

let savingsColor = (percent: float): string => {
  if percent >= 30.0 {
    "text-green-400"
  } else if percent >= 10.0 {
    "text-yellow-400"
  } else {
    "text-gray-400"
  }
}

let isAlreadyOptimized = (original: int, compressed: int): bool => {
  if original <= 0 {
    false
  } else {
    let saved = original - compressed
    let percent = Int.toFloat(saved) /. Int.toFloat(original) *. 100.0
    saved >= 0 && percent <= 0.1
  }
}

let calculateSavings = (original: int, compressed: int): option<(float, string)> => {
  if original <= 0 {
    None
  } else {
    let saved = original - compressed
    let percent = Int.toFloat(saved) /. Int.toFloat(original) *. 100.0
    if percent > 0.0 {
      Some((percent, formatSize(saved)))
    } else {
      None
    }
  }
}

let formatTimeFloat = (milliseconds: float): string => {
  let seconds = milliseconds /. 1000.0
  if seconds >= 60.0 {
    let mins = Int.toFloat(Float.toInt(seconds /. 60.0))
    let secs = Float.toFixed(~digits=1, seconds -. mins *. 60.0)
    `${Float.toFixed(~digits=0, mins)}m ${secs}s`
  } else {
    `${Float.toFixed(~digits=1, seconds)}s`
  }
}

let formatTime = (milliseconds: int): string => {
  formatTimeFloat(Int.toFloat(milliseconds))
}
