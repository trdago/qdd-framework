FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

# Install basic requirements: Go and Node.js
RUN apt-get update && apt-get install -y \
    curl \
    git \
    build-essential \
    software-properties-common

# Install Go 1.22
RUN curl -OL https://golang.org/dl/go1.22.4.linux-amd64.tar.gz && \
    tar -C /usr/local -xvf go1.22.4.linux-amd64.tar.gz && \
    rm go1.22.4.linux-amd64.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"

# Install Node.js 20
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
    apt-get install -y nodejs

# Set up test project dir
WORKDIR /test_env

# Copy QDD binary
COPY qdd-bin /usr/local/bin/qdd
RUN chmod +x /usr/local/bin/qdd

# Set entrypoint
CMD ["bash"]
