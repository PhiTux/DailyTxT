# build
FROM node:18-alpine3.19 as build-vue
WORKDIR /app
ENV PATH /app/node_modules/.bin:$PATH
COPY ./client/package*.json ./client/.npmrc ./
RUN npm config set update-notifier false && \
    npm update -g npm && npm ci --no-audit --maxsockets 1
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
    pip install --user -r requirements.txt && \
    pip install --user gunicorn

# production
FROM python:3-alpine as production
WORKDIR /app
COPY --from=build-vue /app/dist /usr/share/nginx/html
RUN apk update && apk add --no-cache nginx
COPY ./nginx/default.conf /etc/nginx/conf.d/default.conf
COPY ./nginx/nginx.conf /etc/nginx/nginx.conf
COPY --from=builder /root/.local/ /root/.local/
ENV PATH=/root/.local/bin:$PATH
COPY ./server .
CMD gunicorn -b 0.0.0.0:5000 'dailytxt.application:create_app()' --daemon && \
    sed -i -e 's/$PORT/'"$PORT"'/g' /etc/nginx/conf.d/default.conf && \
    nginx -g 'daemon off;'