FROM node:12.16.0-buster-slim as builder

WORKDIR /app
COPY ./sdk/js/package.json ./sdk/js/package-lock.json ./
RUN npm install

COPY ./sdk/js/demo/package.json ./demo/
WORKDIR /app/demo
RUN npm install

WORKDIR /app
COPY ./sdk/js ./

WORKDIR /app/demo
RUN npm run build

FROM nginx:1.17
WORKDIR /dist
COPY --from=builder /app/demo/dist .