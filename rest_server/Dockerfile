FROM node:10

WORKDIR /service
COPY package.json .
RUN npm install
COPY server.js .

CMD ["node", "/service/server.js"]
