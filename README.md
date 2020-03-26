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

* Run ```docker ps``` and annotate the container Id for "test1"

* Run ```docker-compose exec test1 apk add curl```

* Run ```docker-compose exec test1 curl http://docker-info:5000/info/:containerId```

* Check for host/local port mappings over published ports etc on console output

## APIs

* **/_self** - will try to determine which container is calling this API by its remote IP and then returns info about the calling container

* **/info/:containerId** - will get info about the container by its id. If in a Swarm Cluster, it will try to determine which node the container is running on and then will return the contents of the label "publicIp" applied to the node. You can use this label to indicate which public ip can reach the node.

## Swarm Clusters

* If you want /info to return a custom public ip to the container indicating from which public ip this container could be reached ("nodePublicIp", "labelPublicIp" and "publicIp" attributes)
  * Run the container as part of a Stack
  * Label each Swarm node with "publicIp" indicating the public IP (the IP you are doing some external 1x1 NAT to it, or placed the public ip directly on the node)
    * ```docker node update --label-add publicIp=222.22.22.22 aiq283jgrejlsiwlvbfiegyv0```
  * If outside Swarm cluster, place a "Label" in container with name "publicIp"
  * Now /info/:containerId will search for the Node that is running the container and return this label contents
