# Local gofmt review comment test

This script parses `gofmt -d` output and uses `octokit.request` to create
review comment payloads. HTTP requests are intercepted with `undici`'s
MockAgent so you can
inspect payloads without talking to GitHub.

## Setup

```bash
cd scripts/gofmt-review-test
npm install
```

## Create a diff file

```bash
gofmt -d . > /tmp/gofmt.diff
```

## Run

```bash
GOFMT_DIFF_FILE=/tmp/gofmt.diff \
GH_OWNER=your-org \
GH_REPO=your-repo \
GH_PR_NUMBER=123 \
GH_COMMIT_SHA=deadbeef \
node index.js
```

The script will print the intercepted payloads for each comment.
