FROM alpine:3.6

RUN adduser -D node-egress-operator
USER node-egress-operator

ADD tmp/_output/bin/node-egress-operator /usr/local/bin/node-egress-operator
