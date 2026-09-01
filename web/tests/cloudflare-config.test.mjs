import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const configUrl = new URL("../../wrangler.jsonc", import.meta.url);

test("Cloudflare config deploys the Vite build as an SPA", async () => {
  const config = JSON.parse(await readFile(configUrl, "utf8"));

  assert.equal(config.name, "go-pixo");
  assert.equal(config.compatibility_date, "2026-09-01");
  assert.deepEqual(config.assets, {
    directory: "./web/dist",
    not_found_handling: "single-page-application",
  });
  assert.equal("main" in config, false);
  assert.deepEqual(config.routes, [
    {
      pattern: "go-pixo.naufaldi.com",
      custom_domain: true,
    },
  ]);
});
