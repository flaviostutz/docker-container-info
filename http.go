package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	cors "github.com/itsjamie/gin-cors"
)

type HTTPServer struct {
	server                *http.Server
	router                *gin.Engine
	dockerClient          *client.Client
	containersInfo        map[string]map[string]string
	lastContainerInfoTime *time.Time
	cacheTimeout          int
}

func NewHTTPServer(cacheTimeout int) (*HTTPServer, error) {
	router := gin.Default()

	router.Use(cors.Middleware(cors.Config{
		Origins:         "*",
		Methods:         "GET",
		RequestHeaders:  "Origin, Content-Type",
		ExposedHeaders:  "",
		MaxAge:          1 * time.Hour,
		Credentials:     false,
		ValidateHeaders: false,
	}))

	logrus.Debugf("Preparing Docker cli")

	dockerClient, err := client.NewClientWithOpts(client.WithVersion("1.38"))
	if err != nil {
		return nil, fmt.Errorf("Error creating Docker client instance. err=%s", err)
	}

	h := &HTTPServer{
		server: &http.Server{
			Addr:    ":5000",
			Handler: router,
		},
		router:       router,
		dockerClient: dockerClient,
		cacheTimeout: cacheTimeout,
	}

	logrus.Debugf("Registering API routes...")
	router.GET("/_self", h.infoSelf())
	router.GET("/info/:containerId", h.infoContainerID())

	return h, nil
}

func (h *HTTPServer) infoContainerID() func(*gin.Context) {
	return func(c *gin.Context) {

		containerID := c.Param("containerId")
		logrus.Debugf("Getting info for containerId=%s", containerID)

		cinfo, err := h.getContainerInfo(containerID)
		if err != nil {
			logrus.Debugf("Couldn't find container info for id=%s. err=%s", containerID, err)
			c.Header("Cache-Control", "no-cache")
			c.JSON(http.StatusNotFound, gin.H{"message": fmt.Sprintf("Couldn't get info for container %s", containerID)})
			return
		}

		c.Header("Cache-Control", "no-cache")
		c.JSON(http.StatusOK, cinfo)
	}
}

func (h *HTTPServer) infoSelf() func(*gin.Context) {
	return func(c *gin.Context) {

		ip, err := getClientIPByRequestRemoteAddr(c.Request)
		if err != nil {
			logrus.Debugf("Error getting remote IP. err=%s", err)
			c.Header("Cache-Control", "no-cache")
			c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("Couldn't determine caller IP from request")})
			return
		}
		logrus.Debugf("Request source IP=%s", ip)

		cinfo, err := h.getContainerInfoFromIP(ip)
		if err != nil {
			logrus.Debugf("Couldn't find container info for container with IP. err=%s", err)
			c.Header("Cache-Control", "no-cache")
			c.JSON(http.StatusNotFound, gin.H{"message": fmt.Sprintf("Couldn't find info for container with IP %s", ip)})
			return
		}

		c.Header("Cache-Control", "no-cache")
		c.JSON(http.StatusOK, cinfo)
	}
}

func (h *HTTPServer) getContainerInfoFromIP(sourceIP string) (map[string]string, error) {
	cinfos, err := h.getContainersInfo()
	if err != nil {
		return nil, fmt.Errorf("Containers info could not be loaded. err=%s", err)
	}

	//find container id for IP
	for _, cinfo := range cinfos {
		cn := 0
		for {
			ip, ok := cinfo[fmt.Sprintf("ip:%d", cn)]
			if !ok {
				break
			}
			if ip == sourceIP {
				return cinfo, nil
			}
			cn = cn + 1
		}
	}

	return nil, fmt.Errorf("Couldn't find container info for %s", sourceIP)
}

func (h *HTTPServer) getContainerInfo(containerID string) (map[string]string, error) {
	cinfos, err := h.getContainersInfo()
	if err != nil {
		return nil, fmt.Errorf("Containers info could not be loaded. err=%s", err)
	}
	tid := truncateString(containerID, 10)
	cinfo, ok := cinfos[tid]
	if !ok {
		return nil, fmt.Errorf("Container info not found for %s", containerID)
	}

	return cinfo, nil
}

func (h *HTTPServer) getContainersInfo() (map[string]map[string]string, error) {
	if h.cacheValid() {
		return h.containersInfo, nil
	}

	logrus.Debugf("Refreshing containers info cache")
	containers, err := h.dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		logrus.Errorf("Error listing containers. err=%s", err)
		return nil, err
	}

	cinfo := make(map[string]map[string]string)
	for _, c := range containers {

		info := make(map[string]string)

		nodeID, ok := c.Labels["com.docker.swarm.node.id"]
		if ok {
			logrus.Debugf("Getting Swarm node label for public Ip")
			snode, _, err := h.dockerClient.NodeInspectWithRaw(context.Background(), nodeID)
			if err != nil {
				logrus.Warnf("Couldn't get node inspect data. err=%s", err)
			}
			npublicIP, ok := snode.Spec.Labels["publicIp"]
			if ok {
				logrus.Debugf("Node public ip=%s", npublicIP)
				info["nodePublicIp"] = npublicIP
				info["publicIp"] = npublicIP
			}
		}

		pip, ok := c.Labels["publicIp"]
		if ok {
			info["labelPublicIp"] = pip
			info["publicIp"] = pip
		}

		info["id"] = c.ID
		info["created"] = time.Unix(0, c.Created).Format(time.RFC3339)
		info["image"] = c.Image
		info["status"] = c.Status
		info["state"] = c.State
		info["networkMode"] = c.HostConfig.NetworkMode

		for k, v := range c.Labels {
			info[fmt.Sprintf("label:%s", k)] = v
		}

		nc := 0
		for _, v := range c.NetworkSettings.Networks {
			info[fmt.Sprintf("ip:%d", nc)] = v.IPAddress
			nc = nc + 1
		}

		for nc, v := range c.Ports {
			info[fmt.Sprintf("hostBindPort:%d", nc)] = v.IP
			info[fmt.Sprintf("publicPort:%d", nc)] = fmt.Sprintf("%d", v.PublicPort)
			info[fmt.Sprintf("privatePort:%d", nc)] = fmt.Sprintf("%d", v.PrivatePort)
		}

		tid := truncateString(c.ID, 10)
		cinfo[tid] = info
	}

	t := time.Now()
	h.lastContainerInfoTime = &t
	h.containersInfo = cinfo

	return h.containersInfo, nil
}

func (h *HTTPServer) cacheValid() bool {
	if h.cacheTimeout == -1 {
		return false
	}
	if h.lastContainerInfoTime != nil {
		elapsed := time.Since(*h.lastContainerInfoTime)
		if elapsed.Milliseconds() <= int64(h.cacheTimeout) {
			return true
		}
	}
	return false
}

//Start the main HTTP Server entry
func (s *HTTPServer) Start() error {
	logrus.Infof("Starting HTTP Server on port 3000")
	return s.server.ListenAndServe()
}

func truncateString(str string, num int) string {
	bnoden := str
	if len(str) > num {
		if num > 3 {
			num -= 3
		}
		bnoden = str[0:num] + "..."
	}
	return bnoden
}
