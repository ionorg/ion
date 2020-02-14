# FROM ubuntu:xenial-20200114 as certBuilder
# COPY . /app
# RUN /app/scripts/makeKey.sh

FROM node:12.16

COPY . /app
# COPY --from=certBuilder /app/configs/key.pem /app/configs/key.pem
# COPY --from=certBuilder /app/configs/key.pem /app/configs/cert.pem

WORKDIR /app/sdk/js/demo

RUN npm install
RUN npm run build
RUN npm install -g http-server
ENTRYPOINT ["http-server", "./dist", "--port", "9000", "--ssl", "--cert", "./../../../configs/cert.pem", "--key", "./../../../configs/key.pem"]