FROM golang:1.17 as build
WORKDIR /app
COPY . .
RUN go build -v .

FROM gcr.io/distroless/base-debian11
COPY --from=build /app/pswa /
EXPOSE 8080
CMD ["/pswa"]
