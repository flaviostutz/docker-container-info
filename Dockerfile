FROM golang:1.14.1-alpine3.11 AS BUILD

RUN mkdir /docker-info
WORKDIR /docker-info

ADD go.mod .
RUN go mod download

#now build source code
ADD . ./
RUN go build -o /go/bin/docker-info



FROM golang:1.14.1-alpine3.11

ENV LOG_LEVEL 'info'
ENV CACHE_TIMEOUT ''
ENV DOCKER_HOST ''

COPY --from=BUILD /go/bin/* /bin/
ADD startup.sh /

VOLUME [ "/var/run/docker.sock" ]
EXPOSE 5000

CMD [ "/startup.sh" ]
