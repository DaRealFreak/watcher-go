# internal/chrome — undetected browser sessions

Locate/launch a real Chrome and drive it over the DevTools Protocol (CDP) in a
way that stays invisible to CDP-based bot detection (PerimeterX/HUMAN,
Cloudflare, DataDome, …). Use it to get a real browser through a JavaScript
challenge or login, then hand the resulting cookies to the lightweight TLS HTTP
client for the actual scraping/downloading.

It exists because some sites can't be reached with a plain HTTP client at all:

- The anti-bot **sensor JavaScript must run in a real browser** to mint the
  trust cookie the site requires (e.g. PerimeterX's `_px`, Cloudflare's
  `cf_clearance`). No amount of TLS/HTTP fingerprint tuning produces it.
- That cookie is **bound to the browser session**, so it can't just be lifted
  into a fresh HTTP client — the browser has to complete the flow.

## Why "undetected": the `Runtime.enable` tell

Standard automation libraries (chromedp, Puppeteer, Playwright, Selenium) call
the CDP **`Runtime.enable`** command at startup. Enabling the Runtime domain has
a JS-observable side effect (the browser serializes console/exception objects for
the attached client), which anti-bot vendors trap to detect automation. On
DeviantArt this produced a hard block on the login submit:

> "Access to this page has been denied because we believe you are using
> automation tools to browse the website."

`chromedp` **and** a stealthed Playwright **and** real UI clicks all tripped it.

The fix (what tools like `nodriver` do) is to **never call `Runtime.enable`**.
This driver follows a few rules:

1. **Never issue `Runtime.enable` or any `*.enable`.** Run JS with
   `Runtime.evaluate` against the page's *default* execution context; read
   cookies with `Network.getCookies`; navigate with `Page.navigate`. All are
   plain commands that work without enabling their domains.
2. **Launch without `--enable-automation`** and with
   `--disable-blink-features=AutomationControlled`, so `navigator.webdriver`
   stays `false`.
3. Use a **real desktop User-Agent** and a **fresh throwaway profile**.
4. Harvest the trust/session cookies via `Network.getCookies` (it also sees
   HttpOnly cookies, which `document.cookie` cannot) and hand them to the TLS
   client.

## API

- `Find(configPath) (string, error)` — resolve a browser: configured path →
  installed Chrome/Chromium/Edge → cached Chrome-for-Testing → download it.
- `Launch(LaunchOptions) (*Instance, error)` — start Chrome with a debug port and
  a throwaway profile (no `--enable-automation`). `Instance.Close()` kills it.
- `NewSession(SessionOptions) (*Session, error)` — `Find` + `Launch` + connect
  CDP. The high-level entry point. `Session` methods:
  - `Eval(expr, out)` — run JS in the default context (awaits promises).
  - `Navigate(url)` — load a URL and wait for `document.readyState == complete`.
  - `Cookies(url) ([]*fhttp.Cookie, error)` — cookies (incl. HttpOnly) for a URL.
  - `WaitFor(boolExpr, timeout)` — poll a JS predicate until true.
  - `WaitForCookie(name, url, timeout)` — poll until a cookie appears.
  - `Close()` — terminate the browser and clean up.

## Adding a new site

### Pattern A — JavaScript challenge / clearance cookie (e.g. Cloudflare)

For sites that gate content behind a passive JS challenge that resolves into a
clearance cookie:

```go
session, err := chrome.NewSession(chrome.SessionOptions{
    InitialURL: "https://protected.example.com/",
    Headless:   true,
})
if err != nil {
    return err
}
defer session.Close()

// wait for the challenge to clear (Cloudflare sets cf_clearance on success)
if err := session.WaitForCookie("cf_clearance", "https://protected.example.com/", 30*time.Second); err != nil {
    return err // challenge did not clear (interactive captcha, bad IP, ...)
}

cookies, err := session.Cookies("https://protected.example.com/")
// -> inst.GetClient().SetCookies(u, cookies) on your TLS session, then scrape
```

### Pattern B — form login (e.g. PerimeterX / DeviantArt)

For sites where you must submit credentials through a protected endpoint, run the
login flow with same-origin `fetch` inside the page (so it carries the trust
cookie), then confirm and harvest. See
[`../modules/deviantart/browserlogin`](../modules/deviantart/browserlogin) for a
complete example. Sketch:

```go
session, _ := chrome.NewSession(chrome.SessionOptions{InitialURL: loginURL})
defer session.Close()
time.Sleep(5 * time.Second) // let the sensor run and mint its trust cookie

var result struct{ OK bool `json:"ok"` }
_ = session.Eval(loginFlowJS(user, pass), &result) // fetch-based login in-page
cookies, _ := session.Cookies("https://site.example.com")
```

### Reusing the cookies

Feed the harvested cookies to the TLS session's jar
(`session.GetClient().SetCookies(url, cookies)`), persist them (e.g. the
`cookies` table) for reuse, and validate with a site-specific "am I authenticated"
check before relaunching a browser. Only launch the browser when the persisted
cookies no longer validate.

## Limitations

- **Cat-and-mouse.** Anti-bot vendors change detection; a technique that passes
  today may need updates. Keep the driver minimal (fewer CDP calls = fewer tells).
- **Interactive challenges can't be auto-solved.** Passive JS challenges clear on
  their own; press-and-hold / interactive Turnstile / image captchas do not. For
  those, run with `Headless: false` so a human can solve it once, then harvest.
- **IP reputation matters.** Datacenter/VPN IPs raise the risk score; some sites
  need residential IPs even with a real browser.
- **Headless variance.** Most sites accept new headless Chrome; if one doesn't,
  `Headless: false` is the fallback.
- **Process lifecycle.** `Session.Close()` (deferred) kills Chrome and removes
  the temp profile. A hard crash of the host process can orphan a Chrome child.

## Research sources

- Scrapfly — [How to Bypass PerimeterX/HUMAN when Web Scraping](https://scrapfly.io/blog/posts/how-to-bypass-perimeterx-human-anti-scraping)
- Castle — [From Puppeteer stealth to Nodriver: how anti-detect frameworks evolved](https://blog.castle.io/from-puppeteer-stealth-to-nodriver-how-anti-detect-frameworks-evolved-to-evade-bot-detection/)
- DataDome — [How New Headless Chrome & the CDP Signal Are Impacting Bot Detection](https://datadome.co/threat-research/how-new-headless-chrome-the-cdp-signal-are-impacting-bot-detection/)
- ZenRows — [How to Bypass PerimeterX (HUMAN Security)](https://www.zenrows.com/blog/perimeterx-bypass)
- Kameleo — [Bypass Runtime.enable detection](https://kameleo.io/blog/bypass-runtime-enable-with-kameleos-undetectable-browser)
- Brotector (CDP-detection test suite) — https://github.com/kaliiiiiiiiii/brotector
