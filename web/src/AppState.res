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
  | SetProgressive(bool)
  | SetSubsampling(string)
  | SetTrellis(bool)
  | SetOptimizeHuffman(bool)
  | SetCompressionProgress(option<Types.compressionProgress>)
  | SetCompressionTime(option<int>)
  | RemoveItem(string)
  | ClearAll
  | SetOutputFormat(Types.outputFormat)
  | SetActiveCompression(string, option<Types.compressionProgress>)
  | SetProcessingAll(bool)
  | SetTargetWidth(option<int>)
  | SetTargetHeight(option<int>)
  | RequeueProcessedItemsForSettings

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
    compressedMime: None,
    compressedExtension: None,
  }
}

let initialState: Types.appState = {
  wasmReady: false,
  dragActive: false,
  items: [],
  selectedId: None,
  preset: Types.Balanced,
  lossless: false,
  quantization: Types.Colors256,
  dithering: false,
  ditherStrength: 0.5,
  qualityTarget: 75,
  zopfliIterations: 0,
  progressive: false,
  subsampling: "420",
  trellis: false,
  optimizeHuffman: false,
  compressionProgress: None,
  compressionTime: None,
  outputFormat: Types.SameAsInput,
  activeCompressions: [],
  processingAll: false,
  targetWidth: None,
  targetHeight: None,
}

let revokeChangedUrl = (oldUrl: option<string>, newUrl: option<string>) => {
  switch (oldUrl, newUrl) {
  | (Some(old), Some(next)) when old != next => BlobUrl.revokeObjectURL(old)
  | (Some(old), None) => BlobUrl.revokeObjectURL(old)
  | _ => ()
  }
}

let revokeItemUrls = (item: Types.queueItem) => {
  switch item.originalUrl {
  | Some(url) => BlobUrl.revokeObjectURL(url)
  | None => ()
  }
  switch item.compressedUrl {
  | Some(url) => BlobUrl.revokeObjectURL(url)
  | None => ()
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
      let updatedItem = ref(None)
      let items = state.items->Array.map(item =>
        if item.id == id {
          let next = updater(item)
          revokeChangedUrl(item.originalUrl, next.originalUrl)
          revokeChangedUrl(item.compressedUrl, next.compressedUrl)
          updatedItem := Some(next)
          next
        } else {
          item
        }
      )
      switch updatedItem.contents {
      | Some(_) => {...state, items}
      | None => state
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
  | SetProgressive(progressive) => {...state, progressive}
  | SetSubsampling(subsampling) => {...state, subsampling}
  | SetTrellis(trellis) => {...state, trellis}
  | SetOptimizeHuffman(optimize) => {...state, optimizeHuffman: optimize}
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
      state.items
      ->Array.find(item => item.id == id)
      ->Option.forEach(revokeItemUrls)

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
      state.items->Array.forEach(revokeItemUrls)
      {...state, items: [], selectedId: None}
    }
  | SetOutputFormat(fmt) => {...state, outputFormat: fmt}
  | SetProcessingAll(b) => {...state, processingAll: b}
  | SetTargetWidth(w) => {...state, targetWidth: w}
  | SetTargetHeight(h) => {...state, targetHeight: h}
  | RequeueProcessedItemsForSettings => {
      state.items->Array.forEach(item => {
        switch item.compressedUrl {
        | Some(url) => BlobUrl.revokeObjectURL(url)
        | None => ()
        }
      })
      {
        ...state,
        items: state.items->Array.map(item =>
          switch item.status {
          | Types.Done | Types.Error(_) => {
              ...item,
              status: Types.Pending,
              compressedUrl: None,
              compressedBytes: None,
              compressionTime: None,
              compressedMime: None,
              compressedExtension: None,
            }
          | Types.Pending | Types.Decoding | Types.Compressing => item
          },
        ),
        compressionProgress: None,
        processingAll: false,
      }
    }
  | SetActiveCompression(id, progress) => {
      let filtered = state.activeCompressions->Array.filter(((pid, _)) => pid != id)
      let updated = switch progress {
      | Some(p) => Array.concat(filtered, [(id, p)])
      | None => filtered
      }
      {...state, activeCompressions: updated}
    }
  }
}
