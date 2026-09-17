FROM scratch

LABEL org.opencontainers.image.authors="Marek Denis <marek.denis+zaccone@gmail.com>"

EXPOSE 5300/udp

COPY ca-certificates.crt /etc/ssl/certs
COPY goPubIP /

ENTRYPOINT ["/goPubIP"]
