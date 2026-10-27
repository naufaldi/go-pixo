import {expect, test} from '@playwright/test';
import {jpegExifSegment} from '../src/test/jpeg';

// Generate valid native JPEG pixels at runtime; only the bounded EXIF segment is synthetic.
test('native JPEG orientations 1..8 match preflight and unchanged byte dimensions in both endian orders', async ({page}) => {
  await page.goto('/');
  const segments=[true,false].flatMap(le => Array.from({length:8},(_,i) => ({
    orientation:i+1,le,bytes:Array.from(jpegExifSegment(i+1,le)),
  })));
  const cases=await page.evaluate(async segments => {
    const decodePath='/src/interop/imageDecode.ts', formatsPath='/src/interop/imageFormats.ts';
    const {decodeFile}=await import(decodePath);
    const {inspectImage,imageDisplayDimensions}=await import(formatsPath);
    const canvas=document.createElement('canvas'); canvas.width=3; canvas.height=4;
    canvas.getContext('2d')!.fillRect(0,0,3,4);
    const blob=await new Promise<Blob>(resolve => canvas.toBlob(b => resolve(b!),'image/jpeg'));
    canvas.width=0; canvas.height=0;
    const base=new Uint8Array(await blob.arrayBuffer());
    const results=[];
    for (const {orientation,le,bytes:segment} of segments) {
      const bytes=new Uint8Array(base.length+segment.length);
      bytes.set(base.subarray(0,2)); bytes.set(segment,2); bytes.set(base.subarray(2),2+segment.length);
      const file=new File([bytes],'oriented.jpg',{type:'image/jpeg'});
      const pixels=await decodeFile(file,undefined,{lossy:true});
      let canvases=0;
      const create=document.createElement.bind(document);
      document.createElement=((...args: Parameters<typeof create>) => {
        if (args[0]==='canvas') canvases++;
        return create(...args);
      }) as typeof document.createElement;
      let unchanged;
      try { unchanged=await decodeFile(file,undefined,{lossy:false,targetWidth:pixels.width}); }
      finally { document.createElement=create; }
      const preview=new Image(); preview.src=unchanged.previewUrl; await preview.decode();
      results.push({orientation,le,storage:[inspectImage(bytes).width,inspectImage(bytes).height],
        planned:imageDisplayDimensions(inspectImage(bytes)),native:[pixels.width,pixels.height],
        preview:[preview.naturalWidth,preview.naturalHeight],unchanged:[unchanged.width,unchanged.height],
        byteOnly:unchanged.pixels===undefined,canvases,pixelBytes:pixels.pixels.length});
      preview.src=''; URL.revokeObjectURL(pixels.previewUrl); URL.revokeObjectURL(unchanged.previewUrl);
    }
    return results;
  },segments);
  for (const result of cases) {
    const expected=result.orientation>=5 ? [4,3] : [3,4];
    expect(result.storage).toEqual([3,4]);
    expect(result.planned).toEqual({width:expected[0],height:expected[1]});
    expect(result.native).toEqual(expected);
    expect(result.preview).toEqual(expected);
    expect(result.unchanged).toEqual(expected);
    expect(result.byteOnly).toBe(true); expect(result.canvases).toBe(0); expect(result.pixelBytes).toBe(48);
  }
});

test('rotated 3000x4000 JPEG keeps exact bytes at displayed width 4000 and resizes to 5000x3750', async ({page}) => {
  const goRequests: string[]=[];
  page.on('request',request => { if (/\/(main\.wasm|wasm_exec\.js)(?:$|\?)/.test(request.url())) goRequests.push(request.url()); });
  await page.goto('/');
  // index.html loads the runtime script on page load; measure subsequent worker startup.
  goRequests.length=0;
  await page.getByRole('checkbox',{name:'Perfect Quality'}).check();
  await page.getByRole('spinbutton',{name:'Target width'}).fill('4000');
  await page.evaluate(async segment => {
    const canvas=document.createElement('canvas'); canvas.width=3000; canvas.height=4000;
    const context=canvas.getContext('2d')!;
    context.fillStyle='#2878b8'; context.fillRect(0,0,3000,4000);
    const blob=await new Promise<Blob>(resolve => canvas.toBlob(b => resolve(b!),'image/jpeg'));
    canvas.width=0; canvas.height=0;
    const base=new Uint8Array(await blob.arrayBuffer());
    const bytes=new Uint8Array(base.length+segment.length);
    bytes.set(base.subarray(0,2)); bytes.set(segment,2); bytes.set(base.subarray(2),2+segment.length);
    const transfer=new DataTransfer(); transfer.items.add(new File([bytes],'rotated-3000x4000.jpg',{type:'image/jpeg'}));
    const input=document.querySelector<HTMLInputElement>('input[type=file]')!;
    input.files=transfer.files; input.dispatchEvent(new Event('change',{bubbles:true}));
  },Array.from(jpegExifSegment(6)));
  await expect(page.getByTestId('batch-summary')).toContainText('1 completed, 0 failed');
  const unchanged=await page.evaluate(async () => {
    const original=document.querySelector<HTMLImageElement>('img[alt="Original"]')!;
    const output=document.querySelector<HTMLImageElement>('img[alt="Compressed"]')!;
    await Promise.all([original.decode(),output.decode()]);
    const before=new Uint8Array(await fetch(original.src).then(r => r.arrayBuffer()));
    const after=new Uint8Array(await fetch(output.src).then(r => r.arrayBuffer()));
    return {same:before.length===after.length && before.every((b,i) => b===after[i]),
      original:[original.naturalWidth,original.naturalHeight],output:[output.naturalWidth,output.naturalHeight]};
  });
  expect(unchanged).toEqual({same:true,original:[4000,3000],output:[4000,3000]});
  expect(goRequests).toEqual([]);

  // The reviewer's lossless same-format width 3000 must require pixels, then report
  // the existing honest JPEG-encoding limitation instead of silently returning width 4000.
  await page.getByRole('spinbutton',{name:'Target width'}).fill('3000');
  await expect(page.getByTestId('batch-summary')).toContainText('0 completed, 1 failed');
  await expect(page.getByText(/JPEG encoding is lossy/).first()).toBeVisible();

  await page.getByRole('checkbox',{name:'Perfect Quality'}).uncheck();
  await page.getByRole('combobox',{name:'Output format'}).selectOption('webp');
  await page.getByRole('spinbutton',{name:'Target width'}).fill('5000');
  await expect(page.getByTestId('batch-summary')).toContainText('1 completed, 0 failed',{timeout:60000});
  const resized=await page.locator('img[alt="Compressed"]').evaluate(async (output: HTMLImageElement) => {
    const blob=await fetch(output.src).then(r => r.blob());
    const bitmap=await createImageBitmap(blob);
    const result={width:bitmap.width,height:bitmap.height,mime:blob.type}; bitmap.close(); return result;
  });
  expect(resized).toEqual({width:5000,height:3750,mime:'image/webp'});
});
