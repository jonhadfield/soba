FROM alpine:3.14

LABEL maintainer="Jon Hadfield jon@lessknown.co.uk"

RUN apk add --update --no-cache ca-certificates bash curl git jq grep \
    && rm -f "/var/cache/apk/*"

ARG TAG

RUN cd /tmp &&  \
    curl -L https://github.com/jonhadfield/soba/releases/download/$TAG/soba_${TAG}_linux_amd64.tar.gz -o soba.tar.gz \
    && tar -xvzf soba.tar.gz \
    && rm *.gz \
    && mv *amd64/soba /soba \
    && rm -rf /tmp/* \
    && chmod 755 /soba

ENTRYPOINT ["/bin/bash", "-c", "/soba \"$@\"", "--"]
