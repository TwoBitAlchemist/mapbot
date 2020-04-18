FROM debian:testing

RUN for i in $(seq 1 5); do apt-get update && break; done; \
    for i in $(seq 1 5); do apt-get -y install ca-certificates && break; done

ADD fonts /fonts
#ADD emoji /emoji
ADD mapbot /mapbot
ADD run.sh /run.sh

CMD ["/run.sh"]
