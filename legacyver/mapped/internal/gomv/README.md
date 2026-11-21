# Embedded go-multiversion

The packages here are copied from [shawtymarco/go-multiversion](https://github.com/shawtymarco/go-multiversion)
at commit `f4ba8291d1bd` (MIT, see LICENSE), trimmed to what protocols 419, 475 and 486 use:

- `protocols/v1_16_100`, `v1_18_0`, `v1_18_10`: the adapters, with their data in `data/v419`, `v475`, `v486`.
- `protocols/v1_26_20`, `v1_26_45`, `v1_26_50`: only their wire codecs. The 1.16 and 1.18 adapters build on
  `v1_26_20.New()`, which never loads registries, so those adapters' registry, recipe, creative and report code
  and their data (`data/v975`, `v1001`, `v2169`, `v2193`) are not included. `v1_26_20`'s `newRuntimeData`
  returns an error instead.
- `mapping`, `internal/itemconv`, `internal/packetconv`, `internal/packetio`.

Also removed: tests, the `legacytransport`-tagged 1.16 encryption file, the unused `legacy_states.nbt` files and
their loaders, and every function that `deadcode -test ./legacyver/mapped/` reported unreachable (plus the types
and fields `staticcheck -checks U1000` then reported unused). Otherwise only import paths were changed.

To update, copy the same packages from a newer commit, rewrite `github.com/shawtymarco/go-multiversion` to
`github.com/akmalfairuz/legacy-version/legacyver/mapped/internal/gomv`, repeat the trimming above, and run the
tests in `legacyver/mapped`.
