FROM alpine:3.8
MAINTAINER Jon Hadfield <jon@lessknown.co.uk>

ARG build_tag

RUN apk add --update --no-cache ca-certificates bash curl git \
    && rm -f "/var/cache/apk/*" \
    && curl -L https://dl.bintray.com/jonhadfield/soba/soba/$build_tag/soba_linux_amd64 -o /soba \
    && chmod 755 /soba
ENTRYPOINT ["/bin/bash", "-c", "/soba \"$@\"", "--"]
