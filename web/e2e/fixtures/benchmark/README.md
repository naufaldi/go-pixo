# Compression fixtures

`codex-still-life.png` is a 1536 × 1024 image generated with Codex’s built-in image generation tool on 23 October 2026. It contains leaf veins, woven fabric, camera edges, smooth light gradients, and small lettering, so real codec tests exercise both photographic detail and sharp edges. It contains no user photo or private data.

Generation prompt: “Realistic editorial desk still life with a ceramic coffee cup, leafy plant, woven linen, metal camera, and card reading GO PIXO / Local image compression. Natural window light, fine textures, smooth shadows, sharp edges, legible small text; one coherent detailed landscape photograph, no watermark.”

The original PNG is retained unchanged. Tests run it through the actual application and check decodable output, dimensions, smaller file size, and pixel-error bounds. A single synthetic scene is regression evidence, not proof of equal quality or better compression for every image.

`generated-photo-like-1024x768.png` is the existing deterministic benchmark fixture produced by `generate-fixture.mjs`.
