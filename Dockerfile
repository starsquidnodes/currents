FROM golang:1.20-alpine AS build
COPY . /app
WORKDIR /app
RUN mkdir -p dist && go build -o dist/

FROM alpine:latest
ENV GIN_MODE release
COPY --from=build /app/dist/currents /usr/local/bin/
ENTRYPOINT ["currents"]
