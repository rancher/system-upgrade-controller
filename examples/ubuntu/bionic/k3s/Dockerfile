ARG UBUNTU=library/ubuntu@sha256:04d48df82c938587820d7b6006f5071dbbffceb7ca01d2814f81857c631d44df
FROM ${UBUNTU}
ENV LC_ALL C
ENV DEBIAN_FRONTEND noninteractive
RUN set -x \
 && apt-get --assume-yes update \
 && apt-get --assume-yes install \
    bash \
    binutils \
    ca-certificates \
    curl=7.58.0-2ubuntu3 \
    less \
    libcurl4=7.58.0-2ubuntu3 \
    libssl1.1=1.1.0g-2ubuntu4 \
    openssl=1.1.0g-2ubuntu4 \
    systemd \
    systemd-sysv \
 && apt-get --assume-yes clean \
 && rm -rf \
    /etc/systemd/system/*.wants/* \
    /lib/systemd/system/multi-user.target.wants/* \
    /lib/systemd/system/local-fs.target.wants/* \
    /lib/systemd/system/sockets.target.wants/*udev* \
    /lib/systemd/system/sockets.target.wants/*initctl* \
    /lib/systemd/system/basic.target.wants/* \
    /lib/systemd/system/anaconda.target.wants/* \
    /lib/systemd/system/plymouth* \
    /lib/systemd/system/systemd-update-utmp* \
    /sbin/init \
    /tmp/* \
    /var/lib/apt/lists/* \
    /var/tmp/*

ARG INSTALL_K3S_VERSION=v1.17.2+k3s1
ARG INSTALL_K3S_SKIP_START=true
ADD https://raw.githubusercontent.com/rancher/k3s/${INSTALL_K3S_VERSION}/install.sh /usr/share/rancher/k3s/
RUN sh -x /usr/share/rancher/k3s/install.sh || true
RUN k3s --version

ENV PAGER=less

COPY init.sh /sbin/init
RUN chmod +x /sbin/init

VOLUME /var/lib/rancher
ENTRYPOINT ["/sbin/init"]
CMD ["/lib/systemd/systemd"]
