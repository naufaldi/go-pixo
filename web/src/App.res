open React




type action =
  | SetWasmReady(bool)
  | SetDragActive(bool)
  | AddItems(array<Types.queueItem>)
  | UpdateItem(string, Types.queueItem => Types.queueItem)
  | SelectItem(option<string>)
  | SetPreset(Types.preset)
  | SetLossless(bool)
  | SetQuantization(Types.quantizationLevel)
  | SetDithering(bool)
  | SetDitherStrength(float)
  | SetQualityTarget(int)
  | SetZopfliIterations(int)
  | SetCompressionProgress(option<Types.compressionProgress>)
  | SetCompressionTime(option<int>)
  | RemoveItem(string)
  | ClearAll

let generateId = (): string => {
  %raw("Math.random().toString(36).substring(2, 9)")
}

let createQueueItem = (file: Types.Web.File.t): Types.queueItem => {
  let kind = Types.fileKindFromMime(Types.Web.File.type_(file), Types.Web.File.name(file))
  {
    id: generateId(),
    file,
    kind,
    status: Types.Pending,
    originalUrl: None,
    compressedUrl: None,
    originalBytes: Types.Web.File.size(file),
    compressedBytes: None,
    width: None,
    height: None,
    compressionTime: None,
  }
}

let calculateGlobalProgress = (phase: string, subProgress: int): int => {
  switch phase {
  | "preprocess" => 0 + subProgress * 10 / 100
  | "filtering" => 10 + subProgress * 40 / 100
  | "deflate" => 50 + subProgress * 45 / 100
  | "finalize" => 95 + subProgress * 5 / 100
  | _ => subProgress
  }
}

let reducer = (state: Types.appState, action: action): Types.appState => {
  switch action {
  | SetWasmReady(ready) => {...state, wasmReady: ready}
  | SetDragActive(active) => {...state, dragActive: active}
  | AddItems(newItems) => {
      ...state,
      items: Array.concat(state.items, newItems),
      selectedId: switch state.selectedId {
      | None => switch newItems->Array.get(0) {
        | Some(item) => Some(item.id)
        | None => None
        }
      | Some(_) => state.selectedId
      },
    }
  | UpdateItem(id, updater) => {
      let oldItem = ref(None)
      state.items->Array.forEach(item => {
        if item.id == id {
          oldItem := Some(item)
        }
      })
      switch oldItem.contents {
      | Some(item) =>
        let newItem = updater(item)
        switch (item.originalUrl, newItem.originalUrl) {
        | (Some(oldUrl), Some(newUrl)) when oldUrl != newUrl =>
          %raw("URL.revokeObjectURL(oldUrl)")
        | _ => ()
        }
        switch (item.compressedUrl, newItem.compressedUrl) {
        | (Some(oldUrl), Some(newUrl)) when oldUrl != newUrl =>
          %raw("URL.revokeObjectURL(oldUrl)")
        | _ => ()
        }
      | None => ()
      }
      {
        ...state,
        items: state.items->Array.map(item => item.id == id ? updater(item) : item),
      }
    }
  | SelectItem(id) => {...state, selectedId: id}
  | SetPreset(preset) => {...state, preset}
  | SetLossless(lossless) => {...state, lossless}
  | SetQuantization(quantization) => {...state, quantization}
  | SetDithering(dithering) => {...state, dithering}
  | SetDitherStrength(strength) => {...state, ditherStrength: strength}
  | SetQualityTarget(target) => {...state, qualityTarget: target}
  | SetZopfliIterations(iterations) => {...state, zopfliIterations: iterations}
  | SetCompressionProgress(progress) => {
      let newProgress = switch (state.compressionProgress, progress) {
      | (Some(old), Some(new)) if old.itemId == new.itemId =>
        Some({...new, startTime: old.startTime})
      | (_, next) => next
      }
      {...state, compressionProgress: newProgress}
    }
  | SetCompressionTime(time) => {...state, compressionTime: time}
  | RemoveItem(id) => {
      let itemToRemove = state.items->Array.find(item => item.id == id)
      switch itemToRemove {
      | Some(item) => {
          switch item.originalUrl {
          | Some(url) => {
              let _ = url
              %raw("URL.revokeObjectURL(url)")
            }
          | None => ()
          }
          switch item.compressedUrl {
          | Some(url) => {
              let _ = url
              %raw("URL.revokeObjectURL(url)")
            }
          | None => ()
          }
        }
      | None => ()
      }
      let newItems = state.items->Array.filter(item => item.id != id)
      let newSelectedId = if state.selectedId == Some(id) {
        switch newItems->Array.get(0) {
        | Some(item) => Some(item.id)
        | None => None
        }
      } else {
        state.selectedId
      }
      {...state, items: newItems, selectedId: newSelectedId}
    }
  | ClearAll => {
      state.items->Array.forEach(item => {
        switch item.originalUrl {
        | Some(url) => {
            let _ = url
            %raw("URL.revokeObjectURL(url)")
          }
        | None => ()
        }
        switch item.compressedUrl {
        | Some(url) => {
            let _ = url
            %raw("URL.revokeObjectURL(url)")
          }
        | None => ()
        }
      })
      {...state, items: [], selectedId: None}
    }
  }
}

