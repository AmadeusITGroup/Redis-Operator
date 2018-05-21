FROM redis:4.0-alpine

# Add the binary. As it is statically linked, no need to add libc or anything else.
COPY redis-cluster.conf /redis-server/redis.conf
RUN chmod 777 /redis-server/redis.conf
ADD ./redisnode /

# Volume for redis server data
VOLUME /redis-data

ENTRYPOINT [ "/redisnode" ]
