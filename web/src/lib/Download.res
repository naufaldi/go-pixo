open Types

type abortController
type abortSignal
@new external createAbortController: unit => abortController = "AbortController"
@send external abort: abortController => unit = "abort"
@get external signal: abortController => abortSignal = "signal"

@module("../interop/download")
external downloadBlob: (string, string) => unit = "downloadBlob"

@module("../interop/download")
external downloadAll: array<queueItem> => unit = "downloadAll"

@module("../interop/download")
external downloadAllAsZip: (array<queueItem>, ~signal: abortSignal=?) => promise<unit> = "downloadAllAsZip"

