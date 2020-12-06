FROM clearlinux/node:12.18.4

WORKDIR /service
COPY package.json .
RUN npm install
COPY *.js ./

CMD ["node", "/service/server.js"]
