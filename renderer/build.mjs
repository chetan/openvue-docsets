import { cp, mkdir, readFile, readdir, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { execFileSync } from "node:child_process";
import { build } from "vite";
import vue from "@vitejs/plugin-vue";
import AutoImport from "unplugin-auto-import/vite";
import tailwindcss from "tailwindcss";
import { createRequire } from "node:module";
import { featureModule } from "./source.mjs";
import { bundleMedia, bundleLicenses, gravatar } from "./media.mjs";
const primeui = createRequire(import.meta.url)("tailwindcss-primeui");

const root = path.resolve(fileURLToPath(new URL("..", import.meta.url)));
const source = path.resolve(process.env.OPENVUE_SOURCE || ".cache/openvue");
const showcase = path.join(source, "apps/showcase");
const generated = path.join(root, ".cache/renderer-source");
const output = path.resolve(process.env.EXAMPLES_OUTPUT || ".cache/examples");
await mkdir(generated, { recursive: true });
const pages = [];
for (const slug of (await readdir(path.join(showcase, "pages"))).sort()) {
  let text;
  try {
    text = await readFile(path.join(showcase, "pages", slug, "index.vue"), "utf8");
  } catch (error) {
    if (error.code === "ENOENT" || error.code === "ENOTDIR") continue;
    throw error;
  }
  const feature = featureModule(text);
  if (!feature) continue;
  await writeFile(path.join(generated, `${slug}.js`), feature.code);
  pages.push({ slug, features: feature.features });
}
if (!pages.some((page) => page.slug === "datepicker"))
  throw new Error("Missing DatePicker features");
await writeFile(
  path.join(generated, "registry.js"),
  pages.map(({ slug }, i) => `import p${i} from './${slug}.js';`).join("\n") +
    "\nexport default {" +
    pages.map(({ slug }, i) => `${JSON.stringify(slug)}:p${i}`).join(",") +
    "};\n",
);

// Only literal asset paths are rebased. Templates, component props and demo
// logic remain upstream-owned. Dynamic media paths use the same local prefix.
const localAssets = {
  name: "docset-local-assets",
  enforce: "pre",
  transform(code, id) {
    if (!id.startsWith(showcase) || !/\.(vue|js|ts)(\?|$)/.test(id)) return;
    return code
      .replaceAll("https://primefaces.org/cdn/primevue/", "./media/")
      .replaceAll(gravatar, "./media/images/avatar/gravatar.png")
      .replaceAll('"/demo/', '"./demo/')
      .replaceAll("'/demo/", "'./demo/");
  },
};
await build({
  configFile: false,
  root,
  base: "./",
  publicDir: false,
  define: { "process.env.NODE_ENV": JSON.stringify("production") },
  resolve: {
    alias: {
      "@": showcase,
      "/service": path.join(showcase, "service"),
      "#registry": path.join(generated, "registry.js"),
    },
    dedupe: ["vue", "openvue", "@openvue/core", "@openvue/themes"],
    extensions: [".mjs", ".js", ".ts", ".vue", ".json"],
  },
  plugins: [
    localAssets,
    vue({ template: { transformAssetUrls: { tags: {}, includeAbsolute: false } } }),
    AutoImport({
      imports: [
        "vue",
        {
          "openvue/usetoast": ["useToast"],
          "openvue/useconfirm": ["useConfirm"],
          "openvue/usedialog": ["useDialog"],
          "openvue/config": ["usePrimeVue"],
        },
      ],
      dts: false,
    }),
  ],
  css: {
    postcss: {
      plugins: [
        tailwindcss({
          content: [
            path.join(showcase, "doc/**/*.{vue,js}"),
            path.join(root, "renderer/**/*.{js,mjs,vue}"),
          ],
          darkMode: ["selector", ".p-dark"],
          plugins: [primeui],
          theme: { screens: { sm: "576px", md: "768px", lg: "992px", xl: "1200px" } },
        }),
      ],
    },
  },
  build: {
    outDir: output,
    emptyOutDir: true,
    target: "safari16",
    lib: {
      entry: path.join(root, "renderer/main.js"),
      name: "OpenVueDocset",
      formats: ["iife"],
      fileName: () => "examples.js",
      cssFileName: "examples",
    },
    rollupOptions: { output: { inlineDynamicImports: true } },
    chunkSizeWarningLimit: 15000,
  },
});
await cp(path.join(showcase, "public/demo"), path.join(output, "demo"), { recursive: true });
await bundleMedia(showcase, output);
await bundleLicenses(output);
const revision = execFileSync("git", ["-C", source, "rev-parse", "HEAD"], {
  encoding: "utf8",
}).trim();
await writeFile(
  path.join(output, "manifest.json"),
  JSON.stringify({ formatVersion: 2, revision, pages }, null, 2) + "\n",
);
console.log(`Bundled original examples for ${pages.length} OpenVue pages.`);
