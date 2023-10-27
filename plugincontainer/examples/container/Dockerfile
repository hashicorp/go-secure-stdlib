# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

FROM gcr.io/distroless/static-debian12 as root

COPY go-plugin-counter /bin/go-plugin-counter

ENTRYPOINT [ "/bin/go-plugin-counter" ]

FROM docker.mirror.hashicorp.services/alpine as nonroot

COPY go-plugin-counter /bin/go-plugin-counter

RUN apk add libcap && \
    addgroup -S nonroot && \
    adduser -S -G nonroot nonroot && \
    chown -R nonroot:nonroot /bin/go-plugin-counter && \
    cp /bin/go-plugin-counter /bin/go-plugin-counter-mlock && \
    setcap cap_ipc_lock=+ep /bin/go-plugin-counter-mlock

USER nonroot

ENTRYPOINT [ "/bin/go-plugin-counter" ]

FROM nonroot as nonroot-mlock

ENTRYPOINT [ "/bin/go-plugin-counter-mlock" ]

# Set root as the default image.
FROM root