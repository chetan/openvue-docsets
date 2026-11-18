import { test } from "node:test";
import assert from "node:assert/strict";
import { featureModule } from "./source.mjs";

test("extract original feature definitions and only their imports", () => {
  const result = featureModule(`<template><DocComponent /></template><script>
import BasicDoc from '@/doc/basic/BasicDoc.vue';
import Playground from '@/doc/basic/Playground.vue';
export default { data() { return { docs: [{id: 'basic',label:'Basic',component:BasicDoc}], playground:Playground }; } };
</script>`);
  assert.match(result.code, /import BasicDoc/);
  assert.doesNotMatch(result.code, /import Playground/);
  assert.deepEqual(result.features, [{ id: "basic", label: "Basic" }]);
});

test("do not invent demos for pages without a feature list", () => {
  assert.equal(featureModule("<template>Guide</template>"), null);
});
