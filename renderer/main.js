import { createApp, defineComponent, h, onErrorCaptured, onMounted, ref } from "vue";
import * as OpenVueComponents from "openvue";
import OpenVue from "openvue/config";
import Aura from "@openvue/themes/aura";
import { Form, FormField } from "@openvue/forms";
import ToastService from "openvue/toastservice";
import ConfirmationService from "openvue/confirmationservice";
import DialogService from "openvue/dialogservice";
import { createRouter, createMemoryHistory } from "vue-router";
import registry from "#registry";
import "./style.css";
import "@openvue/openicons/primeicons.css";

const mount = document.querySelector("#rendered-examples");
if (mount) {
  const slug = mount.dataset.component;
  const docs = registry[slug];
  if (!docs) throw new Error(`No bundled examples for ${slug}`);
  const ErrorBoundary = defineComponent({
    setup(_, { slots }) {
      const failure = ref(null);
      onErrorCaptured((error) => {
        failure.value = error.message;
        console.error(error);
        return false;
      });
      return () =>
        failure.value
          ? h(
              "p",
              { class: "example-error", role: "alert" },
              `This upstream example could not run: ${failure.value}`,
            )
          : slots.default?.();
    },
  });
  const DocSections = defineComponent({
    props: ["docs"],
    setup(props) {
      return () =>
        props.docs.map((doc) =>
          h("section", { class: "live-section", id: `example-${doc.id}`, "data-example": doc.id }, [
            doc.children
              ? [h("h2", doc.label), h(DocSections, { docs: doc.children })]
              : h(ErrorBoundary, null, {
                  default: () =>
                    h(doc.component, {
                      id: doc.id,
                      label: doc.label,
                      data: doc.data,
                      description: doc.description,
                    }),
                }),
          ]),
        );
    },
  });
  const app = createApp({
    render: () => h(DocSections, { docs }),
  });
  app.use(OpenVue, {
    theme: {
      preset: Aura,
      options: {
        darkModeSelector: ".p-dark",
        cssLayer: { name: "openvue", order: "tailwind-base, openvue, tailwind-utilities" },
      },
    },
  });
  app.use(ToastService).use(ConfirmationService).use(DialogService);
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/:pathMatch(.*)*", component: { render: () => null } }],
  });
  app.use(router);
  for (const [name, component] of Object.entries(OpenVueComponents)) {
    if (
      component &&
      typeof component === "object" &&
      (component.render || component.setup || component.extends)
    )
      app.component(name, component);
  }
  for (const name of [
    "Ripple",
    "Tooltip",
    "BadgeDirective",
    "KeyFilter",
    "FocusTrap",
    "StyleClass",
    "AnimateOnScroll",
  ]) {
    const directive = OpenVueComponents[name];
    if (directive)
      app.directive(name === "BadgeDirective" ? "badge" : name.toLowerCase(), directive);
  }
  app.component("Form", Form).component("FormField", FormField);
  app.component("DocSections", DocSections);
  app.component("DocSectionText", {
    inheritAttrs: false,
    setup(_, { attrs, slots }) {
      return () => [
        h(attrs.level === 2 ? "h3" : "h2", attrs.label),
        h("div", { class: "doc-section-description" }, slots.default?.()),
      ];
    },
  });
  app.component("DocSectionCode", {
    props: ["code"],
    setup(props) {
      return () =>
        h("details", { class: "example-code" }, [
          h("summary", "Example source"),
          ...["basic", "composition", "options", "data"]
            .filter((key) => props.code?.[key])
            .map((key) => h("div", [h("h4", key), h("pre", [h("code", props.code[key])])])),
        ]);
    },
  });
  app.component("DeferredDemo", {
    emits: ["load"],
    props: ["options"],
    setup(_, { emit, slots }) {
      const loaded = ref(false);
      onMounted(() => {
        emit("load");
        loaded.value = true;
      });
      return () => (loaded.value ? slots.default?.() : h("p", "Loading example data…"));
    },
  });
  app.component("NuxtLink", {
    props: ["to"],
    setup(props, { attrs, slots }) {
      return () =>
        h(
          "a",
          {
            ...attrs,
            href:
              typeof props.to === "string" && props.to.startsWith("/")
                ? `https://openvue.dev${props.to}`
                : props.to,
          },
          slots.default?.(),
        );
    },
  });
  app.component("OpenVueNuxtLink", app.component("NuxtLink"));
  app.directive("code", {});
  app.mount(mount);
  document.documentElement.dataset.examplesReady = "true";
  if (location.hash.startsWith("#example-")) {
    requestAnimationFrame(() => document.getElementById(location.hash.slice(1))?.scrollIntoView());
  }
}
