# Publishing

`project-toolkit` publishes to GitHub Packages only.

## Consumer Install

```bash
npm config set @dotnaos:registry https://npm.pkg.github.com
export NODE_AUTH_TOKEN=YOUR_GITHUB_TOKEN
npm install @dotnaos/project-toolkit
```

`NODE_AUTH_TOKEN` should be a GitHub token that can read packages.

## Release Trigger

Publishing is automated by `.github/workflows/publish.yml`.

The workflow runs only when a version tag matching `v*.*.*` is pushed.

Example:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The workflow checks that the tag matches `package.json`, installs dependencies, builds the CLI, and publishes to `https://npm.pkg.github.com` using `secrets.GITHUB_TOKEN`.

## Local Verification

```bash
npm install
npm run build
npm pack --dry-run
npm publish --dry-run
```
