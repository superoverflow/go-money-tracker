FROM golang:latest as build

WORKDIR /src/app
COPY ./main.go .
RUN go mod init money-tracker; go mod tidy
RUN go build

FROM chromedp/headless-shell:latest as final
RUN apt-get update; apt install dumb-init -y
ENTRYPOINT ["dumb-init", "--"]

# fix x509 error 
# see https://stackoverflow.com/questions/52969195/docker-container-running-golang-http-client-getting-error-certificate-signed-by
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/


COPY --from=build /src/app/money-tracker /tmp