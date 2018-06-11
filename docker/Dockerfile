FROM alpine:latest
MAINTAINER Jon Hadfield <jon@lessknown.co.uk>
RUN apk add --update --no-cache ca-certificates bash curl git && rm -f /var/cache/apk/* \
    && curl -L https://dl.bintray.com/jonhadfield/soba/soba_linux_amd64 -o /soba \
    && chmod 755 /soba
ENTRYPOINT ["/bin/bash", "-c", "/soba \"$@\"", "--"]
