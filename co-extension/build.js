const esbuild = require('esbuild');
const fs = require('fs');
const path = require('path');

const production = process.argv.includes('--production');
const watch = process.argv.includes('--watch');

async function main() {
  const extensionCtx = await esbuild.context({
    entryPoints: ['src/extension.ts'],
    bundle: true,
    platform: 'node',
    target: 'node18',
    format: 'cjs',
    outfile: 'dist/extension.js',
    external: ['vscode'],
    sourcemap: !production,
    minify: production,
  });

  const webviewCtx = await esbuild.context({
    entryPoints: ['src/webview/index.tsx'],
    bundle: true,
    platform: 'browser',
    target: 'es2020',
    format: 'iife',
    outfile: 'dist/webview/index.js',
    jsx: 'automatic',
    sourcemap: !production,
    minify: production,
    define: {
      'process.env.NODE_ENV': production ? '"production"' : '"development"'
    },
    globalName: 'cocursor',
  });

  if (watch) {
    await extensionCtx.watch();
    await webviewCtx.watch();
    console.log('Watching...');
  } else {
    await extensionCtx.rebuild();
    await webviewCtx.rebuild();
    await extensionCtx.dispose();
    await webviewCtx.dispose();
    console.log('Build complete');
  }
}

main().catch(e => {
  console.error(e);
  process.exit(1);
});
