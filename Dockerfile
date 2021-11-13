FROM golang:1.17 as build
WORKDIR /app
COPY . .
RUN go build -v .

FROM klakegg/hugo:ext-alpine as hugo
WORKDIR /hugo
COPY hugo /hugo
RUN hugo

FROM gcr.io/distroless/base-debian11
COPY --from=build /app/pswa /
COPY --from=hugo /hugo/public /public
EXPOSE 8080
ENTRYPOINT ["/pswa"]
