import { spawn } from "node:child_process";

const formats = [".tar.gz", ".zip"] as const;

export type Format = (typeof formats)[number];

export function inferFormat(url: string): Format {
  const format = formats.find((format) => url.endsWith(format));
  if (!format) {
    throw new Error(`Unknown archive format ${url}`);
  }
  return format;
}

export interface Archive {
  format: Format;
  path: string;
  extract: string;
}

export async function* extract(
  { format, path, extract }: Archive,
  signal: AbortSignal,
): AsyncGenerator {
  let command: string;
  let args: string[];

  switch (format) {
    case ".tar.gz":
      command = "tar";
      args = ["--extract", "--gzip", "--file", path, "--to-stdout", extract];
      break;

    case ".zip":
      command = "unzip";
      args = ["-p", path, extract];
  }

  const process = spawn(command, args, {
    signal,
    stdio: ["ignore", "pipe", "inherit"],
  });

  const exit = new Promise<Error | null>((resolve) => {
    process
      .on("close", (code, signal) => {
        if (code === 0) {
          resolve(null);
          return;
        }

        const status = code ? `code ${code}` : `signal ${signal}`;
        resolve(new Error(`${command} exited with ${status}`));
      })
      .on("error", resolve);
  });

  yield* process.stdout;

  const error = await exit;
  if (error) {
    throw error;
  }
}
