#FROM alpine:3.9.5 as certBuilder
#RUN apk upgrade --update-cache --available && apk add openssl
#
#RUN openssl req -newkey rsa:2048 -new -nodes -x509 -days 3650 \
#-subj "/C=US/ST=No/L=No/O=Pion/CN=No" \
#-keyout /key.pem -out /cert.pem


FROM node:12.16
COPY ./sdk/js /app

WORKDIR /app
RUN npm install

WORKDIR /app/demo
RUN npm install
RUN npm run build
RUN npm install -g http-server

#COPY --from=certBuilder /key.pem /cert/key.pem
#COPY --from=certBuilder /cert.pem /cert/cert.pem

ENTRYPOINT ["http-server", "./dist"]