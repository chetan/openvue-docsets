"""Verify real docset pages over file:// with all network access disabled."""
import argparse
import json
from pathlib import Path
from playwright.sync_api import sync_playwright, expect


def run():
    parser = argparse.ArgumentParser()
    parser.add_argument("--docset", default="dist/OpenVue.docset")
    parser.add_argument("--all", action="store_true", help="Load every page with rendered examples")
    args = parser.parse_args()
    docs = Path(args.docset).resolve() / "Contents/Resources/Documents"
    report = {}
    with sync_playwright() as playwright:
        for engine in ["chromium", "webkit"]:
            browser = getattr(playwright, engine).launch(headless=True)
            context = browser.new_context(offline=True, viewport={"width": 1280, "height": 900})
            page = context.new_page()
            page.add_init_script("""window.removedDashAnchors = [];
                new MutationObserver(records => {
                    for (const record of records) for (const node of record.removedNodes) {
                        if (node.nodeType === 1 && (node.matches('.dashAnchor') || node.querySelector('.dashAnchor')))
                            window.removedDashAnchors.push(node.outerHTML.slice(0, 200));
                    }
                }).observe(document, { childList: true, subtree: true });""")
            errors, failed, remote = [], [], []
            page.on("pageerror", lambda error: errors.append(str(error)))
            page.on("console", lambda message: errors.append(message.text) if message.type == "error" else None)
            page.on("requestfailed", lambda request: failed.append(request.url))
            page.on("request", lambda request: remote.append(request.url) if request.url.startswith(("https:", "http:")) else None)

            def visit(slug):
                errors.clear()
                failed.clear()
                remote.clear()
                page.goto((docs / f"components-{slug}.html").as_uri())
                # file:// has no network idle contract. Wait for the document
                # and our explicit post-mount signal before inspecting demos.
                page.wait_for_load_state("load")
                expect(page.locator("html")).to_have_attribute("data-examples-ready", "true")
                assert not page.locator(".example-error").count(), page.locator(".example-error").all_text_contents()
                assert not errors, errors
                assert not failed, failed
                assert not remote, remote
                assert page.locator("a.dashAnchor").count() > 0
                assert page.evaluate("window.removedDashAnchors") == []
                current = page.locator('.component-list a[aria-current="page"]')
                expect(current).to_have_attribute("href", f"components-{slug}.html")

            visit("datepicker")
            basic = page.locator("#example-basic input")
            basic.click()
            calendar = page.get_by_role("dialog", name="Choose Date")
            expect(calendar).to_be_visible()
            calendar.get_by_text("15", exact=True).first.click()
            expect(basic).not_to_have_value("")
            expect(calendar).not_to_be_visible()
            expect(page.locator("#example-inline table").first).to_be_visible()
            page.locator("#example-basic").screenshot(path=f".cache/{engine}-datepicker-live.png")
            assert page.locator('a.dashAnchor[name="//apple_ref/cpp/Sample/Basic"]').count() == 1
            basic_anchor = page.locator('a.dashAnchor[name="//apple_ref/cpp/Sample/Basic"]')
            page.evaluate("(name) => { location.hash = name; }", basic_anchor.get_attribute("name"))
            expect(page.locator("#example-basic")).to_be_in_viewport()
            page.locator(".component-list").get_by_role("link", name="Dialog", exact=True).click()
            page.wait_for_load_state("load")
            expect(page.locator('.component-list a[aria-current="page"]')).to_have_text("Dialog")
            page.set_viewport_size({"width":600,"height":900})
            expect(page.locator(".component-browser details")).not_to_have_attribute("open","")
            page.locator(".component-browser summary").click()
            expect(page.locator(".component-browser details")).to_have_attribute("open","")
            page.locator(".component-list").get_by_role("link", name="DatePicker", exact=True).click()
            page.wait_for_load_state("load")
            expect(page.locator('.component-list a[aria-current="page"]')).to_have_text("DatePicker")
            page.set_viewport_size({"width":1280,"height":900})

            visit("button")
            button = page.locator("#example-basic button").first
            expect(button).to_be_visible()
            button.click()

            visit("checkbox")
            checkbox = page.locator("#example-basic input[type=checkbox]").first
            checkbox.check()
            expect(checkbox).to_be_checked()

            visit("dialog")
            page.locator("#example-basic button").first.click()
            expect(page.get_by_role("dialog")).to_be_visible()
            page.keyboard.press("Escape")
            expect(page.get_by_role("dialog")).not_to_be_visible()

            visit("datatable")
            assert page.locator("#example-basic tbody tr").count() > 1
            assert not page.locator(".example-error").count()
            checked = 5
            if args.all:
                for file in sorted(docs.glob("*.html")):
                    if 'id="rendered-examples"' not in file.read_text():
                        continue
                    errors.clear()
                    failed.clear()
                    remote.clear()
                    page.goto(file.as_uri(), wait_until="load")
                    expect(page.locator("html")).to_have_attribute("data-examples-ready", "true")
                    assert not page.locator(".example-error").count(), (file.name, page.locator(".example-error").all_text_contents())
                    assert not errors, (file.name, errors)
                    assert not failed, (file.name, failed)
                    assert not remote, (file.name, remote)
                    assert page.locator("a.dashAnchor").count() > 0, file.name
                    assert page.evaluate("window.removedDashAnchors") == [], file.name
                    checked += 1
            report[engine] = {"checks": checked, "network_requests": 0, "errors": 0}
            context.close()
            browser.close()
    print(json.dumps(report, indent=2))


if __name__ == "__main__":
    run()
