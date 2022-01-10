FROM golang:1.17 as build
WORKDIR /app
COPY . .
RUN go build -v .

FROM node:lts as testroot
WORKDIR /testroot
COPY testroot /testroot
RUN npm install
RUN npm run build

FROM debian:11-slim
ENV SSH_PASSWD root:Docker!
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates openssh-server && \
    echo $SSH_PASSWD | chpasswd && \
    mkdir -p /var/run/sshd
COPY sshd_config /etc/ssh
COPY --from=build /app/pswa /
COPY --from=testroot /testroot/dist /testroot
EXPOSE 8080 2222
CMD ["sh", "-c", "/usr/sbin/sshd && /pswa"]
