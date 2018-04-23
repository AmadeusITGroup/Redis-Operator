package initcontainer

import (
	"os"
	"strconv"

	"github.com/golang/glog"
)

// InitContainer contains all InitContainer needed information
type InitContainer struct {
	config *Config
}

// NewInitContainer builds and returns new InitContainer instance
func NewInitContainer(cfg *Config) (*InitContainer, error) {
	return &InitContainer{
		config: cfg,
	}, nil
}

// Init used to initialized the IniContainer process
func (ic *InitContainer) Init() error {
	return nil
}

// Clear used to clear all resources instanciated by the InitContainer
func (ic *InitContainer) Clear() {

}

// Start used to start the InitContainer process
func (ic *InitContainer) Start(stop <-chan struct{}) error {
	if err := ic.updateNodeConfigFile(); err != nil {
		glog.Errorf("unable to update the configFile, err:%v", err)
		return err
	}
	return nil
}

// updateNodeConfigFile update the redis config file with node information: ip, port
func (ic *InitContainer) updateNodeConfigFile() error {
	err := ic.addSettingInConfigFile("port " + ic.config.Port)
	if err != nil {
		return err
	}
	err = ic.addSettingInConfigFile("bind " + ic.config.Host + " 127.0.0.1")
	if err != nil {
		return err
	}
	err = ic.addSettingInConfigFile("cluster-node-timeout " + strconv.Itoa(ic.config.ClusterNodeTimeout))
	if err != nil {
		return err
	}
	if ic.config.getRenameCommandsFile() != "" {
		err = ic.addSettingInConfigFile("include " + ic.config.getRenameCommandsFile())
		if err != nil {
			return err
		}
	}

	return nil
}

// addSettingInConfigFile add a line in the redis configuration file
func (ic *InitContainer) addSettingInConfigFile(line string) error {
	f, err := os.OpenFile(ic.config.ConfigFile, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.WriteString(line + "\n")
	return err
}
