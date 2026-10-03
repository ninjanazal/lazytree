# Test image: lazytree's lint + tests on a clean Linux userland.
#   make docker-test                      # Debian (default base)
#   make docker-matrix                    # Debian bookworm/trixie + Alpine
# BASE picks the distro/Go image; git comes from that distro's package
# manager, so each base also exercises a different git version.
ARG BASE=golang:1.26
FROM ${BASE}

RUN if command -v apt-get >/dev/null; then \
      apt-get update && apt-get install -y --no-install-recommends git make bash python3 ca-certificates \
      && rm -rf /var/lib/apt/lists/*; \
    else \
      apk add --no-cache git make bash python3 ca-certificates; \
    fi

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Tests create git repos; give git an identity-free, hermetic home.
ENV HOME=/tmp/home
RUN mkdir -p $HOME && git config --global --add safe.directory /src

CMD ["bash", "docker/test.sh"]
