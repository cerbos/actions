import { createReadStream, createWriteStream } from "node:fs";
import { mkdir } from "node:fs/promises";
import { arch, platform as os } from "node:os";
import { join, resolve } from "node:path";
import { Readable } from "node:stream";
import { pipeline } from "node:stream/promises";
import { setTimeout } from "node:timers/promises";

import {
  addPath,
  endGroup,
  getMultilineInput,
  info,
  setFailed,
  setOutput,
  startGroup,
} from "@actions/core";
import { exec } from "@actions/exec";
import {
  cacheDir as addToCache,
  find as findInCache,
} from "@actions/tool-cache";

import manifest from "../../../../toolbox.json" with { type: "json" };

import { type Archive, extract, inferFormat } from "./archive.js";
import { createDigestStream } from "./digest.js";

type Manifest = typeof manifest;
type Tool = keyof Manifest;

interface Source {
  tool: Tool;
  version: string;
  url: string;
  digests: {
    asset: string;
    binary: string;
  };
  extract?: string;
  postInstall: string[];
}

export async function run(): Promise<void> {
  try {
    await installTools(getMultilineInput("tools").map(sourceFromManifest));
  } catch (error) {
    setFailed(errorMessage("Failed to install tools", error));
  }
}

function errorMessage(message: string, error: unknown): string {
  while (error instanceof Error) {
    message += `:\n\t${error.message || error.toString()}`;
    error = error.cause;
  }

  return message;
}

const platform = `${os()}/${arch()}`;

function sourceFromManifest(tool: string): Source {
  validateTool(tool);
  const { version, downloads, postInstall } = manifest[tool];

  type Downloads = Manifest[Tool]["downloads"];
  type Download = Downloads[keyof Downloads];

  const download = (downloads as Record<string, Download>)[platform];
  if (!download) {
    throw new Error(`Unsupported platform "${platform}" for tool "${tool}"`);
  }

  return {
    tool,
    version,
    ...download,
    postInstall,
  };
}

function validateTool(tool: string): asserts tool is Tool {
  if (!(tool in manifest)) {
    throw new Error(`Unknown tool ${tool}`);
  }
}

async function installTools(sources: Source[]): Promise<void> {
  const controller = new AbortController();

  const signal = AbortSignal.any([
    controller.signal,
    AbortSignal.timeout(60_000),
  ]);

  await Promise.all(
    sources.map(async (source) => {
      try {
        await installTool(source, signal);
      } catch (error) {
        controller.abort(error);
        throw error;
      }
    }),
  );

  for (const source of sources) {
    info(`Installed ${source.tool} ${source.version}`);
    await postInstallTool(source);
    setOutput(source.tool, source.version);
  }
}

async function installTool(source: Source, signal: AbortSignal): Promise<void> {
  const key = `cerbos-toolbox-${source.tool}`;
  let path = findInCache(key, source.version);

  if (path) {
    await pipeline(
      createReadStream(join(path, source.tool)),
      createDigestStream(source.digests.binary),
    );
  } else {
    path = await addToCache(
      await downloadTool(source, signal),
      key,
      source.version,
    );
  }

  addPath(path);
}

async function downloadTool(
  source: Source,
  signal: AbortSignal,
): Promise<string> {
  try {
    const [responseBody, path] = await Promise.all([
      downloadWithRetries(source, signal),
      createDirectory(source),
    ]);

    const binary = createWriteStream(resolve(path, source.tool), {
      flags: "wx",
      mode: 0o777,
    });

    let target = binary;
    let archive: Archive | undefined;
    if (source.extract) {
      const format = inferFormat(source.url);

      archive = {
        format,
        path: `${binary.path}${format}`,
        extract: source.extract,
      };

      target = createWriteStream(archive.path, {
        flags: "wx",
        mode: 0o600,
      });
    }

    await pipeline(
      responseBody,
      createDigestStream(source.digests.asset),
      target,
      { signal },
    );

    if (archive) {
      await pipeline(
        extract(archive, signal),
        createDigestStream(source.digests.binary),
        binary,
      );
    }

    return path;
  } catch (error) {
    throw new Error(`Failed to download tool "${source.tool}"`, {
      cause: error,
    });
  }
}

async function downloadWithRetries(
  source: Source,
  signal: AbortSignal,
): Promise<Readable> {
  for (let attempt = 1; ; attempt++) {
    signal.throwIfAborted();

    try {
      return await download(source.url, signal);
    } catch (error) {
      console.error(
        errorMessage(
          `Failed to download tool "${source.tool}" (attempt ${attempt})`,
          error,
        ),
      );
    }

    await backoff(attempt, signal);
  }
}

async function backoff(attempt: number, signal: AbortSignal): Promise<void> {
  const initial = 500;
  const multiplier = 1.5;
  const jitter = 0.5 + Math.random();

  const delay = initial * multiplier ** attempt * jitter;

  await setTimeout(delay, undefined, { signal });
}

async function download(url: string, signal: AbortSignal): Promise<Readable> {
  try {
    const response = await fetch(url, { signal });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }

    if (!response.body) {
      throw new Error(`Missing response body`);
    }

    return Readable.fromWeb(response.body);
  } catch (error) {
    throw new Error(`GET ${url} failed`, { cause: error });
  }
}

async function createDirectory({ tool, version }: Source): Promise<string> {
  const tempDir = process.env["RUNNER_TEMP"];
  if (!tempDir) {
    throw new Error("Missing RUNNER_TEMP");
  }

  const path = resolve(tempDir, `cerbos-toolbox-${tool}-${version}`);
  await mkdir(path);
  return path;
}

async function postInstallTool({
  tool,
  postInstall: [command, ...args],
}: Source): Promise<void> {
  if (!command) {
    return;
  }

  startGroup(`Post-install ${tool}`);
  await exec(command, args);
  endGroup();
}
