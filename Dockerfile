FROM golang:latest as build

WORKDIR /src/app
COPY ./main.go .
RUN go mod init money-tracker; go mod tidy
RUN go build

FROM chromedp/headless-shell:latest as final
RUN apt-get update; apt install dumb-init -y
ENTRYPOINT ["dumb-init", "--"]
COPY --from=build /src/app/money-tracker /tmp