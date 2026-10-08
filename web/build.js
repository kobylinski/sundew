import { readdir, rm } from 'node:fs/promises';
import { build } from 'vite';

// Keep the committed Go-only fallback; replace generated files, including old hashes.
const output = new URL('../internal/ui/static/', import.meta.url);
for (const name of await readdir(output)) {
  if (name !== 'placeholder.html') await rm(new URL(name, output), { recursive: true, force: true });
}
await build();
