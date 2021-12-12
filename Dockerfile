# build
FROM node:15.5.0-alpine3.10 as build-vue
WORKDIR /app
ENV PATH /app/node_modules/.bin:$PATH
COPY ./client/package*.json ./client/.npmrc ./
RUN npm config set update-notifier false && \
    npm install
COPY ./client .
RUN npm run build

# python packages
FROM python:3-alpine as builder
WORKDIR /app
RUN apk update && apk add --no-cache python3 && \
    python3 -m ensurepip && \
    rm -r /usr/lib/python*/ensurepip && \
    pip3 install --upgrade pip setuptools && \
    if [ ! -e /usr/bin/pip ]; then ln -s pip3 /usr/bin/pip ; fi && \
    if [[ ! -e /usr/bin/python ]]; then ln -sf /usr/bin/python3 /usr/bin/python; fi && \
    rm -r /root/.cache
COPY ./server/requirements.txt ./
RUN apk update && apk add --no-cache gcc musl-dev libressl-dev libffi-dev python3-dev && \
    pip install --user -r requirements.txt

# production
FROM nginx:stable-alpine as production
WORKDIR /app
COPY --from=build-vue /app/dist /usr/share/nginx/html
COPY ./nginx/default.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /root/.local/ /usr/local/
RUN apk update && apk add --no-cache py3-gunicorn
COPY ./server .
CMD gunicorn -b 0.0.0.0:5000 'dailytxt.application:create_app()' --daemon && \
    sed -i -e 's/$PORT/'"$PORT"'/g' /etc/nginx/conf.d/default.conf && \
    nginx -g 'daemon off;'