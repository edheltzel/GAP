# Atlas Config Glossary

- **Harness**: A coding-agent runtime managed by Atlas Config, such as Claude Code or Pi.
- **Selected harness**: A harness named in `harness.txt` as part of this configuration. `harness.txt` is the only source for harness selection.
- **Available harness**: A harness detected as installed on the current machine.
- **Eligible harness**: A harness that is both selected and available. Atlas attempts to install assets for eligible harnesses and reports unsupported capabilities at the end.
- **Global skill**: A universal skill named in `skills.txt`. Every global skill is installed through skills.sh for every selected harness.
- **Shared asset**: Reusable behavior stored once for use by more than one harness.
- **Harness wrapper**: The smallest harness-native file needed to expose a shared asset in that harness.
- **Managed link**: A direct symlink from a harness runtime folder to one Atlas-owned file or asset directory. Runtime folders themselves remain local directories.
