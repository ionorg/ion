FROM node:12.16.0-buster-slim

WORKDIR /app
COPY ./sdk/js/package.json ./
RUN npm install

COPY ./sdk/js/demo/package.json ./demo/
WORKDIR /app/demo
RUN npm install

WORKDIR /app
COPY ./sdk/js ./

WORKDIR /app/demo
RUN npm run build

RUN apt-get update && apt-get install -y \
    curl \
 && rm -rf /var/lib/apt/lists/*

ENV ENABLE_TELEMETRY="false"
RUN curl https://getcaddy.com | bash -s personal

ENTRYPOINT ["/usr/local/bin/caddy"]
CMD ["--conf", "/etc/Caddyfile", "--log", "stdout", "--agree=true"]
