# Copyright IBM Corp. 2020, 2025
# SPDX-License-Identifier: MPL-2.0

FROM gcr.io/distroless/static-debian12 as root

COPY go-plugin-counter /bin/go-plugin-counter

ENTRYPOINT [ "/bin/go-plugin-counter" ]

# This is docker.mirror.hashicorp.services/ubuntu:24.04 as of 2024-07-31.
FROM docker.mirror.hashicorp.services/ubuntu@sha256:2e863c44b718727c860746568e1d54afd13b2fa71b160f5cd9058fc436217b30 as nonroot

COPY go-plugin-counter /bin/go-plugin-counter

RUN apt-get update && apt-get install -y libcap2-bin acl && \
    groupadd --system nonroot && \
    useradd --system -g nonroot nonroot && \
    chown -R nonroot:nonroot /bin/go-plugin-counter && \
    cp /bin/go-plugin-counter /bin/go-plugin-counter-mlock && \
    setcap cap_ipc_lock=+ep /bin/go-plugin-counter-mlock

USER nonroot

ENTRYPOINT [ "/bin/go-plugin-counter" ]

FROM nonroot as nonroot-mlock

ENTRYPOINT [ "/bin/go-plugin-counter-mlock" ]

# Set root as the default image.
FROM root
