# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

FROM ubuntu

COPY go-plugin-counter /bin/go-plugin-counter

RUN useradd -ms /bin/bash newuser

USER newuser

ENTRYPOINT [ "/bin/go-plugin-counter" ]