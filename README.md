# docker-info

REST API that exposes some information about running Docker containers.

Created to help containers know more info about itself, like random published port on host when it needs this info to notify clients in sharded configurations.

## Usage

* Create docker-compose.yml

```yml
version: '3.5'
services:
  docker-info:
    image: flaviostutz/docker-info
    environment:
      - LOG_LEVEL=debug
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 5000:5000
  test1:
    image: alpine
    command: sleep 99999
    restart: always
    ports:
      - 7000:5000
```

* Run ```docker-compose up -d```

* Run ```docker-compose exec test1 apk add curl```

* Run ```docker-compose exec test1 curl http://docker-info:5000/info/_self```

* Check for host/local port mappings over published ports etc on console output
