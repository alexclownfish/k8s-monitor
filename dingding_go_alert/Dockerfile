FROM centos
WORKDIR /build
COPY alertGo .
RUN chmod 777 alertGo
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN echo 'Asia/Shanghai' >/etc/timezone
EXPOSE 8088
CMD ["/build/alertGo"]
