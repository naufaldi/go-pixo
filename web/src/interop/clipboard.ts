export function filesFromPasteEvent(event: ClipboardEvent): File[] {
  const list = event.clipboardData?.items;
  if (!list) return [];

  return Array.from(list)
    .filter((item) => item.type.startsWith('image/'))
    .map((item) => item.getAsFile())
    .filter((file): file is File => file != null);
}
