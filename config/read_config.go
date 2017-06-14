package config

import (
	"strings"

	"strconv"

	"fmt"

	"encoding/json"

	"sync"

	"sort"

	"github.com/xiaonanln/goworld/gwlog"
	"gopkg.in/ini.v1"
)

const (
	DEFAULT_CONFIG_FILE  = "goworld.ini"
	DEFAULT_LOCALHOST_IP = "127.0.0.1"
)

var (
	configFilePath = DEFAULT_CONFIG_FILE
	goWorldConfig  *GoWorldConfig
	configLock     sync.Mutex
)

type ServerConfig struct {
	Ip         string
	Port       int
	BootEntity string
}

type DispatcherConfig struct {
	Ip   string
	Port int
}

type GoWorldConfig struct {
	Dispatcher DispatcherConfig
	Servers    map[int]*ServerConfig
	Storage    StorageConfig
}

type StorageConfig struct {
	Type string
	// Filesystem Storage Configs
	Directory string // directory for filesystem storage
	// MongoDB storage configs
}

func SetConfigFile(f string) {
	configFilePath = f
}

func Get() *GoWorldConfig {
	configLock.Lock()
	defer configLock.Unlock() // protect concurrent access from Games & Gate
	if goWorldConfig == nil {
		goWorldConfig = readGoWorldConfig()
	}
	return goWorldConfig
}

func Reload() *GoWorldConfig {
	configLock.Lock()
	defer configLock.Unlock()

	goWorldConfig = nil
	return Get()
}

func GetServer(serverid int) *ServerConfig {
	return Get().Servers[serverid]
}

func GetServerIDs() []int {
	cfg := Get()
	serverIDs := make([]int, 0, len(cfg.Servers))
	for id, _ := range cfg.Servers {
		serverIDs = append(serverIDs, id)
	}
	sort.Ints(serverIDs)
	return serverIDs
}

func GetDispatcher() *DispatcherConfig {
	return &Get().Dispatcher
}

func GetStorage() *StorageConfig {
	return &Get().Storage
}

func DumpPretty(cfg interface{}) string {
	s, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return err.Error()
	}
	return string(s)
}

func readGoWorldConfig() *GoWorldConfig {
	config := GoWorldConfig{
		Servers: map[int]*ServerConfig{},
	}
	gwlog.Info("Using config file: %s", configFilePath)
	iniFile, err := ini.Load(configFilePath)
	checkConfigError(err, "")
	for _, sec := range iniFile.Sections() {
		secName := sec.Name()
		if secName == "DEFAULT" {
			continue
		}

		//gwlog.Info("Section %s", sec.Name())
		secName = strings.ToLower(secName)
		if secName == "dispatcher" {
			// dispatcher config
			readDispatcherConfig(sec, &config.Dispatcher)
		} else if secName[:6] == "server" {
			// server config
			id, err := strconv.Atoi(secName[6:])
			checkConfigError(err, fmt.Sprintf("invalid server name: %s", secName))
			config.Servers[id] = readServerConfig(sec)
		} else if secName == "storage" {
			// storage config
			readStorageConfig(sec, &config.Storage)
		} else {
			gwlog.Warn("unknown section: %s", secName)
		}

	}
	return &config
}

func readServerConfig(sec *ini.Section) *ServerConfig {
	sc := &ServerConfig{
		Ip: DEFAULT_LOCALHOST_IP,
	}
	for _, key := range sec.Keys() {
		name := strings.ToLower(key.Name())
		if name == "ip" {
			sc.Ip = key.MustString(DEFAULT_LOCALHOST_IP)
		} else if name == "port" {
			sc.Port = key.MustInt(0)
		} else if name == "boot_entity" {
			sc.BootEntity = key.MustString("")
		}
	}
	// validate game config
	if sc.BootEntity == "" {
		panic("boot_entity is not set in server config")
	}
	return sc
}

func readDispatcherConfig(sec *ini.Section, config *DispatcherConfig) {
	config.Ip = DEFAULT_LOCALHOST_IP
	for _, key := range sec.Keys() {
		name := strings.ToLower(key.Name())
		if name == "ip" {
			config.Ip = key.MustString(DEFAULT_LOCALHOST_IP)
		} else if name == "port" {
			config.Port = key.MustInt(0)
		}
	}
	return
}

func readStorageConfig(sec *ini.Section, config *StorageConfig) {
	// setup default values
	config.Type = "filesystem"
	config.Directory = "_entity_storage"

	for _, key := range sec.Keys() {
		name := strings.ToLower(key.Name())
		if name == "type" {
			config.Type = key.MustString("filesystem")
		} else if name == "directory" {
			config.Directory = key.MustString("_entity_storage")
		}
	}

	validateStorageConfig(config)
}

func checkConfigError(err error, msg string) {
	if err != nil {
		if msg == "" {
			msg = err.Error()
		}
		gwlog.Panicf("read config error: %s", msg)
	}
}

func validateStorageConfig(config *StorageConfig) {
	if config.Type == "filesystem" {
		// directory must be set
	} else {
		gwlog.Panicf("unknown storage type: %s", config.Type)
	}
}