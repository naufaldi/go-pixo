open Types

@module("../interop/download")
external downloadBlob: (string, string) => unit = "downloadBlob"

@module("../interop/download")
external downloadAll: array<queueItem> => unit = "downloadAll"

@module("../interop/download")
external downloadAllAsZip: array<queueItem> => promise<unit> = "downloadAllAsZip"

