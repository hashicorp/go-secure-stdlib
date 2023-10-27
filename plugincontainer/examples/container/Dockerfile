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
    setcap cap_ipc_lock=+p /bin/go-plugin-counter

USER nonroot

ENTRYPOINT [ "/bin/go-plugin-counter" ]

# Set root as the default image.
FROM root