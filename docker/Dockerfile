# Track the 3.22 branch rather than the frozen 3.22.0 point release: a pinned
# patch tag never receives security updates, which left 19 fixable HIGH/CRITICAL
# CVEs (openssl, musl, zlib) baked into every published image.
FROM alpine:3.22

LABEL maintainer="Jon Hadfield jon@lessknown.co.uk"

RUN addgroup -S nonroot \
    && adduser -S nonroot -G nonroot \
    && apk upgrade --no-cache \
    && apk add --update --no-cache bash ca-certificates curl git git-lfs grep jq \
    && rm -f "/var/cache/apk/*"

ARG TAG

WORKDIR /tmp
RUN curl -L --proto "=https" --tlsv1.2 -o soba.tar.gz "https://github.com/jonhadfield/soba/releases/download/${TAG}/soba_linux_amd64.tar.gz" \
    && tar -xvzf soba.tar.gz \
    && rm ./*.gz \
    && mv ./soba /soba \
    && rm -rf /tmp/* \
    && chmod 755 /soba

USER nonroot

ENTRYPOINT ["/bin/bash", "-c", "/soba \"$@\"", "--"]
