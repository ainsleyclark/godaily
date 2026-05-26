# TODO

## Digest Service

pkg/services/digest

- Some of the stuff in helpers_test.go we don't need, such as mockSlack, there are already
  generated mock stubs for this, as well asc mockEmail? It should all be in the mocks package, let
  me know if not.
- I don't want a separate recap service, it should be part of the main digest one.
