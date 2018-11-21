package common

import "os"

var (
	RedisProtocol                       = "tcp"
	RedisHost                           = "192.168.1.95"
	RedisPort                           = "6379"
	ICE_SERVER_BASE_URL                 = "https://networktraversal.googleapis.com"
	ICE_SERVER_URL_TEMP                 = "%s/v1alpha/iceconfig?key=%s"
	ICE_SERVER_API_KEY                  = os.ExpandEnv("$ICE_SERVER_API_KEY")
	ICE_SERVER_OVERRIDE   []interface{} = nil
	WSS_INSTANCE_HOST_KEY               = "host_port_pair"
	WSS_INSTANCE_NAME_KEY               = "vm_name"
	WSS_INSTANCE_ZONE_KEY               = "zone"
	WSS_INSTANCES                       = []map[string]string{{
		WSS_INSTANCE_HOST_KEY: "192.168.1.5:8089",
		WSS_INSTANCE_NAME_KEY: "wsserver-std",
		WSS_INSTANCE_ZONE_KEY: "us-central1-a",
	}, {
		WSS_INSTANCE_HOST_KEY: "192.168.1.5:8089",
		WSS_INSTANCE_NAME_KEY: "wsserver-std-2",
		WSS_INSTANCE_ZONE_KEY: "us-central1-f",
	}}
)