@react.component
let make = () => {
  let (state, dispatch) = React.useReducer(
    reducer,
    {
      wasmReady: false,
      dragActive: false,
      items: [],
      selectedId: None,
      preset: Types.Balanced,
      lossless: true,
      quantization: Types.Lossless,
      dithering: false,
      ditherStrength: 0.5,
      qualityTarget: 75,
      zopfliIterations: 0,
      compressionProgress: None,
      compressionTime: None,
    },
  )
  
  let workerRef = React.useRef(Nullable.null)
  let processingRef = React.useRef(false)
  let itemsRef = React.useRef(state.items)
  let compressionProgressRef = React.useRef(state.compressionProgress)
  
  React.useEffect1(() => {
    itemsRef.current = state.items
    None
  }, [state.items])
  
  React.useEffect1(() => {
    compressionProgressRef.current = state.compressionProgress
    None
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
      let data = %raw("event.data");
      logWorkerMessage(data)
      let msgType = %raw("data.type");
      switch msgType {
      | "ready" =>
        logWasmReady()
        dispatch(SetWasmReady(true))
      | "progress" =>
        let id: option<string> = %raw("typeof data.id === 'string' ? data.id : undefined");
        let phase: option<string> = %raw("typeof data.phase === 'string' ? data.phase : undefined");
        let subProgress: option<int> = %raw("typeof data.progress === 'number' ? Math.round(data.progress) : undefined");
        switch (id, phase, subProgress) {
        | (Some(id), Some(phase), Some(subProgress)) =>
          let fileSize = switch itemsRef.current->Array.find(item => item.id == id) {
          | Some(item) => item.originalBytes
          | None => 0
          };
          let globalProgress = calculateGlobalProgress(phase, subProgress)
          dispatch(SetCompressionProgress(Some({
            isActive: true,
            itemId: id,
            phase,
            progress: globalProgress,
            startTime: %raw("performance.now()"),
            fileSize,
          })))
        | _ => ()
        }
      | "compressed" =>
        let id: option<string> = %raw("typeof data.id === 'string' ? data.id : undefined");
        switch id {
        | Some(id) =>
          let compressedBytes = %raw("new Uint8Array(data.compressedBytes)");
          let compressedUrl = %raw(`
            (() => {
              const blob = new Blob([new Uint8Array(compressedBytes)], { type: 'image/png' });
              return URL.createObjectURL(blob);
            })()
          `);
          let compressedSize = compressedBytes->Array.length;
          logCompressed(id, compressedSize)
          
          // Check if image is already optimized (compressed size equals original size)
          // Note: The worker already handles the case where compressed > original by reverting to original.
          // So if compressedSize == originalBytes here, it means the image is already optimized
          // and our compression couldn't reduce it further (not a regression, just a limitation).
          let isAlreadyOptimized = switch itemsRef.current->Array.find(item => item.id == id) {
          | Some(item) => compressedSize == item.originalBytes
          | None => false
          }
          if (isAlreadyOptimized) {
            %raw("console.debug('[app] image already optimized, no size reduction possible')")
          }

          // Calculate compression time
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
          }))
          dispatch(SetCompressionProgress(None))
        | None =>
          logMissingId("[app] compressed message missing id", data)
        }
      | "error" =>
        let id: option<string> = %raw("typeof data.id === 'string' ? data.id : undefined");
        let errorMsg = %raw("data.error");
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
      | Some(_w) => %raw("w.terminate()")
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
  
  let processItem = (item: Types.queueItem): Promise.t<unit> => {
    switch item.kind {
    | Types.Png =>
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
          
          let pixels: 'a = %raw("new Uint8Array(result.pixels)")
          let presetInt = Types.presetToInt(state.preset)
          let lossy = !state.lossless
          let maxColors = Types.quantizationToInt(state.quantization)
          let ditheringEnabled = state.dithering
          let ditherStrength = state.ditherStrength
          let qualityTarget = state.qualityTarget
          let zopfliIterations = state.zopfliIterations
          let originalFileBytes = %raw("new Uint8Array(result.originalFileBytes)")
          let postCompress: ('a, string, 'a, int, int, int, int, bool, int, bool, float, int, int, 'a) => unit = %raw(
            "(worker, id, pixels, width, height, colorType, preset, lossy, maxColors, dithering, ditherStrength, qualityTarget, zopfliIterations, originalFileBytes) => { worker.postMessage({ type: 'compress', id, pixels, width, height, colorType, preset, lossy, maxColors, dithering, ditherStrength, qualityTarget, zopfliIterations, originalFileBytes }); }"
          )

          switch workerRef.current->Nullable.toOption {
          | Some(worker) =>
            postCompress(worker, item.id, pixels, result.width, result.height, result.colorType, presetInt, lossy, maxColors, ditheringEnabled, ditherStrength, qualityTarget, zopfliIterations, originalFileBytes)
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
    | Types.Jpeg =>
      ImageDecode.decodeFile(item.file)
        ->Promise.then(result => {
          dispatch(UpdateItem(item.id, item => {
            ...item,
            status: Types.Error("JPEG compression not supported yet"),
            originalUrl: Some(result.previewUrl),
            width: Some(result.width),
            height: Some(result.height),
          }))
          Promise.resolve()
        })
        ->Promise.catch(_ => {
          dispatch(UpdateItem(item.id, item => {
            ...item,
            status: Types.Error("Failed to decode JPEG"),
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
      let pendingItems = state.items->Array.filter(item => {
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
    let handlePaste = (_e: 'a) => {
      let items = %raw("e.clipboardData?.items")
      if items->Nullable.isNullable == false {
        let itemArray = %raw("Array.from(items)")
        let imageItems = itemArray->Array.filter(_item => {
          %raw("item.type.startsWith('image/')")
        })
        if imageItems->Array.length > 0 {
          let files = imageItems->Array.map(_item => {
            %raw("item.getAsFile()")
          })
          let items = files->Array.map(createQueueItem)
          dispatch(AddItems(items))
        }
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
  
  let handleDownload = () => {
    switch selectedItem {
    | Some(item) =>
      switch item.compressedUrl {
      | Some(url) =>
        let filename = "compressed_" ++ Types.Web.File.name(item.file)
        Download.downloadBlob(url, filename)
      | None => ()
      }
    | None => ()
    }
  }
  
  let handleDownloadAll = () => {
    Download.downloadAll(state.items)
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
        onSelect={id => dispatch(SelectItem(Some(id)))}
        onRemove={id => dispatch(RemoveItem(id))}
        onClearAll={() => dispatch(ClearAll)}
      />
    </main>
    
    <BottomBar
      format=formatText
      preset={state.preset}
      lossless={state.lossless}
      quantization={state.quantization}
      dithering={state.dithering}
      ditherStrength={state.ditherStrength}
      qualityTarget={state.qualityTarget}
      zopfliIterations={state.zopfliIterations}
      onPresetChange={preset => dispatch(SetPreset(preset))}
      onLosslessChange={lossless => dispatch(SetLossless(lossless))}
      onQuantizationChange={quantization => dispatch(SetQuantization(quantization))}
      onDitheringChange={dithering => dispatch(SetDithering(dithering))}
      onDitheringStrengthChange={strength => dispatch(SetDitherStrength(strength))}
      onQualityTargetChange={target => dispatch(SetQualityTarget(target))}
      onZopfliIterationsChange={iterations => dispatch(SetZopfliIterations(iterations))}
      onDownload=handleDownload
      onDownloadAll=handleDownloadAll
      hasCompletedItems
    />
  </div>
}
