FROM fedora:34

RUN dnf -y install busybox \
    && dnf clean all \
    && rm -rf /var/cache/yum

COPY tra_client /usr/local/bin
COPY tra_server /usr/local/bin

EXPOSE 50052
EXPOSE 50053

ENTRYPOINT ["/usr/local/bin/tra_server"]
