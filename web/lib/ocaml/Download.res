open Types

@module("../interop/download")
external downloadBlob: (string, string) => unit = "downloadBlob"

@module("../interop/download")
external downloadAll: array<queueItem> => unit = "downloadAll"

