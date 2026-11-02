import {test, expect} from '@playwright/test'
import {fileURLToPath} from 'node:url'

const png = fileURLToPath(new URL('./fixtures/sample-32x32.png', import.meta.url))
const heic = fileURLToPath(new URL('../src/interop/codecs-fixtures/rainbow-451x461.heic', import.meta.url))

test('retains valid PNG bytes when worker bitmap decoding is unavailable', async ({page}) => {
  for (const pattern of ['**/src/worker.ts*','**/src/interop/codecsDecode.worker.ts*']) {
    await page.route(pattern, async route => {
      const response=await route.fetch()
      await route.fulfill({response,body:'globalThis.createImageBitmap=undefined;\n'+await response.text()})
    })
  }
  await page.goto('/')
  await page.getByRole('checkbox', {name:'Perfect Quality'}).check()
  await page.locator('input[type=file]').setInputFiles(png)
  await expect(page.getByTestId('batch-summary')).toContainText('1 completed, 0 failed, 0 remaining')
  await expect(page.getByTestId('download-primary')).toBeEnabled()
})

test('rejects corrupt passthrough and contained HEIC inputs, then accepts a valid retry', async ({page}) => {
  await page.goto('/')
  await page.getByRole('checkbox', {name:'Perfect Quality'}).check()
  const input = page.locator('input[type=file]')
  await input.setInputFiles({name:'corrupt.jpg', mimeType:'image/jpeg', buffer:Buffer.from([255,216,255,192,0,7,8,0,1,0,1])})
  await expect(page.getByTestId('batch-summary')).toContainText('0 completed, 1 failed')
  await page.getByLabel('Add images').setInputFiles(heic)
  await expect(page.getByTestId('batch-summary')).toContainText('0 completed, 2 failed')
  await page.getByText('rainbow-451x461.heic', {exact:true}).click()
  await expect(page.getByRole('alert')).toContainText('temporarily unavailable')
  await page.getByLabel('Add images').setInputFiles(png)
  await expect(page.getByTestId('batch-summary')).toContainText('1 completed, 2 failed, 0 remaining')
  await expect(page.getByTestId('download-primary')).toBeDisabled()
  await page.getByText('sample-32x32.png', {exact:true}).click()
  await expect(page.getByTestId('download-primary')).toBeEnabled()
})

test('decodes a 24 MP PNG in a worker without main-thread pixel readback', async ({page}) => {
  test.setTimeout(120_000)
  await page.goto('/')
  const preset = page.getByRole('slider', {name:'Compression preset'})
  await preset.fill('3')
  const started = await page.evaluate(async () => {
    const canvas=document.createElement('canvas')
    canvas.width=6000; canvas.height=4000
    const context=canvas.getContext('2d')!
    context.fillStyle='#e23744'; context.fillRect(0,0,6000,4000)
    const blob=await new Promise<Blob>(resolve => canvas.toBlob(value=>resolve(value!), 'image/png'))
    canvas.width=0; canvas.height=0
    let readbacks=0
    const original=CanvasRenderingContext2D.prototype.getImageData
    CanvasRenderingContext2D.prototype.getImageData=function(...args: Parameters<typeof original>) {
      readbacks++
      return original.apply(this,args)
    }
    Object.assign(window,{getDecodeReadbacks:()=>readbacks})
    const transfer=new DataTransfer()
    transfer.items.add(new File([blob],'large-24mp.png',{type:'image/png'}))
    const input=document.querySelector<HTMLInputElement>('input[type=file]')!
    input.files=transfer.files
    const started=performance.now()
    input.dispatchEvent(new Event('change',{bubbles:true}))
    return started
  })
  await expect(page.getByTestId('batch-summary')).toContainText('1 completed, 0 failed, 0 remaining', {timeout:90_000})
  const result=await page.evaluate(async () => {
    const image=document.querySelector<HTMLImageElement>('img[alt="Compressed"]')!
    await image.decode()
    return {width:image.naturalWidth,height:image.naturalHeight,
      readbacks:(window as unknown as {getDecodeReadbacks():number}).getDecodeReadbacks(),finished:performance.now()}
  })
  expect(result).toMatchObject({width:6000,height:4000,readbacks:0})
  console.log(`24 MP decode/compress completed in ${(result.finished-started).toFixed(0)} ms; main-thread readbacks: ${result.readbacks}`)
})
