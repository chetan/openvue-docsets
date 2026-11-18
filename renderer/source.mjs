import { parse as parseSFC } from "@vue/compiler-sfc";
import { parse } from "@babel/parser";

function walk(node, visit) {
  if (!node || typeof node !== "object") return;
  visit(node);
  for (const value of Object.values(node)) {
    if (Array.isArray(value)) value.forEach((item) => walk(item, visit));
    else if (value && typeof value === "object") walk(value, visit);
  }
}

// Extract the website's feature list, not its Nuxt shell, playground or API tabs.
// Preserve the original demo imports and configuration; do not synthesize demos
// by executing Markdown snippets.
export function featureModule(source) {
  const { descriptor, errors } = parseSFC(source);
  if (errors.length) throw new Error(String(errors[0]));
  if (!descriptor.script) return null;
  const script = descriptor.script.content;
  const tree = parse(script, { sourceType: "module" });
  let docs;
  walk(tree, (node) => {
    if (
      node.type === "ObjectProperty" &&
      node.key.name === "docs" &&
      node.value.type === "ArrayExpression"
    ) {
      if (docs) throw new Error("Multiple feature lists in page");
      docs = node.value;
    }
  });
  if (!docs) return null;
  const identifiers = new Set();
  walk(docs, (node) => {
    if (node.type === "Identifier") identifiers.add(node.name);
  });
  const imports = tree.program.body.filter(
    (node) =>
      node.type === "ImportDeclaration" &&
      node.specifiers.some((specifier) => identifiers.has(specifier.local.name)),
  );
  const features = [];
  walk(docs, (node) => {
    if (node.type !== "ObjectExpression") return;
    const fields = Object.fromEntries(
      node.properties
        .filter((property) => property.type === "ObjectProperty")
        .map((property) => [property.key.name, property.value]),
    );
    if (fields.id?.type === "StringLiteral" && fields.label?.type === "StringLiteral") {
      features.push({ id: fields.id.value, label: fields.label.value });
    }
  });
  return {
    code:
      imports.map((node) => script.slice(node.start, node.end)).join("\n") +
      "\nexport default " +
      script.slice(docs.start, docs.end) +
      ";\n",
    features,
  };
}
