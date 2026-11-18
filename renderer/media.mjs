import { mkdir, readFile, readdir, writeFile, stat } from "node:fs/promises";
import path from "node:path";
import { pathToFileURL } from "node:url";

export const gravatar = "https://www.gravatar.com/avatar/05dfd4b41340d09cae045235eb0893c3?d=mp";

async function files(root) {
  const found = [];
  for (const item of await readdir(root, { withFileTypes: true })) {
    const file = path.join(root, item.name);
    if (item.isDirectory()) found.push(...(await files(file)));
    else if (/\.(vue|js)$/.test(item.name)) found.push(file);
  }
  return found;
}

export async function bundleMedia(showcase, output) {
  const prefix = "https://primefaces.org/cdn/primevue/";
  const urls = new Set();
  for (const file of [
    ...(await files(path.join(showcase, "doc"))),
    ...(await files(path.join(showcase, "service"))),
  ]) {
    const code = await readFile(file, "utf8");
    for (const match of code.matchAll(
      /https:\/\/primefaces\.org\/cdn\/primevue\/[a-zA-Z0-9_./-]+\.(?:png|jpg|jpeg|svg|webp|mp4)/g,
    )) {
      urls.add(match[0]);
    }
  }
  const { ProductService } = await import(
    pathToFileURL(path.join(showcase, "service/ProductService.js"))
  );
  for (const product of ProductService.getProductsData())
    urls.add(prefix + "images/product/" + product.image);
  const { CustomerService } = await import(
    pathToFileURL(path.join(showcase, "service/CustomerService.js"))
  );
  for (const customer of CustomerService.getData())
    urls.add(prefix + "images/avatar/" + customer.representative.image);
  // Paginator's TemplateDoc interpolates these twelve files at runtime.
  for (let i = 1; i <= 12; i++) urls.add(prefix + `images/nature/nature${i}.jpg`);
  urls.add(gravatar);
  const queue = [...urls].sort();
  const manifest = [];
  // Cache downloaded assets outside the disposable renderer build directory.
  const cache = path.resolve(".cache/media");
  await Promise.all(
    Array.from({ length: 8 }, async () => {
      for (;;) {
        const url = queue.shift();
        if (!url) return;
        const relative = url === gravatar ? "images/avatar/gravatar.png" : url.slice(prefix.length);
        if (relative.split("/").includes("..")) throw new Error("Unsafe media path");
        const cached = path.join(cache, relative);
        let data;
        try {
          data = await readFile(cached);
        } catch (error) {
          if (error.code !== "ENOENT") throw error;
          const response = await fetch(url, { signal: AbortSignal.timeout(30000) });
          if (!response.ok) throw new Error(`Media download failed: ${response.status} ${url}`);
          data = Buffer.from(await response.arrayBuffer());
          if (data.length > 25 * 1024 * 1024) throw new Error(`Media too large: ${url}`);
          await mkdir(path.dirname(cached), { recursive: true });
          await writeFile(cached, data);
        }
        const destination = path.join(output, "media", relative);
        await mkdir(path.dirname(destination), { recursive: true });
        await writeFile(destination, data);
        manifest.push({ url, file: "media/" + relative, bytes: data.length });
      }
    }),
  );
  manifest.sort((a, b) => a.url.localeCompare(b.url));
  await writeFile(
    path.join(output, "media-manifest.json"),
    JSON.stringify(manifest, null, 2) + "\n",
  );
  console.log(`Bundled ${manifest.length} demo media files.`);
}

export async function bundleLicenses(output) {
  const packages = [];
  for (const entry of await readdir("node_modules", { withFileTypes: true })) {
    if (entry.name.startsWith(".")) continue;
    if (entry.name.startsWith("@")) {
      for (const name of await readdir(path.join("node_modules", entry.name)))
        packages.push(path.join("node_modules", entry.name, name));
    } else packages.push(path.join("node_modules", entry.name));
  }
  const texts = [];
  for (const dir of packages.sort()) {
    if (!(await stat(dir)).isDirectory()) continue;
    const pkg = JSON.parse(await readFile(path.join(dir, "package.json"), "utf8"));
    const licenses = (await readdir(dir)).filter((name) =>
      /^licen[sc]e(?:[.-]|$)|^copying(?:[.-]|$)/i.test(name),
    );
    for (const file of licenses) {
      if (!(await stat(path.join(dir, file))).isFile()) continue;
      texts.push(
        `${pkg.name}@${pkg.version}\n${file}\n${await readFile(path.join(dir, file), "utf8")}`,
      );
    }
  }
  await writeFile(
    path.join(output, "THIRD_PARTY_LICENSES.txt"),
    texts.join("\n\n--------------------\n\n"),
  );
}
