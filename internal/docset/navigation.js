const browser = document.querySelector(".component-browser");
if (browser) {
  const details = browser.querySelector("details");
  const list = browser.querySelector(".component-list");
  const current = list.querySelector('[aria-current="page"]');
  const narrow = matchMedia("(max-width: 47.99rem)");
  const revealCurrent = () => {
    if (details.open && current) {
      const delta = current.getBoundingClientRect().top - list.getBoundingClientRect().top;
      list.scrollTop += delta - list.clientHeight / 2 + current.clientHeight / 2;
    }
  };
  const adapt = () => {
    details.open = !narrow.matches;
    revealCurrent();
  };
  adapt();
  narrow.addEventListener("change", adapt);
  details.addEventListener("toggle", revealCurrent);
}
