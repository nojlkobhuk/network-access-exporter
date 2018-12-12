package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/sirupsen/logrus"
)

const (
	defaultsListenAddr        = ":9407"
	defaultsMetricsPath       = "/metrics"
	defaultsLogLevel          = "info"
	defaultsConnectionTimeout = 500 * time.Millisecond
)

var (
	connectionTimeout   = flag.Duration("timeout", 0, "Connection timeout")
	logLevel            = flag.String("log-level", "", "Logging level")
	listenAddress       = flag.String("web.listen-address", "", "Listen address")
	metricsPath         = flag.String("web.telemetry-path", "", "Metrics path")
	resourcesCollection = flag.String("resources", "", "Resources list")
	configFile          = flag.String("config-file", "", "Configuration file in YAML format")
)

type ItemRaw Item
type ItemMap map[string][]ItemRaw

type Config struct {
	ConnectionTimeout time.Duration `yaml:"connectionTimeout"`
	LogLevel          string        `yaml:"logLevel"`
	ListenAddr        string        `yaml:"listenAddr"`
	MetricsPath       string        `yaml:"metricsPath"`
	RawItems          ItemMap       `yaml:"resources"`
	Items             []Item        `yaml:"-"`
	File              string        `yaml:"-"`
}

func (i *ItemRaw) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var (
		value Item
		str   string
	)
	if err := unmarshal(&value); err != nil {
		if err := unmarshal(&str); err != nil {
			return err
		}
		value = Item{
			Resource: str,
		}
	}
	*i = (ItemRaw)(value)
	return nil
}

func (i *ItemMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var (
		value map[string][]ItemRaw
		slice []ItemRaw
	)
	// try to parse map[string][]string
	if err := unmarshal(&value); err != nil {
		// try to parse []string
		if err := unmarshal(&slice); err != nil {
			return err
		}
		value = map[string][]ItemRaw{}
		value["all"] = slice
	}
	*i = (ItemMap)(value)
	return nil
}

func LoadConfig() (*Config, error) {
	nc := &Config{
		File: *configFile,
	}
	if err := nc.LoadFromFile(); err != nil {
		return nil, err
	}
	if err := nc.LoadFromFlags(); err != nil {
		return nil, err
	}
	nc.SetEmptyToDefaults()
	if err := parseConfig(nc); err != nil {
		return nil, err
	}
	return nc, nil
}

func (c *Config) SetEmptyToDefaults() {
	if len(c.ListenAddr) == 0 {
		c.ListenAddr = defaultsListenAddr
	}
	if len(c.MetricsPath) == 0 {
		c.MetricsPath = defaultsMetricsPath
	}
	if len(c.LogLevel) == 0 {
		c.LogLevel = defaultsLogLevel
	}
	if c.ConnectionTimeout.Nanoseconds() == 0 {
		c.ConnectionTimeout = defaultsConnectionTimeout
	}
}

func (c *Config) LoadFromFlags() error {
	if connectionTimeout.Nanoseconds() != 0 {
		c.ConnectionTimeout = *connectionTimeout
	}
	if len(*logLevel) != 0 {
		c.LogLevel = *logLevel
	}
	if len(*listenAddress) != 0 {
		c.ListenAddr = *listenAddress
	}
	if len(*metricsPath) != 0 {
		c.MetricsPath = *metricsPath
	}
	if len(*resourcesCollection) != 0 {
		resources := strings.Split(*resourcesCollection, ",")
		for _, resourceRaw := range resources {
			resourceRaw = strings.TrimSpace(resourceRaw)
			if len(resourceRaw) == 0 {
				continue
			}
			item := Item{
				Resource: resourceRaw,
			}
			c.Items = append(c.Items, item)
		}
	}
	return nil
}

func (c *Config) LoadFromFile() error {
	if len(*configFile) == 0 {
		return nil
	}
	b, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(b, &c); err != nil {
		return err
	}
	return nil
}

func parseConfig(c *Config) error {
	lvl, err := logrus.ParseLevel(c.LogLevel)
	if err != nil {
		return err
	}
	logrus.SetLevel(lvl)
	if len(c.RawItems) == 0 {
		return errors.New("empty items list")
	}
	for group, items := range c.RawItems {
		for _, item := range items {
			hostPort := strings.Split(item.Resource, ":")
			if len(hostPort) != 2 {
				return fmt.Errorf("incorrect item: %+v", item.Resource)
			}
			portInt, err := strconv.Atoi(hostPort[1])
			if err != nil {
				return fmt.Errorf("incorrent port in item: %+v", item.Resource)
			}
			item := Item{
				Host:  hostPort[0],
				Port:  portInt,
				Group: group,
			}
			c.Items = append(c.Items, item)
		}
	}
	return nil
}
