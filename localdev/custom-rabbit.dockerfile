FROM rabbitmq:3-management

MAINTAINER <ssmith2347@gmail.com>

ADD rabbitmq.conf /etc/rabbitmq/
ADD definitions.json /etc/rabbitmq/