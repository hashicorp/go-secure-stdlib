Stdlib for HashiCorp Secure products
=================

These libraries are maintained by engineers in the HashiCorp's Secure division
as a stdlib for its projects -- Vault, Vault plugins, Boundary, etc. -- to
reduce code duplication and increase consistency.

Each library is its own Go module, although some of them may have dependencies
on others within the repo. The libraries follow Go module versioning rules.

Most of the libraries in here were originally pulled from
vault/helper/metricsutil, vault/sdk/helper, and vault/internalshared; see there
for contribution and change history prior to their move here.

All modules are licensed according to MPLv2 as contained in the LICENSE file;
this file is duplicated in each module.
