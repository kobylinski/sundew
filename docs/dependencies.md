# Dependencies

Checked on 2026-10-08. The Go application uses only the standard library: no third-party Go
module is added. `go list -m all` reports only `github.com/kobylinski/sundew`.

Messages are kept in memory, as decided in
[the memory-only decision](journal/2026-10-08/decision-messages-are-kept-in-memory-only.md).
No SQLite driver is required.

Build/test infrastructure uses the official `golang:1.26-alpine` build image,
`python:3.14-alpine` for isolated HTTP acceptance clients and the actions listed below.
None ships in the scratch runtime image. Future third-party Go modules
must have an upstream commit within six months when added, with its date and source recorded here.

## GitHub Actions and workflow linting

Checked on **2026-10-08** against the upstream GitHub repository APIs. CI and release
use the same full SHA pins and retain the existing action major versions. Maintenance
means repository activity within six months; the pinned release itself may be older.

| Dependency | Version and pin | Latest upstream commit at check |
| --- | --- | --- |
| [actions/checkout](https://github.com/actions/checkout) | v4.4.0, `11d5960a326750d5838078e36cf38b85af677262` (2026-07-16) | [2026-07-20](https://github.com/actions/checkout/commit/f548e57e544e1ff5a4c46bf1e1b8685f8e4a348a) — within six months |
| [actions/setup-go](https://github.com/actions/setup-go) | v5.6.0, `40f1582b2485089dde7abd97c1529aa768e1baff` (2025-12-15) | [2026-09-28](https://github.com/actions/setup-go/commit/90ad2b35f69faf97585ad74d28fa006d2739b7af) — within six months |
| [rhysd/actionlint](https://github.com/rhysd/actionlint) | v1.7.12 container, local verification only | [2026-04-19](https://github.com/rhysd/actionlint/commit/011a6d15e749bb3f2d771eed9c7aa0e7e3e10ee7) — within six months |

The release workflow uses the hosted runner's Docker CLI/Buildx and an isolated
BuildKit builder. It introduces no Docker action or application module.

## Twilio SDK acceptance

Maintenance checked 2026-10-08; these SDKs run only in acceptance containers.

| Dependency | Purpose and pin | Maintenance evidence |
| --- | --- | --- |
| [twilio-python](https://github.com/twilio/twilio-python) | Container acceptance only; `twilio==9.11.2` | [Commit 2fd57cf](https://github.com/twilio/twilio-python/commit/2fd57cf8f344c472c6e3b14205ad266d1bd6babc), 2026-09-28; within six months. |
| [twilio-node](https://github.com/twilio/twilio-node) | Container acceptance only; `twilio@6.1.2` | [Commit 54049bf](https://github.com/twilio/twilio-node/commit/54049bffe48104324efd101e7c7180db1a16c27f), 2026-10-07; within six months. |

## Svelte UI prototype

Checked on **8 October 2026** for the Svelte UI prototype in `prototypes/web-ui/`.

The three direct dependencies pass the six-month maintenance rule. Marek approved a **prototype-only exception** for the 12 older transitive packages on 8 October 2026 at 04:27 UTC, answering `tasks:40a0d78a850976e8#q1` with `prototype_exception`. This permits the pinned dependency set for this disposable design prototype only. It does not approve the dependencies for the production UI.

All versions are pinned by `prototypes/web-ui/package-lock.json`. Nothing installs on the Go server or in the released runtime image. Optional native binaries for other platforms are included because they are in the lockfile.

| Modules (locked versions) | Latest repository commit | Check |
| --- | --- | --- |
| `source-map-js@1.2.2` | [2026-09-30](https://github.com/7rulnik/source-map-js/commit/b158388e8721f622ef3fc82b0b545c7146bfc203) | Within six months |
| `aria-query@5.3.1` | [2024-04-01](https://github.com/A11yance/aria-query/commit/c151dafa6ab17143479e5aeaa0826a3d657cc5be) | Older than six months — prototype exception approved |
| `axobject-query@4.1.0` | [2024-09-24](https://github.com/A11yance/axobject-query/commit/526857dddc3a592284441bd70e4a2162770375e5) | Older than six months — prototype exception approved |
| `acorn@8.19.0` | [2026-10-07](https://github.com/acornjs/acorn/commit/ac794b1532dcc5031adb49c44bd5214a61ef8eb5) | Within six months |
| `nanoid@3.3.20` | [2026-10-05](https://github.com/ai/nanoid/commit/3167e108865702c0539d8e552365ed840ac9d8d9) | Within six months |
| `picocolors@1.1.1` | [2024-11-18](https://github.com/alexeyraspopov/picocolors/commit/0e7c4af2de299dd7bc5916f2bddd151fa2f66740) | Older than six months — prototype exception approved |
| `esm-env@1.2.2` | [2026-01-27](https://github.com/benmccann/esm-env/commit/ad2a6a391f42d124304f01531e675944131e8a06) | Older than six months — prototype exception approved |
| `@types/estree@1.0.9` | [2026-10-08](https://github.com/DefinitelyTyped/DefinitelyTyped/commit/709ce6673161fb406dd4d995ba11d2a8ad52600b) | Within six months |
| `fsevents@2.3.3` | [2023-08-21](https://github.com/fsevents/fsevents/commit/2db891e51aa0f2975c5eaaf6aa30f13d720a830a) | Older than six months — prototype exception approved |
| `locate-character@3.0.0` | [2018-01-14](https://gitlab.com/Rich-Harris/locate-character/-/commit/f2e7151428b243fc3dd54db2f90d61dae0fa42ab) | Older than six months — prototype exception approved |
| `@jridgewell/resolve-uri@3.1.2` | [2025-08-04](https://github.com/jridgewell/resolve-uri/commit/730a61ad423359a972b7935a4304db04983a3251) | Older than six months — prototype exception approved |
| `@jridgewell/gen-mapping@0.3.13`, `@jridgewell/remapping@2.3.5`, `@jridgewell/sourcemap-codec@1.6.0`, `@jridgewell/trace-mapping@0.3.31` | [2026-08-28](https://github.com/jridgewell/sourcemaps/commit/b7600158f4de012d246b7834a73c7c2190e75136) | Within six months |
| `detect-libc@2.1.2` | [2025-10-05](https://github.com/lovell/detect-libc/commit/98928b7d26bd5c50dee12af0e433b47825d84df0) | Older than six months — prototype exception approved |
| `clsx@2.1.1` | [2024-04-23](https://github.com/lukeed/clsx/commit/925494cf31bcd97d3337aacd34e659e80cae7fe2) | Older than six months — prototype exception approved |
| `picomatch@4.0.7` | [2026-08-27](https://github.com/micromatch/picomatch/commit/9fab7bf61913ea334c4203ed321c426cfde44a42) | Within six months |
| `@oxc-project/types@0.153.0` | [2026-10-08](https://github.com/oxc-project/oxc/commit/4a28c91f33e39d7f5c5a5bb4658a5946d3440e60) | Within six months |
| `lightningcss@1.33.0`, `lightningcss-android-arm64@1.33.0`, `lightningcss-darwin-arm64@1.33.0`, `lightningcss-darwin-x64@1.33.0`, `lightningcss-freebsd-x64@1.33.0`, `lightningcss-linux-arm-gnueabihf@1.33.0`, `lightningcss-linux-arm64-gnu@1.33.0`, `lightningcss-linux-arm64-musl@1.33.0`, `lightningcss-linux-x64-gnu@1.33.0`, `lightningcss-linux-x64-musl@1.33.0`, `lightningcss-win32-arm64-msvc@1.33.0`, `lightningcss-win32-x64-msvc@1.33.0` | [2026-09-29](https://github.com/parcel-bundler/lightningcss/commit/987e1bf1800c6f3223b097037be326c64a81b694) | Within six months |
| `postcss@8.5.29` | [2026-10-05](https://github.com/postcss/postcss/commit/1fd8f881875330b284d65961c2c158f61d4e3bab) | Within six months |
| `is-reference@3.0.3` | [2024-11-12](https://github.com/Rich-Harris/is-reference/commit/8bb053129bfabe2f6a7d7ed050159d67ebe82829) | Older than six months — prototype exception approved |
| `magic-string@1.4.3`, `magic-string@0.30.21` | [2026-10-05](https://github.com/Rich-Harris/magic-string/commit/993d27d8aee16d2ca46646a5c3c0b290c8084323) | Within six months |
| `@rolldown/pluginutils@1.0.1` | [2026-10-05](https://github.com/rolldown/plugins/commit/a0e8ff7595616d1b5b7ea9174da2f628054cba79) | Within six months |
| `@rolldown/binding-android-arm-eabi@1.2.13`, `@rolldown/binding-android-arm64@1.2.13`, `@rolldown/binding-darwin-arm64@1.2.13`, `@rolldown/binding-darwin-x64@1.2.13`, `@rolldown/binding-freebsd-x64@1.2.13`, `@rolldown/binding-linux-arm-gnueabihf@1.2.13`, `@rolldown/binding-linux-arm64-gnu@1.2.13`, `@rolldown/binding-linux-arm64-musl@1.2.13`, `@rolldown/binding-linux-ppc64-gnu@1.2.13`, `@rolldown/binding-linux-s390x-gnu@1.2.13`, `@rolldown/binding-linux-x64-gnu@1.2.13`, `@rolldown/binding-linux-x64-musl@1.2.13`, `@rolldown/binding-openharmony-arm64@1.2.13`, `@rolldown/binding-win32-arm64-msvc@1.2.13`, `@rolldown/binding-win32-x64-msvc@1.2.13`, `rolldown@1.2.13` | [2026-10-07](https://github.com/rolldown/rolldown/commit/ccf2ab4a2844af98e4653339f579550e50121b2c) | Within six months |
| `tinyglobby@0.2.17` | [2026-08-30](https://github.com/SuperchupuDev/tinyglobby/commit/d9f76e12a99902e6a564bbc43dc6e7c84978a5c2) | Within six months |
| `@sveltejs/acorn-typescript@1.0.13` | [2026-10-02](https://github.com/sveltejs/acorn-typescript/commit/85a27a053a7a71090ac426e6f501b2a8f65133e6) | Within six months |
| `devalue@5.9.4` | [2026-10-07](https://github.com/sveltejs/devalue/commit/5cd712a164b528f81fd8258b1cab8e4fca760137) | Within six months |
| `esrap@2.4.0` | [2026-09-26](https://github.com/sveltejs/esrap/commit/ca0c2520f55a655cd77edf14dab273b261f05610) | Within six months |
| `svelte@5.57.2` | [2026-10-07](https://github.com/sveltejs/svelte/commit/707c28146b0f0a6d5404a1bd4769874c3c24851a) | Within six months |
| `@sveltejs/vite-plugin-svelte@7.3.1` | [2026-10-05](https://github.com/sveltejs/vite-plugin-svelte/commit/d2b4cf8d4b0f4e04481a92e0ce950434e693d356) | Within six months |
| `zimmerframe@1.1.5` | [2026-09-01](https://github.com/sveltejs/zimmerframe/commit/89c2d8dde680ab003a7f60b7dc3ea5a2008df1f9) | Within six months |
| `vitefu@1.1.3` | [2026-04-01](https://github.com/bluwy/vitefu/commit/e6f42533ca9af9f698970479d1c0c6dafd0dee89) | Older than six months — prototype exception approved |
| `obug@2.2.1` | [2026-09-18](https://github.com/sxzz/obug/commit/07dac0dc8fffcdf1a5bbe1aa442dbf6e5492e338) | Within six months |
| `deepmerge@4.3.1` | [2026-09-24](https://github.com/TehShrike/deepmerge/commit/d774850765a872f0cd642b0c87fc7e841af71f4c) | Within six months |
| `fdir@6.5.0` | [2025-08-16](https://github.com/thecodrr/fdir/commit/790c182eb54809ea7bc14f1bcb40acce89506bb4) | Older than six months — prototype exception approved |
| `vite@8.3.3` | [2026-10-08](https://github.com/vitejs/vite/commit/3d674869a50a0bca84a8cc8dc55bd35d57100e37) | Within six months |

Maintenance is checked against repository activity, not npm publish dates. This table records every package in the lockfile, including optional native packages and both locked versions of `magic-string`.

No implementation dependency is approved solely by being present in this prototype. The eventual application task must apply the dependency policy agreed by the operator.

## Production Svelte UI

Checked on **2026-10-08** for `web/`. The production lockfile is identical to the prototype's
lockfile apart from the root package name (verified by parsed JSON comparison). The complete
version/platform package table above therefore also enumerates the production set, with the
same cited maintenance evidence. Direct dependencies remain `svelte@5.57.2`,
`@sveltejs/vite-plugin-svelte@7.3.1`, and `vite@8.3.3`; their cited upstream commits were
rechecked on this date. No test dependency is added: tests use Node's built-in runner.
`npm ci` reports zero known vulnerabilities. Only compiled JS/CSS ships in the Go binary;
Node and build modules are absent from the scratch runtime. Build infrastructure adds the
official `node:24-alpine` stage and `actions/setup-node@v4`.

The twelve older transitive packages remain explicitly listed above. The production exception
is pending the operator's answer through the Integrator on `tasks:4f6ecdb55a9238e9`; the
prototype exception does not authorize production integration. The task permits implementation
with this pinned set while that decision is pending, and `build` must wait for approval.
