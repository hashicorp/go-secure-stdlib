FROM ubuntu

RUN groupadd go-plugin

COPY go-plugin-counter /bin/go-plugin-counter

ENTRYPOINT [ "/bin/go-plugin-counter" ]