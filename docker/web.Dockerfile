FROM node:lts-alpine

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

ENV ENABLE_TELEMETRY="false"

FROM caddy:2.0.0-rc.3-alpine
RUN mkdir -p /app/demo
COPY --from=0 /app/demo/dist /app/demo/dist
