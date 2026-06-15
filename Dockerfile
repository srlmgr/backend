# This file is used by goreleaser
ARG BUILDPLATFORM
FROM --platform=$BUILDPLATFORM alpine:3.24

ARG TARGETPLATFORM

ENTRYPOINT ["/backend"]

# TODO
# HEALTHCHECK --interval=2s --timeout=2s --start-period=5s --retries=3 CMD [ "/thc" ]
RUN apk add --no-cache ca-certificates

COPY $TARGETPLATFORM/backend /
