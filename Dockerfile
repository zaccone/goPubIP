FROM scratch

MAINTAINER Marek Denis <marek.denis+zaccone@gmail.com>

EXPOSE 5300

COPY ca-certificates.crt /etc/ssl/certs
ADD goPubIP /
CMD ["/goPubIP"]
