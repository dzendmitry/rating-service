FROM mongo:3.4.2

COPY mongo-entrypoint.sh /usr/local/bin/

RUN mkdir -p /data
COPY mongo-base /data

RUN chmod +x /usr/local/bin/mongo-entrypoint.sh

EXPOSE 27017

ENTRYPOINT ["mongo-entrypoint.sh"]
