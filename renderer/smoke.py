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
                page.wait_for_load_state("networkidle")
                expect(page.locator("html")).to_have_attribute("data-examples-ready", "true")
                assert not page.locator(".example-error").count(), page.locator(".example-error").all_text_contents()
                assert not errors, errors
                assert not failed, failed
                assert not remote, remote

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
                    page.goto(file.as_uri(), wait_until="networkidle")
                    expect(page.locator("html")).to_have_attribute("data-examples-ready", "true")
                    assert not page.locator(".example-error").count(), (file.name, page.locator(".example-error").all_text_contents())
                    assert not errors, (file.name, errors)
                    assert not failed, (file.name, failed)
                    assert not remote, (file.name, remote)
                    checked += 1
            report[engine] = {"checks": checked, "network_requests": 0, "errors": 0}
            context.close()
            browser.close()
    print(json.dumps(report, indent=2))


if __name__ == "__main__":
    run()
