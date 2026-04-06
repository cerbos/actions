import { isBuiltin } from "node:module";

import { defineConfig, type RolldownOptions } from "rolldown";

const common = {
  external: isBuiltin,
  platform: "node",
} satisfies RolldownOptions;

export default defineConfig([
  {
    ...common,
    input: "src/install-tools/index.ts",
    output: {
      file: "../../install-tools/action.mjs",
    },
  },
]);
