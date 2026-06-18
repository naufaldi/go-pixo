open React
open AppState

@send external terminateWorker: 'a => unit = "terminate"

@module("./interop/clipboard.ts")
external filesFromPasteEvent: 'a => array<Types.Web.File.t> = "filesFromPasteEvent"
@react.component
let make = () => {
  let (state, dispatch) = React.useReducer(
    reducer,
    initialState,
  )
  
  let workerRef = React.useRef(Nullable.null)
  let processingRef = React.useRef(false)
  let itemsRef = React.useRef(state.items)
  let stateRef = React.useRef(state)
  let compressionProgressRef = React.useRef(state.compressionProgress)

  React.useEffect1(() => {
    itemsRef.current = state.items
    stateRef.current = state
    None
  }, [state])

  React.useEffect1(() => {
    compressionProgressRef.current = state.compressionProgress
    None
  }, [state.compressionProgress])

  React.useEffect1(() => {
    switch state.compressionProgress {
    | Some(progress) when progress.predictable =>
      let setIntervalFn: (unit => unit, int) => int = %raw("(fn, ms) => setInterval(fn, ms)")
      let clearIntervalFn: int => unit = %raw("id => clearInterval(id)")
      let tick = () => {
        switch compressionProgressRef.current {
        | Some(current) when current.predictable =>
          let now = %raw("performance.now()")
          let advanced = Progress.advancePredicted(current, now)
          if advanced.progress > current.progress {
            dispatch(SetCompressionProgress(Some(advanced)))
          }
        | _ => ()
        }
      }
      let intervalId = setIntervalFn(tick, 100)
      Some(() => clearIntervalFn(intervalId))
    | _ => None
    }
  }, [state.compressionProgress])
  
  React.useEffect0(() => {
    let setOnMessage: ('a, 'b) => unit = %raw("(worker, handler) => { worker.onmessage = handler }")
    let postInit: 'a => unit = %raw("worker => worker.postMessage({ type: 'init' })")
    let logPostingInit: unit => unit = %raw("() => console.debug('[app] posting init')")
    let logWorkerMessage: 'a => unit = %raw("data => console.debug('[app] worker message', data)")
    let logWasmReady: unit => unit = %raw("() => console.debug('[app] wasm ready')")
    let logCompressed: (string, int) => unit = %raw("(id, size) => console.debug('[app] compressed', id, size)")
    let logWorkerError: (string, 'a) => unit = %raw("(id, err) => console.debug('[app] worker error', id, err)")
    let logMissingId: (string, 'a) => unit = %raw("(label, data) => console.error(label, data)")

    // Initialize Web Worker for compression
    let worker = %raw("new Worker(new URL('./worker.ts', import.meta.url), { type: 'module' })");
    workerRef.current = Nullable.make(worker);
    
    let handleWorkerMessage = (_event: {..}) => {
      let data = %raw("_event.data");
      logWorkerMessage(data)
      let message = WorkerMessages.parse(data)
      let msgType = WorkerMessages.messageType(message)
      switch msgType {
      | "ready" =>
        logWasmReady()
        dispatch(SetWasmReady(true))
      | "progress" =>
        let id = WorkerMessages.id(message)
        let phase = WorkerMessages.phase(message)
        let globalProgress = WorkerMessages.progress(message)
        let predictable = WorkerMessages.predictable(message)
        let phaseTarget = WorkerMessages.phaseTarget(message)
        switch (id, phase, globalProgress) {
        | (Some(id), Some(phase), Some(globalProgress)) =>
          let fileSize = switch itemsRef.current->Array.find(item => item.id == id) {
          | Some(item) => item.originalBytes
          | None => 0
          };
          let now = %raw("performance.now()")
          let target = switch phaseTarget {
          | Some(value) => value
          | None => globalProgress
          }
          let updated = Progress.applyWorkerUpdate(
            compressionProgressRef.current,
            phase,
            globalProgress,
            predictable,
            target,
            now,
          )
          dispatch(SetCompressionProgress(Some({
            ...updated,
            itemId: id,
            fileSize,
          })))
        | _ => ()
        }
      | "compressed" =>
        let id = WorkerMessages.id(message)
        switch id {
        | Some(id) =>
          let compressedBytes = WorkerMessages.compressedBytes(message);
          let resolvedFormat = WorkerMessages.outputFormat(message);
          let mimeType = CompressionSettings.mimeForFormat(resolvedFormat);
          let extension = CompressionSettings.extensionForFormat(resolvedFormat);
          let createBlobUrl: (string, 'a) => string = %raw("(mimeType, compressedBytes) => { const blob = new Blob([compressedBytes], { type: mimeType }); return URL.createObjectURL(blob); }")
          let compressedUrl = createBlobUrl(mimeType, compressedBytes);
          let compressedSize = compressedBytes->Array.length;
          logCompressed(id, compressedSize)

          let compressionTime = switch compressionProgressRef.current {
          | Some(progress) when progress.itemId == id =>
            let elapsed = %raw("performance.now()") -. progress.startTime
            Some(Int.fromFloat(elapsed))
          | _ => None
          }

          dispatch(UpdateItem(id, item => {
            ...item,
            status: Types.Done,
            compressedUrl: Some(compressedUrl),
            compressedBytes: Some(compressedSize),
            compressionTime,
            compressedMime: Some(mimeType),
            compressedExtension: Some(extension),
          }))
          dispatch(SetCompressionProgress(None))
        | None =>
          logMissingId("[app] compressed message missing id", data)
        }
      | "error" =>
        let id = WorkerMessages.id(message)
        let errorMsg = WorkerMessages.error(message)
        switch id {
        | Some(id) =>
          logWorkerError(id, errorMsg)
          dispatch(UpdateItem(id, item => {
            ...item,
            status: Types.Error(errorMsg),
          }))
          dispatch(SetCompressionProgress(None))
        | None =>
          logMissingId("Worker error (no id):", errorMsg)
        }
      | _ => ()
      }
    };
    setOnMessage(worker, handleWorkerMessage)
    logPostingInit()
    postInit(worker)
    
    Some(() => {
      switch workerRef.current->Nullable.toOption {
      | Some(w) => terminateWorker(w)
      | None => ()
      }
    })
  })
  
  let handleDragEnter = (e: ReactEvent.Mouse.t) => {
    ReactEvent.Mouse.preventDefault(e)
    dispatch(SetDragActive(true))
  }
  
  let handleDragOver = (e: ReactEvent.Mouse.t) => {
    ReactEvent.Mouse.preventDefault(e)
  }
  
  let handleDragLeave = (e: ReactEvent.Mouse.t) => {
    ReactEvent.Mouse.preventDefault(e)
    dispatch(SetDragActive(false))
  }
  
  let handleDrop = (e: ReactEvent.Mouse.t) => {
    ReactEvent.Mouse.preventDefault(e)
    dispatch(SetDragActive(false))
    let files = %raw("e.nativeEvent.dataTransfer?.files")
    if files->Nullable.isNullable == false {
      let fileArray = %raw("Array.from(files)")
      let items = fileArray->Array.map(createQueueItem)
      dispatch(AddItems(items))
    }
  }
  
  let handleFileSelect = (files: array<Types.Web.File.t>) => {
    let items = files->Array.map(createQueueItem)
    dispatch(AddItems(items))
  }
  
  let inputFormatForKind = (kind: Types.fileKind): string => {
    Types.kindToInputFormat(kind)
  }

  let processItem = (item: Types.queueItem): Promise.t<unit> => {
    switch item.kind {
    | Types.Png | Types.Webp | Types.Jpeg =>
      dispatch(UpdateItem(item.id, item => {...item, status: Types.Decoding}))
      ImageDecode.decodeFile(item.file)
        ->Promise.then(result => {
          dispatch(UpdateItem(item.id, item => {
            ...item,
            status: Types.Compressing,
            originalUrl: Some(result.previewUrl),
            width: Some(result.width),
            height: Some(result.height),
          }))

          let now = %raw("performance.now()")
          dispatch(SetCompressionProgress(Some(
            Progress.seedProgress(item.id, item.originalBytes, now),
          )))

          let currentState = stateRef.current
          let settings = CompressionSettings.settingsForState(currentState)
          let pixels: 'a = %raw("new Uint8Array(result.pixels)")
          let lossy = !currentState.lossless
          let originalFileBytes = %raw("new Uint8Array(result.originalFileBytes)")
          let inputFormat = inputFormatForKind(item.kind)
          let effectiveFormat = CompressionSettings.resolveForItem(item.kind, currentState.outputFormat)
          let postCompress: ('a, WorkerMessages.compressMessage) => unit = %raw(
            "(worker, message) => worker.postMessage(message, [message.pixels.buffer, message.originalFileBytes.buffer].filter(Boolean))"
          )

          switch workerRef.current->Nullable.toOption {
          | Some(worker) =>
            let message = WorkerMessages.buildCompressMessage(
              item.id,
              pixels,
              result.width,
              result.height,
              result.colorType,
              inputFormat,
              effectiveFormat,
              settings.presetInt,
              lossy,
              settings.maxColors,
              settings.dithering,
              settings.ditherStrength,
              settings.qualityTarget,
              settings.zopfliIterations,
              settings.progressive,
              settings.trellis,
              settings.subsampling,
              settings.optimizeHuffman,
              originalFileBytes,
              currentState.targetWidth,
              currentState.targetHeight,
            )
            postCompress(worker, message)
          | None =>
            dispatch(UpdateItem(item.id, item => {
              ...item,
              status: Types.Error("Worker not available"),
            }))
          }

          Promise.resolve()
        })
        ->Promise.catch(_err => {
          let errorMsg = %raw("err.message || 'Failed to process image'")
          dispatch(UpdateItem(item.id, item => {
            ...item,
            status: Types.Error(errorMsg),
          }))
          Promise.resolve()
        })
    | Types.Unknown =>
      dispatch(UpdateItem(item.id, item => {
        ...item,
        status: Types.Error("Unsupported file type"),
      }))
      Promise.resolve()
    }
  }
  
  let processQueue = () => {
    if processingRef.current {
      ()
    } else {
      processingRef.current = true
      let pendingItems = itemsRef.current->Array.filter(item => {
        switch item.status {
        | Types.Pending => true
        | Types.Decoding | Types.Compressing | Types.Done | Types.Error(_) => false
        }
      })
      
      let rec processNext = (index: int): Promise.t<unit> => {
        if index >= pendingItems->Array.length {
          processingRef.current = false
          Promise.resolve()
        } else {
          switch pendingItems->Array.get(index) {
          | Some(item) =>
            processItem(item)
              ->Promise.then(_ => processNext(index + 1))
          | None =>
            processingRef.current = false
            Promise.resolve()
          }
        }
      }
      
      processNext(0)->ignore
    }
  }
  
  React.useEffect2(() => {
    let hasPending =
      state.items->Array.some(item =>
        switch item.status {
        | Types.Pending => true
        | Types.Decoding | Types.Compressing | Types.Done | Types.Error(_) => false
        }
      )

    if state.wasmReady && hasPending && !processingRef.current {
      processQueue()
    }
    None
  }, (state.wasmReady, state.items))
  
  let handlePasteRef = React.useRef(Nullable.null)

  React.useEffect0(() => {
    let handlePaste = (e: 'a) => {
      let files = filesFromPasteEvent(e)
      if files->Array.length > 0 {
        let items = files->Array.map(createQueueItem)
        dispatch(AddItems(items))
      }
    }
    handlePasteRef.current = Nullable.make(handlePaste)
    let _ = %raw("window.addEventListener('paste', handlePaste)")

    Some(() => {
      %raw("window.removeEventListener('paste', handlePaste)")
    })
  })
  
  let selectedItem = switch state.selectedId {
  | Some(id) =>
    let found = ref(None)
    state.items->Array.forEach(item => {
      if item.id == id {
        found := Some(item)
      }
    })
    found.contents
  | None => None
  }
  
  let formatText = switch selectedItem {
  | Some(item) => switch item.kind {
    | Png => "PNG"
    | Jpeg => "JPEG"
    | Webp => "WebP"
    | Unknown => "Unknown"
    }
  | None => "PNG"
  }
  
  let hasCompletedItems = {
    let found = ref(false)
    state.items->Array.forEach(item => {
      switch item.status {
      | Types.Done => found := true
      | Types.Pending | Types.Decoding | Types.Compressing | Types.Error(_) => ()
      }
    })
    found.contents
  }

  let completedCount = state.items->Array.filter(item => item.status == Types.Done)->Array.length

  let getAppliedOptimizations = (): option<array<string>> => {
    if !hasCompletedItems {
      None
    } else {
      let optimizations = switch state.preset {
      | Types.Ultra =>
        ["Combined filter strategy", "Maximum Zopfli passes", "Full alpha + color optimization"]
      | Types.Smaller =>
        let base = ["Maximum compression"]
        let withTrellis = if state.trellis {
          base->Array.concat(["Smart quality balance"])
        } else {
          base
        }
        let withHuffman = if state.optimizeHuffman {
          withTrellis->Array.concat(["Optimized file structure"])
        } else {
          withTrellis
        }
        withHuffman
      | Types.Faster =>
        let base = ["Fast compression", "Quality preservation"]
        let withSubsampling = if state.subsampling == "420" {
          base->Array.concat(["Speed optimization"])
        } else {
          base
        }
        withSubsampling
      | Types.Balanced => ["Balanced compression"]
      }
      
      let withProgressive = if state.progressive && formatText == "JPEG" {
        optimizations->Array.concat(["Progressive loading"])
      } else {
        optimizations
      }
      
      let final = if state.lossless {
        withProgressive->Array.concat(["Perfect quality preserved"])
      } else {
        withProgressive->Array.concat(["Size optimized"])
      }
      
      Some(final)
    }
  }
  
  let handleDownload = () => {
    switch selectedItem {
    | Some(item) =>
      switch item.compressedUrl {
      | Some(url) =>
        let extension = switch item.compressedExtension {
        | Some(ext) => ext
        | None => CompressionSettings.extensionForFormat(
            CompressionSettings.resolveForItem(item.kind, state.outputFormat),
          )
        }
        let filename = CompressionSettings.buildCompressedFilename(
          Types.Web.File.name(item.file),
          extension,
        )
        Download.downloadBlob(url, filename)
      | None => ()
      }
    | None => ()
    }
  }
  
  let handleDownloadAll = () => {
    Download.downloadAll(state.items)
  }

  let handleDownloadZip = () => {
    let _ = Download.downloadAllAsZip(state.items)
  }

  let handleCompressAll = () => {
    let hasRequeueable = state.items->Array.some(item =>
      switch item.status {
      | Types.Done | Types.Error(_) => true
      | Types.Pending | Types.Decoding | Types.Compressing => false
      },
    )
    if hasRequeueable {
      dispatch(RequeueProcessedItemsForSettings)
    }
  }

  let applySettingChange = (update: unit => unit) => {
    let hadProcessed = state.items->Array.some(item =>
      switch item.status {
      | Types.Done | Types.Error(_) => true
      | Types.Pending | Types.Decoding | Types.Compressing => false
      },
    )
    update()
    if hadProcessed {
      processingRef.current = false
      dispatch(RequeueProcessedItemsForSettings)
    }
  }

  <div className="min-h-screen flex flex-col bg-neutral-950 text-neutral-100 pb-20">
    <header className="pt-8 pb-6 px-6 text-center">
      <h1 className="text-4xl font-black tracking-tight text-neutral-100 mb-2">
        {React.string("Go-Pixo")}
      </h1>
      <p className="text-neutral-400">
        {React.string("Fast, local, client-side PNG compression powered by Go & WASM.")}
      </p>
      <div className="flex items-center justify-center gap-4 mt-4">
        <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-emerald-500 text-xs font-medium">
          <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
          </svg>
          {React.string("Runs locally on your device")}
        </div>
        <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-blue-500/10 border border-blue-500/20 text-blue-500 text-xs font-medium">
          <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          {React.string("No data sent to servers")}
        </div>
      </div>
    </header>
    
    <main className="flex-1 px-6 pb-6">
      {switch selectedItem {
      | Some(item) =>
        let itemProgress = switch state.compressionProgress {
        | Some(progress) when progress.itemId == item.id => Some(progress)
        | _ => None
        }
        <CompareView
          originalUrl={item.originalUrl}
          compressedUrl={item.compressedUrl}
          originalBytes={item.originalBytes}
          compressedBytes={item.compressedBytes}
          compressionTime={item.compressionTime}
          compressionProgress={itemProgress}
          onRemove={() => dispatch(RemoveItem(item.id))}
        />
      | None =>
        <Dropzone
          dragActive={state.dragActive}
          onDragEnter=handleDragEnter
          onDragOver=handleDragOver
          onDragLeave=handleDragLeave
          onDrop=handleDrop
          onFileSelect={handleFileSelect}
        />
      }}
      
      <FileQueue
        items={state.items}
        selectedId={state.selectedId}
        compressionProgress={state.compressionProgress}
        activeCompressions={state.activeCompressions}
        onSelect={id => dispatch(SelectItem(Some(id)))}
        onRemove={id => dispatch(RemoveItem(id))}
        onClearAll={() => dispatch(ClearAll)}
      />
    </main>
    
    <BottomBar
      format={formatText}
      preset={state.preset}
      lossless={state.lossless}
      onPresetChange={preset =>
        applySettingChange(() => dispatch(SetPreset(preset)))}
      onLosslessChange={lossless =>
        applySettingChange(() => {
          dispatch(SetLossless(lossless))
          dispatch(SetQuantization(if lossless { Types.Lossless } else { Types.Colors256 }))
        })}
      onDownload={handleDownload}
      onDownloadAll={handleDownloadAll}
      onDownloadZip={handleDownloadZip}
      hasCompletedItems={hasCompletedItems}
      completedCount={completedCount}
      appliedOptimizations={getAppliedOptimizations()}
      outputFormat={state.outputFormat}
      onOutputFormatChange={fmt =>
        applySettingChange(() => dispatch(SetOutputFormat(fmt)))}
      processingAll={state.processingAll}
      onCompressAll={handleCompressAll}
      targetWidth={state.targetWidth}
      targetHeight={state.targetHeight}
      onTargetWidthChange={w =>
        applySettingChange(() => dispatch(SetTargetWidth(w)))}
      onTargetHeightChange={h =>
        applySettingChange(() => dispatch(SetTargetHeight(h)))}
    />
  </div>
}
