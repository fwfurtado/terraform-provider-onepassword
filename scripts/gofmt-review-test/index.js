import fs from "node:fs";
import path from "node:path";
import process from "node:process";

import { Octokit } from "@octokit/core";
import { MockAgent, setGlobalDispatcher } from "undici";

const diffPath = process.env.GOFMT_DIFF_FILE || process.argv[2];
if (!diffPath) {
  console.error("Usage: GOFMT_DIFF_FILE=path/to/diff node index.js");
  process.exit(1);
}

const diff = fs.readFileSync(path.resolve(diffPath), "utf8");
if (!diff.trim()) {
  console.log("No diff to process.");
  process.exit(0);
}

const owner = process.env.GH_OWNER || "example";
const repo = process.env.GH_REPO || "example-repo";
const pullNumber = Number(process.env.GH_PR_NUMBER || 1);
const commitId = process.env.GH_COMMIT_SHA || "deadbeef";
const token = process.env.GITHUB_TOKEN || "test-token";

const comments = parseDiff(diff);

const mockAgent = new MockAgent();
mockAgent.disableNetConnect();
setGlobalDispatcher(mockAgent);

const mockPool = mockAgent.get("https://api.github.com");
const intercept = {
  path: `/repos/${owner}/${repo}/pulls/${pullNumber}/comments`,
  method: "POST",
};

for (let i = 0; i < comments.length; i++) {
  mockPool.intercept(intercept).reply(201, async ({ body }) => {
    const raw =
      typeof body === "string"
        ? body
        : Buffer.isBuffer(body)
        ? body.toString("utf8")
        : body
        ? Buffer.from(body).toString("utf8")
        : "";
    const requestBody = raw ? JSON.parse(raw) : {};
    console.log("Intercepted payload:");
    console.log(JSON.stringify(requestBody, null, 2));
    return { id: 1 };
  });
}

const octokit = new Octokit({
  auth: token,
});

for (const comment of comments) {
  await octokit.request("POST /repos/{owner}/{repo}/pulls/{pull_number}/comments", {
    owner,
    repo,
    pull_number: pullNumber,
    ...comment,
  });
}

console.log(`Generated ${comments.length} comment payload(s).`);

function parseDiff(diffText) {
  let filePath = null;
  let oldLine = 0;
  let newLine = 0;
  let hunkActive = false;
  let removedLines = [];
  let addedLines = [];
  let hunkOldStart = null;
  let hunkNewStart = null;
  let firstRemovedLine = null;
  let firstAddedLine = null;
  const results = [];

  const flushHunk = () => {
    if (!filePath || !hunkActive) {
      return;
    }

    const leftLine = firstRemovedLine ?? hunkOldStart;
    if (removedLines.length > 0 && leftLine !== null) {
      results.push({
        commit_id: commitId,
        path: filePath,
        line: leftLine,
        side: "LEFT",
        body: `gofmt:\n${removedLines.join("\n")}`,
      });
    }

    const rightLine = firstAddedLine ?? hunkNewStart;
    if (addedLines.length > 0 && rightLine !== null) {
      results.push({
        commit_id: commitId,
        path: filePath,
        line: rightLine,
        side: "RIGHT",
        body: `gofmt:\n${addedLines.join("\n")}`,
      });
    }

    removedLines = [];
    addedLines = [];
    hunkOldStart = null;
    hunkNewStart = null;
    firstRemovedLine = null;
    firstAddedLine = null;
    hunkActive = false;
  };

  for (const line of diffText.split("\n")) {
    if (line.startsWith("diff ")) {
      flushHunk();
      filePath = null;
      oldLine = 0;
      newLine = 0;
      continue;
    }

    if (line.startsWith("+++ ")) {
      filePath = line.replace("+++ ", "").trim();
      if (filePath.startsWith("a/") || filePath.startsWith("b/")) {
        filePath = filePath.slice(2);
      }
      continue;
    }

    const hunk = /^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/.exec(line);
    if (hunk) {
      flushHunk();
      oldLine = Number(hunk[1]);
      newLine = Number(hunk[2]);
      hunkOldStart = oldLine;
      hunkNewStart = newLine;
      firstRemovedLine = null;
      firstAddedLine = null;
      hunkActive = true;
      continue;
    }

    if (!filePath || line.startsWith("--- ")) {
      continue;
    }

    if (line.startsWith("+") && !line.startsWith("+++")) {
      addedLines.push(line);
      if (firstAddedLine === null) {
        firstAddedLine = newLine;
      }
      newLine++;
      continue;
    }

    if (line.startsWith("-") && !line.startsWith("---")) {
      removedLines.push(line);
      if (firstRemovedLine === null) {
        firstRemovedLine = oldLine;
      }
      oldLine++;
      continue;
    }

    if (line.startsWith(" ")) {
      oldLine++;
      newLine++;
    }
  }

  flushHunk();

  return results;
}
