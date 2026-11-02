open Types

type abortController
type abortSignal
@new external createAbortController: unit => abortController = "AbortController"
@send external abort: abortController => unit = "abort"
@get external signal: abortController => abortSignal = "signal"

@module("../interop/download")
external buildCompressedFilename: (string, string) => string = "buildCompressedFilename"

@module("../interop/download")
external downloadBlob: (string, string, ~signal: abortSignal=?) => promise<unit> = "downloadBlob"

@module("../interop/download")
external downloadAll: (array<queueItem>, ~signal: abortSignal=?) => promise<unit> = "downloadAll"

@module("../interop/download")
external downloadAllAsZip: (array<queueItem>, ~signal: abortSignal=?) => promise<unit> = "downloadAllAsZip"

