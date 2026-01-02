open React
open Types



type action =
  | SetWasmReady(bool)
  | SetDragActive(bool)
  | AddItems(array<queueItem>)
  | UpdateItem(string, queueItem => queueItem)
  | SelectItem(option<string>)
  | SetPreset(preset)
  | SetLossless(bool)
  | SetCompareMode(compareMode)

let generateId = (): string => {
  %raw("Math.random().toString(36).substring(2, 9)")
}

let createQueueItem = (file: Types.Web.File.t): queueItem => {
  let kind = fileKindFromMime(Web.File.type_(file), Web.File.name(file))
  {
    id: generateId(),
    file,
    kind,
    status: Pending,
    originalUrl: None,
    compressedUrl: None,
    originalBytes: Web.File.size(file),
    compressedBytes: None,
    width: None,
    height: None,
  }
}

let reducer = (state: appState, action: action): appState => {
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
  | SetCompareMode(mode) => {...state, compareMode: mode}
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
      preset: Balanced,
      lossless: true,
      compareMode: SideBySide,
    },
  )
  
  let processingRef = React.useRef(false)
  
  React.useEffect0(() => {
    Wasm.initWasm()
      ->Promise.then(_ => {
        dispatch(SetWasmReady(true))
        Promise.resolve()
      })
      ->Promise.catch(_ => {
        dispatch(SetWasmReady(true))
        Promise.resolve()
      })
      ->ignore
    None
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
    if files->Js.Nullable.isNullable == false {
      let fileArray = %raw("Array.from(files)")
      let items = fileArray->Array.map(createQueueItem)
      dispatch(AddItems(items))
    }
  }
  
  let handleFileSelect = (files: array<Types.Web.File.t>) => {
    let items = files->Array.map(createQueueItem)
    dispatch(AddItems(items))
  }
  
  let processItem = (item: queueItem): Promise.t<unit> => {
    switch item.kind {
    | Png =>
      dispatch(UpdateItem(item.id, item => {...item, status: Decoding}))
      ImageDecode.decodeFile(item.file)
        ->Promise.then(result => {
          dispatch(UpdateItem(item.id, item => {
            ...item,
            status: Compressing,
            originalUrl: Some(result.previewUrl),
            width: Some(result.width),
            height: Some(result.height),
          }))
          
          let pixels = %raw("new Uint8Array(result.pixels)")
          let presetInt = presetToInt(state.preset)
          let lossy = !state.lossless
          
          let compressedBytes = Wasm.encodePngImageWithOptions(
            pixels,
            result.width,
            result.height,
            result.colorType,
            presetInt,
            lossy,
          )
          
          let compressedUrl = %raw("
            (() => {
              const blob = new Blob([new Uint8Array(compressedBytes)], { type: 'image/png' });
              return URL.createObjectURL(blob);
            })()
          ")
          let compressedSize = compressedBytes->Array.length
          
          dispatch(UpdateItem(item.id, item => {
            ...item,
            status: Done,
            compressedUrl: Some(compressedUrl),
            compressedBytes: Some(compressedSize),
          }))
          
          Promise.resolve()
        })
        ->Promise.catch(err => {
          let errorMsg = %raw("err.message || 'Failed to process image'")
          dispatch(UpdateItem(item.id, item => {
            ...item,
            status: Error(errorMsg),
          }))
          Promise.resolve()
        })
    | Jpeg =>
      ImageDecode.decodeFile(item.file)
        ->Promise.then(result => {
          dispatch(UpdateItem(item.id, item => {
            ...item,
            status: Error("JPEG compression not supported yet"),
            originalUrl: Some(result.previewUrl),
            width: Some(result.width),
            height: Some(result.height),
          }))
          Promise.resolve()
        })
        ->Promise.catch(_ => {
          dispatch(UpdateItem(item.id, item => {
            ...item,
            status: Error("Failed to decode JPEG"),
          }))
          Promise.resolve()
        })
    | Unknown =>
      dispatch(UpdateItem(item.id, item => {
        ...item,
        status: Error("Unsupported file type"),
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
        | Pending => true
        | _ => false
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
  
  React.useEffect(() => {
    if state.wasmReady && state.items->Array.length > 0 {
      processQueue()
    }
    None
  }, [state.wasmReady])
  
  React.useEffect(() => {
    if state.wasmReady {
      processQueue()
    }
    None
  }, [state.items])
  
  let handlePasteRef = React.useRef(Nullable.null)

  React.useEffect0(() => {
    let handlePaste = (e: 'a) => {
      let items = %raw("e.clipboardData?.items")
      if items->Js.Nullable.isNullable == false {
        let itemArray = %raw("Array.from(items)")
        let imageItems = itemArray->Array.filter(item => {
          %raw("item.type.startsWith('image/')")
        })
        if imageItems->Array.length > 0 {
          let files = imageItems->Array.map(item => {
            %raw("item.getAsFile()")
          })
          let items = files->Array.map(createQueueItem)
          dispatch(AddItems(items))
        }
      }
    }
    handlePasteRef.current = Js.Nullable.return(handlePaste)
    %raw("window.addEventListener('paste', handlePaste)")

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
      | Done => found := true
      | _ => ()
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
    </header>
    
    <main className="flex-1 px-6 pb-6">
      <Dropzone
        dragActive={state.dragActive}
        onDragEnter=handleDragEnter
        onDragOver=handleDragOver
        onDragLeave=handleDragLeave
        onDrop=handleDrop
        onFileSelect=handleFileSelect
      />
      
      {switch selectedItem {
      | Some(item) =>
        <CompareView
          mode={state.compareMode}
          originalUrl={item.originalUrl}
          compressedUrl={item.compressedUrl}
          originalBytes={item.originalBytes}
          compressedBytes={item.compressedBytes}
        />
      | None => React.null
      }}
      
      <FileQueue
        items={state.items}
        selectedId={state.selectedId}
        onSelect={id => dispatch(SelectItem(Some(id)))}
      />
    </main>
    
    <BottomBar
      format=formatText
      preset={state.preset}
      lossless={state.lossless}
      compareMode={state.compareMode}
      onPresetChange={preset => dispatch(SetPreset(preset))}
      onLosslessChange={lossless => dispatch(SetLossless(lossless))}
      onCompareModeChange={mode => dispatch(SetCompareMode(mode))}
      onDownload=handleDownload
      onDownloadAll=handleDownloadAll
      hasCompletedItems
    />
  </div>
}
