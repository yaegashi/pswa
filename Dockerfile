FROM golang:1.17 as build
WORKDIR /app
COPY . .
RUN go build -v .

FROM node:lts as testroot
WORKDIR /testroot
COPY testroot /testroot
RUN npm install
RUN npm run build

FROM gcr.io/distroless/base-debian11
COPY --from=build /app/pswa /
COPY --from=testroot /testroot/dist /testroot
EXPOSE 8080
ENTRYPOINT ["/pswa"]
