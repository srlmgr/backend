# This file is used by goreleaser
FROM scratch
ARG TARGETPLATFORM
ENTRYPOINT ["/backend"]
COPY $TARGETPLATFORM/backend /
