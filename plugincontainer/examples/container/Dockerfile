# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

FROM gcr.io/distroless/static-debian12 AS root

COPY go-plugin-counter /bin/go-plugin-counter

ENTRYPOINT [ "/bin/go-plugin-counter" ]

FROM docker.mirror.hashicorp.services/ubuntu:22.04 AS nonroot

COPY go-plugin-counter /bin/go-plugin-counter
COPY go-plugin-counter /bin/go-plugin-counter-root
COPY go-plugin-counter /bin/go-plugin-counter-runsc
COPY go-plugin-counter /bin/go-plugin-counter-mlock
COPY go-plugin-counter /bin/go-plugin-counter-mlock-runsc

RUN apt-get update && apt-get install -y libcap2-bin acl adduser file && \
    addgroup --system nonroot && \
    adduser --system --ingroup nonroot nonroot && \
    chown -R nonroot:nonroot /bin/go-plugin-counter /bin/go-plugin-counter-runsc && \
    setcap cap_dac_override=+ep /bin/go-plugin-counter-root && \
    setcap cap_dac_override=+ep /bin/go-plugin-counter-runsc && \
    setcap cap_ipc_lock=+ep /bin/go-plugin-counter-mlock && \
    setcap cap_dac_override,cap_ipc_lock=+ep /bin/go-plugin-counter-mlock-runsc

USER nonroot

ENTRYPOINT [ "/bin/go-plugin-counter" ]

FROM nonroot AS root-runsc

ENTRYPOINT [ "/bin/go-plugin-counter-root" ]

FROM nonroot AS nonroot-mlock

ENTRYPOINT [ "/bin/go-plugin-counter-mlock" ]

FROM nonroot AS nonroot-runsc

ENTRYPOINT [ "/bin/go-plugin-counter-runsc" ]

FROM nonroot-mlock AS nonroot-mlock-runsc

ENTRYPOINT [ "/bin/go-plugin-counter-mlock-runsc" ]

# Set root as the default image.
FROM root
