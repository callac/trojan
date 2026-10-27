FROM --platform=$TARGETPLATFORM debian:12-slim

ARG TARGETARCH

ARG GITHUB_REPO="callac/trojan"

ARG VERSION_CHECK="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"

ARG DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download"

ARG SERVICE_URL="https://raw.githubusercontent.com/${GITHUB_REPO}/master/asset/trojan-web.service"

ARG SYSTEMCTL_URL="https://raw.githubusercontent.com/gdraheim/docker-systemctl-replacement/master/files/docker/systemctl.py"

RUN apt-get update && \
    apt-get install -y --no-install-recommends curl socat bash-completion cron iptables openssl unzip procps ca-certificates && \
    LATEST_VERSION=$(curl -H 'Cache-Control: no-cache' -s "$VERSION_CHECK" | grep 'tag_name' | cut -d\" -f4) && \
    [[ $TARGETARCH =~ "arm64" ]] && ARCH="arm64" || ARCH="amd64" && \
    curl -fL "$DOWNLOAD_URL/$LATEST_VERSION/trojan-linux-$ARCH" -o /usr/local/bin/trojan && \
    curl -L "$SERVICE_URL" -o /etc/systemd/system/trojan-web.service && \
    curl -L "$SYSTEMCTL_URL" -o /usr/bin/systemctl && \
    chmod +x /usr/local/bin/trojan /usr/bin/systemctl && \
    echo "source <(trojan completion bash)" >> ~/.bashrc && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

