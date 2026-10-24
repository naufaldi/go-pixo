open React
open AppState

type pool
type compressionResult = {compressedBytes: array<int>, outputFormat: string}
@new @module("./workerPool.ts") external createPool: unit => pool = "WorkerPool"
@get external capacity: pool => int = "capacity"
@send external terminatePool: pool => unit = "terminate"
@send external cancelTask: (pool, string) => unit = "cancel"
@send external submit: (pool, WorkerMessages.compressMessage, (string, int, bool, option<int>) => unit) => Promise.t<compressionResult> = "submit"
@module("./interop/clipboard.ts") external filesFromPasteEvent: 'a => array<Types.Web.File.t> = "filesFromPasteEvent"
let errorMessage = error => error->JsExn.fromException->Option.flatMap(JsExn.message)->Option.getOr("Failed to process image")
let createBlobUrl: (string, array<int>) => string = %raw("(mime, bytes) => URL.createObjectURL(new Blob([bytes], {type: mime}))")
let now: unit => float = %raw("() => performance.now()")
let selectedFiles: ReactEvent.Form.t => array<Types.Web.File.t> = %raw("e => Array.from(e.target.files || [])")
let clearFileInput: ReactEvent.Form.t => unit = %raw("e => { e.target.value = '' }")
let addPasteListener: ('a => unit) => unit = %raw("handler => window.addEventListener('paste', handler)")
let removePasteListener: ('a => unit) => unit = %raw("handler => window.removeEventListener('paste', handler)")
let taskKey = (item: Types.queueItem) => item.id ++ ":" ++ Int.toString(item.generation)

@react.component
let make = () => {
  let (state, dispatch) = React.useReducer(reducer, initialState)
  let stateRef = React.useRef(state)
  let poolRef = React.useRef(None: option<pool>)
  let mountedRef = React.useRef(true)
  let epochRef = React.useRef(0)
  let activeRef = React.useRef([]: array<string>)
  let removedRef = React.useRef([]: array<string>)
  let pendingDownloadRef = React.useRef(None: option<string>)
  let zipExportRef = React.useRef(None: option<Download.abortController>)
  let cancelZipExport = () => {
    let previous = zipExportRef.current
    // Invalidate ownership before abort can settle any old callbacks.
    zipExportRef.current = None
    previous->Option.forEach(Download.abort)
    if previous->Option.isSome && mountedRef.current { dispatch(SetExportingZip(false)) }
  }
  // Async callbacks read the latest committed intent; work snapshots settings at start.
  stateRef.current = state

  React.useEffect0(() => {
    mountedRef.current = true
    poolRef.current = Some(createPool())
    Some(() => {
      mountedRef.current = false
      cancelZipExport()
      epochRef.current = epochRef.current + 1
      poolRef.current->Option.forEach(terminatePool)
      stateRef.current.items->Array.forEach(revokeItemUrls)
    })
  })

  let isCurrent = (item: Types.queueItem, epoch: int) =>
    mountedRef.current && epochRef.current == epoch &&
    !(removedRef.current->Array.includes(item.id)) &&
    stateRef.current.items->Array.some(current => current.id == item.id && current.generation == item.generation)

  let processItem = (item: Types.queueItem, settingsState: Types.appState, workerPool: pool) => {
    let epoch = epochRef.current
    let key = taskKey(item)
    let started = now()
    dispatch(UpdateTask(item.id, item.generation, current => {...current, status: Types.Decoding}))
    dispatch(SetTaskProgress(item.id, item.generation, Some(Progress.seedProgress(item.id, item.originalBytes, started))))
    ImageDecode.decodeFile(item.file)
    ->Promise.then(result => {
      if !isCurrent(item, epoch) {
        BlobUrl.revokeObjectURL(result.previewUrl)
        Promise.resolve()
      } else {
        dispatch(UpdateTask(item.id, item.generation, current => {
          ...current, status: Types.Compressing, originalUrl: Some(result.previewUrl),
          width: Some(result.width), height: Some(result.height),
          kind: Types.fileKindFromMime(if result.inputFormat == "svg" { "image/svg+xml" } else { "image/" ++ result.inputFormat }, ""),
        }))
        let settings = CompressionSettings.settingsForState(settingsState)
        let message = WorkerMessages.buildCompressMessage(
          key, result.pixels, result.width, result.height, result.colorType,
          result.inputFormat,
          Types.outputFormatToChoice(settingsState.outputFormat),
          settings.presetInt, !settingsState.lossless, settings.maxColors,
          settings.dithering, settings.ditherStrength, settings.qualityTarget,
          settings.zopfliIterations, settings.progressive, settings.trellis,
          settings.subsampling, settings.optimizeHuffman, result.originalFileBytes,
          settingsState.targetWidth, settingsState.targetHeight,
        )
        submit(workerPool, message, (phase, percent, predictable, phaseTarget) => {
          if isCurrent(item, epoch) {
            let previous = stateRef.current.activeCompressions
              ->Array.find(((id, _)) => id == item.id)->Option.map(((_, p)) => p)
            let progress = Progress.applyWorkerUpdate(previous, phase, percent, predictable, phaseTarget->Option.getOr(percent), now())
            dispatch(SetTaskProgress(item.id, item.generation, Some({...progress, itemId: item.id, fileSize: item.originalBytes, startTime: started})))
          }
        })
        ->Promise.then(result => {
          if isCurrent(item, epoch) {
            let mime = CompressionSettings.mimeForFormat(result.outputFormat)
            let extension = CompressionSettings.extensionForFormat(result.outputFormat)
            let url = createBlobUrl(mime, result.compressedBytes)
            dispatch(UpdateTask(item.id, item.generation, current => {
              ...current, status: Types.Done, compressedUrl: Some(url),
              compressedBytes: Some(result.compressedBytes->Array.length),
              compressedMime: Some(mime), compressedExtension: Some(extension),
              compressionTime: Some(Int.fromFloat(now() -. started)),
            }))
            if pendingDownloadRef.current == Some(item.id) {
              pendingDownloadRef.current = None
              Download.downloadBlob(url, CompressionSettings.buildCompressedFilename(Types.Web.File.name(item.file), extension))
            }
          }
          Promise.resolve()
        })
      }
    })
    ->Promise.catch(error => {
      if isCurrent(item, epoch) {
        dispatch(UpdateTask(item.id, item.generation, current => {...current, status: Types.Error(errorMessage(error))}))
        if pendingDownloadRef.current == Some(item.id) { pendingDownloadRef.current = None }
      }
      Promise.resolve()
    })
    ->Promise.then(_ => {
      activeRef.current = activeRef.current->Array.filter(active => active != key)
      if mountedRef.current {
        dispatch(SetTaskProgress(item.id, item.generation, None))
        // A stale decode can release capacity without changing a current item.
        dispatch(SetProcessingAll(activeRef.current->Array.length > 0))
      }
      Promise.resolve()
    })->ignore
  }

  React.useEffect1(() => {
    switch poolRef.current {
    | Some(workerPool) =>
      state.items->Array.forEach(item => {
        let key = taskKey(item)
        if item.status == Types.Pending && activeRef.current->Array.length < capacity(workerPool) && !(activeRef.current->Array.includes(key)) {
          // Reserve before decoding, so re-renders cannot start this task twice.
          activeRef.current = activeRef.current->Array.concat([key])
          processItem(item, state, workerPool)
        }
      })
    | None => ()
    }
    None
  }, [state])

  let handleFileSelect = files => {
    if files->Array.length > 0 {
      cancelZipExport()
      dispatch(AddItems(files->Array.map(createQueueItem)))
    }
  }
  let handleDragEnter = e => { ReactEvent.Mouse.preventDefault(e); dispatch(SetDragActive(true)) }
  let handleDragOver = e => ReactEvent.Mouse.preventDefault(e)
  let handleDragLeave = e => { ReactEvent.Mouse.preventDefault(e); dispatch(SetDragActive(false)) }
  let handleDrop = e => {
    ReactEvent.Mouse.preventDefault(e)
    dispatch(SetDragActive(false))
    let files: array<Types.Web.File.t> = %raw("Array.from(e.nativeEvent.dataTransfer?.files || [])")
    handleFileSelect(files)
  }
  React.useEffect0(() => {
    let handlePaste = e => {
      let files = filesFromPasteEvent(e)
      if files->Array.length > 0 { handleFileSelect(files) }
    }
    addPasteListener(handlePaste)
    Some(() => removePasteListener(handlePaste))
  })

  let resetPool = () => {
    epochRef.current = epochRef.current + 1
    poolRef.current->Option.forEach(terminatePool)
    poolRef.current = Some(createPool())
  }
  let applySettingChange = update => {
    cancelZipExport()
    pendingDownloadRef.current = None
    resetPool()
    update()
    dispatch(RequeueProcessedItemsForSettings)
  }
  let handleRemove = id => {
    cancelZipExport()
    removedRef.current = removedRef.current->Array.concat([id])
    stateRef.current.items->Array.find(item => item.id == id)->Option.forEach(item =>
      poolRef.current->Option.forEach(workerPool => cancelTask(workerPool, taskKey(item))))
    if pendingDownloadRef.current == Some(id) { pendingDownloadRef.current = None }
    dispatch(RemoveItem(id))
  }
  let handleClear = () => { cancelZipExport(); resetPool(); pendingDownloadRef.current = None; dispatch(ClearAll) }
  let selectedItem = state.items->Array.find(item => Some(item.id) == state.selectedId)
  let hasActive = state.items->Array.some(item => switch item.status {
    | Types.Pending | Types.Decoding | Types.Compressing => true
    | Types.Done | Types.Error(_) => false
  })
  let completedCount = state.items->Array.filter(item => item.status == Types.Done)->Array.length
  let appliedOptimizations = if completedCount == 0 { None } else {
    let preset = switch state.preset {
      | Types.Ultra => "Ultra" | Types.Smaller => "Smaller"
      | Types.Balanced => "Balanced" | Types.Faster => "Faster"
    }
    let base = [preset ++ " preset", if state.lossless { "Lossless mode" } else { "Lossy compression allowed" }]
    let withFormat = if state.outputFormat == Types.AutoSmallest {
      base->Array.concat(["Auto chose the smallest tested result"])
    } else { base }
    let resize = switch (state.targetWidth, state.targetHeight) {
      | (Some(width), _) => "Target width " ++ Int.toString(width) ++ " px, kept each image's aspect ratio"
      | (None, Some(height)) => "Target height " ++ Int.toString(height) ++ " px, kept each image's aspect ratio"
      | (None, None) => "Kept original dimensions"
    }
    Some(withFormat->Array.concat([resize]))
  }
  let errorCount = state.items->Array.filter(item => switch item.status { | Types.Error(_) => true | Types.Pending | Types.Decoding | Types.Compressing | Types.Done => false })->Array.length
  let savedBytes = state.items->Array.reduce(0, (sum, item) => switch (item.status, item.compressedBytes) {
    | (Types.Done, Some(bytes)) => sum + item.originalBytes - bytes
    | (Types.Done, None) | (Types.Pending | Types.Decoding | Types.Compressing | Types.Error(_), _) => sum
  })
  let outputFormatText = switch selectedItem {
    | Some(item) => switch item.compressedExtension {
      | Some(".jpg") => "JPEG" | Some(".webp") => "WebP" | Some(".avif") => "AVIF" | Some(".png") => "PNG"
      | _ => if state.outputFormat == Types.AutoSmallest { "Auto" } else { CompressionSettings.resolveForItem(item.kind, state.outputFormat)->String.toUpperCase }
      }
    | None => if state.outputFormat == Types.AutoSmallest { "Auto" } else { "Same as input" }
  }
  let handleDownload = () => selectedItem->Option.forEach(item => {
    switch (item.status, item.compressedUrl, item.compressedExtension) {
    | (Types.Done, Some(url), Some(extension)) =>
      Download.downloadBlob(url, CompressionSettings.buildCompressedFilename(Types.Web.File.name(item.file), extension))
    | (Types.Done, _, _) | (Types.Pending | Types.Decoding | Types.Compressing | Types.Error(_), _, _) => ()
    }
  })
  let handleDownloadFormat = fmt => selectedItem->Option.forEach(item => {
    let extension = CompressionSettings.extensionForFormat(CompressionSettings.resolveForItem(item.kind, fmt))
    if item.status == Types.Done && item.compressedExtension == Some(extension) { handleDownload() }
    else {
      applySettingChange(() => dispatch(SetOutputFormat(fmt)))
      pendingDownloadRef.current = Some(item.id)
    }
  })
  let handleDownloadAll = () => {
    if !hasActive && completedCount > 0 { Download.downloadAll(state.items) }
  }
  let handleDownloadZip = () => {
    if !hasActive && completedCount > 0 && zipExportRef.current->Option.isNone {
      let controller = Download.createAbortController()
      // Reserve synchronously; React may batch multiple clicks before rendering.
      zipExportRef.current = Some(controller)
      dispatch(SetDownloadError(None))
      dispatch(SetExportingZip(true))
      let ownsExport = () => mountedRef.current && zipExportRef.current->Option.map(current => current === controller)->Option.getOr(false)
      Download.downloadAllAsZip(state.items, ~signal=Download.signal(controller))
      ->Promise.catch(error => {
        if ownsExport() { dispatch(SetDownloadError(Some(errorMessage(error)))) }
        Promise.resolve()
      })
      ->Promise.then(_ => {
        if ownsExport() {
          zipExportRef.current = None
          dispatch(SetExportingZip(false))
        }
        Promise.resolve()
      })->ignore
    }
  }
  let clampDimension = value => Math.max(1.0, Math.min(8000.0, Int.toFloat(value)))->Int.fromFloat
  // Display the selected image's paired dimension, but send only the chosen
  // axis to the engine. Every image derives its own aspect ratio there.
  let selectedRatio = selectedItem->Option.flatMap(item => switch (item.width, item.height) {
    | (Some(w), Some(h)) when w > 0 && h > 0 => Some(Int.toFloat(w) /. Int.toFloat(h))
    | _ => None
  })
  let displayWidth = switch (state.targetWidth, state.targetHeight, selectedRatio) {
    | (None, Some(h), Some(ratio)) => Some(Int.fromFloat(Math.max(1.0, Math.round(Int.toFloat(h) *. ratio))))
    | _ => state.targetWidth
  }
  let displayHeight = switch (state.targetWidth, state.targetHeight, selectedRatio) {
    | (Some(w), None, Some(ratio)) => Some(Int.fromFloat(Math.max(1.0, Math.round(Int.toFloat(w) /. ratio))))
    | _ => state.targetHeight
  }
  let handleTargetWidthChange = width => applySettingChange(() => {
    dispatch(SetTargetWidth(width->Option.map(clampDimension)))
    dispatch(SetTargetHeight(None))
  })
  let handleTargetHeightChange = height => applySettingChange(() => {
    dispatch(SetTargetHeight(height->Option.map(clampDimension)))
    dispatch(SetTargetWidth(None))
  })
  let handleCompressAll = () => {
    if !hasActive { applySettingChange(() => ()) }
  }

  <div className="min-h-screen flex flex-col bg-neutral-950 text-neutral-100 pb-72 sm:pb-48 lg:pb-36">
    <header className="pt-8 pb-6 px-4 sm:px-6 text-center">
      <h1 className="text-4xl font-black tracking-tight text-neutral-100 mb-2">{React.string("Go-Pixo")}</h1>
      <p className="text-neutral-400">{React.string("Private batch image compression. PNG and JPEG powered by Go & WASM.")}</p>
      <div className="mt-3"><OfflineStatus /><a href="/licenses/third-party.txt" className="mt-1 inline-block text-xs text-neutral-500 underline hover:text-neutral-300">{React.string("Open-source licenses")}</a></div>
      <div className="flex flex-wrap items-center justify-center gap-3 mt-4 text-xs text-emerald-400">
        <span>{React.string("Runs locally on your device")}</span>
        <span>{React.string("No data sent to servers")}</span>
      </div>
      <p className="mt-3 text-xs text-neutral-500">{React.string("PNG, JPEG, WebP, AVIF, HEIC, TIFF, GIF, BMP and SVG inputs. Still images only; up to 24 MP and 64 MiB.")}</p>
    </header>
    <main className="flex-1 w-full max-w-6xl mx-auto px-4 sm:px-6 pb-6">
      {switch selectedItem {
      | Some(item) =>
        let itemProgress = state.activeCompressions->Array.find(((id, _)) => id == item.id)->Option.map(((_, p)) => p)
        switch item.status {
        | Types.Error(message) =>
          <div role="alert" className="my-8 rounded-lg border border-red-800 bg-red-950/20 p-6 text-red-300">
            <p>{React.string(Types.Web.File.name(item.file) ++ ": " ++ message)}</p>
            <button onClick={_ => handleRemove(item.id)} className="mt-3 text-sm underline">{React.string("Remove image")}</button>
          </div>
        | Types.Pending | Types.Decoding | Types.Compressing | Types.Done => <CompareView originalUrl={item.originalUrl} compressedUrl={item.compressedUrl}
            originalBytes={item.originalBytes} compressedBytes={item.compressedBytes}
            width={item.width} height={item.height} compressionTime={item.compressionTime}
            compressionProgress={itemProgress} onRemove={() => handleRemove(item.id)} />
        }
      | None => <Dropzone dragActive={state.dragActive} onDragEnter=handleDragEnter onDragOver=handleDragOver
          onDragLeave=handleDragLeave onDrop=handleDrop onFileSelect=handleFileSelect />
      }}
      {state.items->Array.length > 0 ? <div className="mt-6 space-y-3">
        <label className="inline-flex cursor-pointer rounded border border-neutral-700 px-4 py-2 text-sm hover:bg-neutral-900">
          {React.string("Add images")}
          <input type_="file" multiple=true accept=Types.acceptedImageTypes className="sr-only" onChange={e => {
            handleFileSelect(selectedFiles(e))
            clearFileInput(e)
          }} />
        </label>
        <p dataTestId="batch-summary" role="status" className="text-sm text-neutral-400">
          {React.string(Int.toString(completedCount) ++ " completed, " ++ Int.toString(errorCount) ++ " failed, " ++ Int.toString(state.items->Array.length - completedCount - errorCount) ++ " remaining. " ++
            if savedBytes >= 0 { "Saved " ++ Display.formatSize(savedBytes) } else { "Output increased by " ++ Display.formatSize(-savedBytes) })}
        </p>
      </div> : React.null}
      {switch state.downloadError {
      | Some(message) => <p role="alert" className="mt-3 text-sm text-red-400">{React.string("ZIP download failed: " ++ message)}</p>
      | None => React.null
      }}
      <FileQueue items={state.items} selectedId={state.selectedId} compressionProgress=None activeCompressions={state.activeCompressions}
        onSelect={id => dispatch(SelectItem(Some(id)))} onRemove=handleRemove onClearAll=handleClear />
    </main>
    <BottomBar format=outputFormatText outputFormat={state.outputFormat} onOutputFormatChange={fmt => applySettingChange(() => dispatch(SetOutputFormat(fmt)))}
      preset={state.preset} lossless={state.lossless} onPresetChange={preset => applySettingChange(() => dispatch(SetPreset(preset)))}
      onLosslessChange={lossless => applySettingChange(() => { dispatch(SetLossless(lossless)); dispatch(SetQuantization(if lossless { Types.Lossless } else { Types.Colors256 })) })}
      onDownload=handleDownload onDownloadFormat=handleDownloadFormat onDownloadAll=handleDownloadAll onDownloadZip=handleDownloadZip exportingZip={state.exportingZip}
      hasCompletedItems={completedCount > 0} completedCount failedCount=errorCount appliedOptimizations processingAll=hasActive onCompressAll=handleCompressAll
      targetWidth=displayWidth targetHeight=displayHeight onTargetWidthChange=handleTargetWidthChange onTargetHeightChange=handleTargetHeightChange />
  </div>
}
