# CI Layout Notes

This repository keeps packaging assets under `build/package` and documents CI concerns under `build/ci`.

The executable GitHub Actions workflow remains in `.github/workflows/ci.yml` because GitHub requires that exact location.

If this template later adds reusable CI helper scripts, they should live under `build/ci` and be invoked from the workflow.