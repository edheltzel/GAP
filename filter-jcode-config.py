#!/usr/bin/env python3
"""Remove machine-local Jcode settings from config.toml before Git indexes it."""

import re
import sys


SECTION = re.compile(r"^\s*\[\[?([^\]]+)\]\]?\s*$")


def main() -> None:
    skipped_section = False
    output: list[str] = []

    for line in sys.stdin:
        match = SECTION.match(line)
        if match:
            section = match.group(1)
            skipped_section = section == "launch_hotkeys" or section.startswith(
                "launch_hotkeys."
            )
            if skipped_section:
                continue

        if skipped_section:
            continue

        if line.lstrip().startswith("theme ="):
            continue

        output.append(line)

    while output and not output[-1].strip():
        output.pop()
    rendered = "".join(output)
    sys.stdout.write(rendered)
    if rendered and not rendered.endswith("\n"):
        sys.stdout.write("\n")


if __name__ == "__main__":
    main()
