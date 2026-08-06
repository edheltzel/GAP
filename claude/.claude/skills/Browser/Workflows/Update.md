# Update Workflow

## Voice notification

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running the Update workflow in the Browser skill to verify browser capabilities"}' \
  > /dev/null 2>&1 &
```

Running **Update** in **Browser**...

Verify the installed browser CLI without making network calls unless the user explicitly supplies and approves a target URL.

## Steps

```bash
BROWSER_CLI=/opt/homebrew/bin/chrome-devtools-axi
test -x "$BROWSER_CLI"
"$BROWSER_CLI" --help
"$BROWSER_CLI" update --check
```

For an approved local smoke target:

```bash
"$BROWSER_CLI" open http://127.0.0.1:<port>
"$BROWSER_CLI" snapshot
"$BROWSER_CLI" screenshot /tmp/browser-update-test.png
"$BROWSER_CLI" stop
```

Also confirm `Stories/*.yaml` and `Recipes/*.md` are readable. Do not run the mutating `update` command unless the user explicitly requests an upgrade.
