Libs shared by HashiCorp Secure products
=================

These libraries are used by Vault, Boundary, and possibly other HashiCorp
products to reduce code duplication and increase consistency. They are not
generally libraries needed by Vault plugins -- those are in Vault's `sdk` module
-- although plugins may use these libraries if they find them useful (especially
`strutil` and `parseutil`).

**There are no compatibility guarantees.** Things in here may change or move or
disappear at any time.

The libraries in here were originally pulled from vault/helper/metricsutil,
vault/sdk/helper, and vault/internalshared. See there for contribution and
change history prior to their move here.
