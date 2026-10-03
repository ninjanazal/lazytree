# vhs (terminal recorder) + git, for recording README screenshots:
#   make screenshots
FROM ghcr.io/charmbracelet/vhs:latest
RUN apt-get update && apt-get install -y --no-install-recommends git \
    && rm -rf /var/lib/apt/lists/* \
    && git config --system --add safe.directory '*'
