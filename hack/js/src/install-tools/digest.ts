import { createHash, type Hash } from "node:crypto";
import { Transform, type TransformCallback } from "node:stream";

export function createDigestStream(digest: string): Transform {
  return new DigestStream(digest);
}

class DigestStream extends Transform {
  private readonly hash: Hash;

  public constructor(private readonly digest: string) {
    super();
    this.hash = createHash("sha256");
  }

  public override _transform(
    chunk: unknown,
    encoding: BufferEncoding,
    callback: TransformCallback,
  ): void {
    this.push(chunk, encoding);
    this.hash.write(chunk, callback);
  }

  public override _flush(callback: TransformCallback): void {
    let error: Error | null = null;
    if (this.digest != `sha256:${this.hash.digest("hex")}`) {
      error = new Error("digest mismatch");
    }
    callback(error);
  }
}
