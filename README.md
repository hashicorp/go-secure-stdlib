HashiCorp-internal libs
=================

Do not use these unless you know what you're doing.

These libraries are used by other HashiCorp products to reduce code duplication
and increase consistency. They are not libraries needed by Vault plugins --
those are in Vault's sdk/ module.

There are no compatibility guarantees. Things in here may change or move or
disappear at any time.

If you are a Vault plugin author and think you need a library in here in your
plugin, please open an issue in the Vault repository for discussion.

The libraries in here were originally pulled from vault/helper/metricsutil
(metricsutil) and vault/internalshared (the rest). See there for contribution
and change history prior to their move here.
