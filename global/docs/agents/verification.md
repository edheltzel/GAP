# Verification / adversarial gate

Claim is not proof.

## Rules

- Reproduce bugs in an end-user-like path before fixing.
- Run the project's real tests / checks you can reach.
- Prefer an adversarial or second-pass review before raising a PR when that lane exists.
- Fix lint / test failures / flakiness you touch or clearly unblock; do not ignore red you introduced.
- Markdown rules alone are not a hard edge - hooks, CI, and sandboxes still matter.

## Done checklist

- [ ] Repro or reason it could not be reproduced
- [ ] Tests / checks run (name them)
- [ ] Review / adversarial step done or explicitly N/A
