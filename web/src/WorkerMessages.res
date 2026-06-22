type parsed
type compressMessage

@module("./interop/workerMessages.ts")
external parse: 'a => parsed = "parseWorkerMessage"

@module("./interop/workerMessages.ts")
external buildCompressMessage: (
  string,
  'a,
  int,
  int,
  int,
  string,
  string,
  int,
  bool,
  int,
  bool,
  float,
  int,
  int,
  bool,
  bool,
  string,
  bool,
  'a,
  option<int>,
  option<int>,
) => compressMessage = "buildCompressMessageFromFields"

@get external messageType: parsed => string = "type"
@get external id: parsed => option<string> = "id"
@get external phase: parsed => option<string> = "phase"
@get external progress: parsed => option<int> = "progress"
@get external predictable: parsed => bool = "predictable"
@get external phaseTarget: parsed => option<int> = "phaseTarget"
@get external compressedBytes: parsed => 'a = "compressedBytes"
@get external outputFormat: parsed => string = "outputFormat"
@get external error: parsed => string = "error"
