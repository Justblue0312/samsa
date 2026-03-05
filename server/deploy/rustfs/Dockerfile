FROM alpine:latest

# Install dependencies
# minio-client (mc) is often available as 'mc' or via curl.
# Alpine edge/community might have it, or we download binary.
# Downloading binary is safer for latest version.

RUN apk add --no-cache curl bash

# Download mc binary
RUN curl https://dl.min.io/client/mc/release/linux-amd64/mc \
    --create-dirs \
    -o /usr/local/bin/mc && \
    chmod +x /usr/local/bin/mc

WORKDIR /s3-init

COPY setup.sh /usr/local/bin/setup.sh
RUN chmod +x /usr/local/bin/setup.sh

CMD ["/usr/local/bin/setup.sh"]
