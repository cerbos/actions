import { type ChildProcess, spawn } from "node:child_process";
import { createHash } from "node:crypto";
import { createWriteStream } from "node:fs";
import { mkdir } from "node:fs/promises";
import { arch, platform as os } from "node:os";
import { resolve } from "node:path";
import {
  PassThrough,
  Readable,
  Transform,
  type TransformCallback,
} from "node:stream";
import { pipeline } from "node:stream/promises";

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

type Manifest = typeof manifest;
type Tool = keyof Manifest;

interface Source {
  tool: Tool;
  version: string;
  url: string;
  digest: string;
  extract?: string;
  postInstall: string[];
}

async function run(): Promise<void> {
  try {
    await installTools(getMultilineInput("tools").map(sourceFromManifest));
  } catch (error) {
    let message = "Failed to install tools";

    while (error instanceof Error) {
      message += `:\n${error.message}`;
      error = error.cause;
    }

    setFailed(message);
  }
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

  await Promise.all(
    sources.map(async (source) => {
      try {
        await installTool(source, controller.signal);
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

  if (!path) {
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
    const [response, path] = await Promise.all([
      fetch(source.url, { signal }),
      createDirectory(source),
    ]);

    if (!response.ok) {
      throw new Error(`GET ${source.url}: HTTP ${response.status}`);
    }

    if (!response.body) {
      throw new Error(`GET ${source.url}: missing response body`);
    }

    const hash = createHash("sha256");

    await pipeline(
      Readable.fromWeb(response.body),
      async function* (source: AsyncIterable<Buffer>) {
        for await (const chunk of source) {
          hash.update(chunk);
          yield chunk;
        }
      },
      createExtractStream(source, signal),
      createWriteStream(resolve(path, source.tool), {
        flags: "wx",
        mode: 0o777,
      }),
      { signal },
    );

    const digest = `sha256:${hash.digest("hex")}`;

    if (digest !== source.digest) {
      throw new Error("Digest mismatch");
    }

    return path;
  } catch (error) {
    throw new Error(`Failed to download tool "${source.tool}"`, {
      cause: error,
    });
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

function createExtractStream(
  { extract }: Source,
  signal: AbortSignal,
): Transform {
  return extract ? new ExtractStream(extract, signal) : new PassThrough();
}

class ExtractStream extends Transform {
  private readonly process: ChildProcess;
  private stdoutEnded = false;

  public constructor(extract: string, signal: AbortSignal) {
    super();

    const emitError: (error: Error) => void = this.emit.bind(this, "error");

    this.process = spawn(
      "tar",
      ["--extract", "--gzip", "--to-stdout", extract],
      {
        signal,
        stdio: ["pipe", "pipe", "inherit"],
      },
    );

    this.process
      .on("close", (code, signal) => {
        if (code !== 0) {
          const status = code ? `code ${code}` : `signal ${signal}`;
          emitError(new Error(`tar exited with ${status}`));
        }
      })
      .on("error", emitError);

    this.process.stdin?.on("error", emitError);

    this.process.stdout
      ?.on("data", this.push.bind(this))
      .on("end", () => {
        this.stdoutEnded = true;
      })
      .on("error", emitError);
  }

  public override _transform(
    chunk: unknown,
    encoding: BufferEncoding,
    callback: TransformCallback,
  ): void {
    this.process.stdin?.write(chunk, encoding, callback);
  }

  public override _flush(callback: TransformCallback): void {
    this.process.stdin?.end();

    if (this.stdoutEnded) {
      callback();
    } else {
      this.process.stdout?.once("end", callback);
    }
  }
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

await run();
