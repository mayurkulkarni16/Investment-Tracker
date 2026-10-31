import { spawnSync } from 'node:child_process';

const api = process.env.VITE_API_URL;
if (!api || !/^https:\/\/[^/]+\/api\/v1\/?$/.test(api)) {
  throw new Error('Set VITE_API_URL to your deployed HTTPS backend, e.g. https://your-host/api/v1, before building Android. Relative URLs cannot work in a packaged app.');
}
const result = spawnSync(process.execPath, ['node_modules/typescript/bin/tsc', '-b'], { stdio: 'inherit' });
if (result.status !== 0) process.exit(result.status ?? 1);
const build = spawnSync(process.execPath, ['node_modules/vite/bin/vite.js', 'build'], { stdio: 'inherit' });
process.exit(build.status ?? 1);
