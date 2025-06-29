FROM alpine:3.22.0

LABEL maintainer="Jon Hadfield jon@lessknown.co.uk"

RUN addgroup -S nonroot \
    && adduser -S nonroot -G nonroot

RUN apk add --update --no-cache ca-certificates bash curl git git-lfs jq grep \
    && rm -f "/var/cache/apk/*"

ARG TAG

WORKDIR /tmp
RUN curl -L -o soba.tar.gz https://github.com/jonhadfield/soba/releases/download/${TAG}/soba_linux_amd64.tar.gz \
    && tar -xvzf soba.tar.gz \
    && rm ./*.gz \
    && mv ./soba /soba \
    && rm -rf /tmp/* \
    && chmod 755 /soba

USER nonroot

ENTRYPOINT ["/bin/bash", "-c", "/soba \"$@\"", "--"]
