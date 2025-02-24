package factory

import (
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/asaskevich/govalidator"
	"github.com/comp590/ocss/internal/logger"
	"gopkg.in/yaml.v3"
)

const (
	OCSSDefaultConfigPath = "./config/ocsscfg.yaml"
)

// Config represents the root configuration structure.
type Config struct {
	OCSSName      string         `yaml:"ocssName" valid:"required"`
	Configuration *Configuration `yaml:"configuration" valid:"required"`
	Logger        *Logger        `yaml:"logger" valid:"required"`

	sync.RWMutex
}

// Validate validates the entire Config struct.
func (c *Config) Validate() (bool, error) {
	if configuration := c.Configuration; configuration != nil {
		if result, err := configuration.Validate(); err != nil {
			return result, err
		}
	}

	result, err := govalidator.ValidateStruct(c)
	return result, appendInvalid(err)
}

// Configuration encapsulates all configuration sections.
type Configuration struct {
	NetworkManager *NetworkManager `yaml:"networkManager" valid:"required"`
	User           *User           `yaml:"user" valid:"required"`
}

// Validate validates the Configuration struct.
func (conf *Configuration) Validate() (bool, error) {
	if conf.NetworkManager != nil {
		if result, err := conf.NetworkManager.Validate(); err != nil {
			return result, err
		}
	}

	if conf.User != nil {
		if result, err := conf.User.Validate(); err != nil {
			return result, err
		}
	}

	result, err := govalidator.ValidateStruct(conf)
	return result, appendInvalid(err)
}

// NetworkManager encapsulates network-related configurations.
type NetworkManager struct {
	PacketSwitchs  []PacketSwitch  `yaml:"packetSwitchs" valid:"required"`
	OpticalSwitchs []OpticalSwitch `yaml:"opticalSwitchs" valid:"required"`
	Servers        []Server        `yaml:"servers" valid:"required"`
	Links          []Link          `yaml:"links" valid:"required"`
	AppServer      Server          `yaml:"appServer" valid:"required"`
}

// Validate validates the NetworkManager struct.
func (nm *NetworkManager) Validate() (bool, error) {
	for _, ps := range nm.PacketSwitchs {
		if result, err := ps.Validate(); err != nil {
			return result, err
		}
	}

	for _, os := range nm.OpticalSwitchs {
		if result, err := os.Validate(); err != nil {
			return result, err
		}
	}

	for _, srv := range nm.Servers {
		if result, err := srv.Validate(); err != nil {
			return result, err
		}
	}

	for _, link := range nm.Links {
		if result, err := link.Validate(); err != nil {
			return result, err
		}
	}

	result, err := govalidator.ValidateStruct(nm)
	return result, appendInvalid(err)
}

// User encapsulates user-related configurations.
type User struct {
	Link []Link    `yaml:"link" valid:"required"`
	ToR  []UserTor `yaml:"tor" valid:"required"`
	OCS  []UserOCS `yaml:"ocs" valid:"required"`
}

// Validate validates the User struct.
func (u *User) Validate() (bool, error) {
	for _, link := range u.Link {
		if result, err := link.Validate(); err != nil {
			return result, err
		}
	}

	for _, ocs := range u.OCS {
		if result, err := ocs.Validate(); err != nil {
			return result, err
		}
	}

	result, err := govalidator.ValidateStruct(u)
	return result, appendInvalid(err)
}

// PacketSwitch represents a network packet switch.
type PacketSwitch struct {
	Device string `yaml:"device" valid:"required"`
	ID     int    `yaml:"id" valid:"required"`
	Ports  string `yaml:"ports" valid:"required"`
}

// Validate validates the PacketSwitch struct.
func (ps *PacketSwitch) Validate() (bool, error) {
	result, err := govalidator.ValidateStruct(ps)
	return result, appendInvalid(err)
}

// OpticalSwitch represents an optical switch.
type OpticalSwitch struct {
	Device string `yaml:"device" valid:"required"`
	Ports  string `yaml:"ports" valid:"required"`
	Ip     string `yaml:"ip" valid:"required,ip"`
}

// Validate validates the OpticalSwitch struct.
func (os *OpticalSwitch) Validate() (bool, error) {
	result, err := govalidator.ValidateStruct(os)
	return result, appendInvalid(err)
}

// Server represents a server configuration.
type Server struct {
	Name string `yaml:"name" valid:"required"`
	IP   string `yaml:"ip" valid:"required,ip"`
	Port int    `yaml:"port" valid:"required"`
}

// Validate validates the Server struct.
func (s *Server) Validate() (bool, error) {
	result, err := govalidator.ValidateStruct(s)
	return result, appendInvalid(err)
}

// Link represents a network link between two entities.
type Link struct {
	Source           string `yaml:"source" valid:"required"`
	Destination      string `yaml:"destination" valid:"required"`
	SourcePorts      string `yaml:"sourcePorts" valid:"required"`
	DestinationPorts string `yaml:"destinationPorts" valid:"required"`
}

// Validate validates the Link struct, including length of SourcePorts and DestinationPorts.
func (l *Link) Validate() (bool, error) {
	result, err := govalidator.ValidateStruct(l)
	if !result || err != nil {
		return result, appendInvalid(err)
	}

	return true, nil
}

// UserOCS represents a user-defined OCS with multiple sections.
type UserOCS struct {
	Device   string    `yaml:"device" valid:"required"`
	Sections []Section `yaml:"sections,omitempty"`
}

type UserTor struct {
	Device string `yaml:"device" valid:"required"`
	ToRs   []ToR  `yaml:"tors,omitempty"`
}

// Validate validates the UserOCS struct, including length of SourcePorts and DestinationPorts for each section.
func (uocs *UserOCS) Validate() (bool, error) {
	if uocs.Device != "" {
		if len(uocs.Sections) == 0 {
			return false, fmt.Errorf("UserOCS '%s' must have at least one Section", uocs.Device)
		}

		for _, section := range uocs.Sections {
			if result, err := section.Validate(); err != nil {
				return result, err
			}
		}
	}

	result, err := govalidator.ValidateStruct(uocs)
	return result, appendInvalid(err)
}

func (utor *UserTor) Validate() (bool, error) {
	if utor.Device != "" {
		if len(utor.ToRs) == 0 {
			return false, fmt.Errorf("UserOCS '%s' must have at least one Section", utor.Device)
		}

		for _, tor := range utor.ToRs {
			if result, err := tor.Validate(); err != nil {
				return result, err
			}
		}
	}

	result, err := govalidator.ValidateStruct(utor)
	return result, appendInvalid(err)
}

// Section represents a section of an OCS.
type Section struct {
	Name             string `yaml:"name" valid:"required"`
	Ports            string `yaml:"ports" valid:"required"`
	SourcePorts      string `yaml:"sourcePorts" valid:"required"`
	DestinationPorts string `yaml:"destinationPorts" valid:"required"`
}

// Validate validates the Section struct, ensuring SourcePorts and DestinationPorts have equal lengths.
func (section *Section) Validate() (bool, error) {
	result, err := govalidator.ValidateStruct(section)
	if !result || err != nil {
		return result, appendInvalid(err)
	}

	return true, nil
}

type ToR struct {
	Name  string `yaml:"name" valid:"required"`
	Ports string `yaml:"ports" valid:"required"`
}

func (tor *ToR) Validate() (bool, error) {
	result, err := govalidator.ValidateStruct(tor)
	if !result || err != nil {
		return result, appendInvalid(err)
	}

	return true, nil
}

// Logger represents logger configuration.
type Logger struct {
	Enable       bool   `yaml:"enable" valid:"type(bool)"`
	Level        string `yaml:"level" valid:"required,in(trace|debug|info|warn|error|fatal|panic)"`
	ReportCaller bool   `yaml:"reportCaller" valid:"type(bool)"`
}

// Validate validates the Logger struct.
func (l *Logger) Validate() (bool, error) {
	result, err := govalidator.ValidateStruct(l)
	return result, appendInvalid(err)
}

// appendInvalid appends and formats validation errors.
func appendInvalid(err error) error {
	if err == nil {
		return nil
	}

	var errs govalidator.Errors

	switch e := err.(type) {
	case govalidator.Errors:
		for _, errItem := range e.Errors() {
			errs = append(errs, fmt.Errorf("Invalid %s", errItem))
		}
	default:
		errs = append(errs, fmt.Errorf("Invalid %w", err))
	}

	return errs
}

// LoadConfig reads and parses the YAML configuration file.
func LoadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		logger.CfgLog.Errorf("Failed to read config file: %v", err)
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		logger.CfgLog.Errorf("Failed to unmarshal config: %v", err)
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if valid, err := config.Validate(); !valid {
		logger.CfgLog.Errorf("Config validation failed: %v", err)
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	logger.CfgLog.Infof("Configuration loaded successfully from [%s]", path)
	return &config, nil
}

// SetLogLevel sets the logging level.
func (c *Config) SetLogLevel(level string) {
	c.Lock()
	defer c.Unlock()

	if c.Logger == nil {
		logger.CfgLog.Warnf("Logger should not be nil")
		c.Logger = &Logger{
			Level: level,
		}
	} else {
		c.Logger.Level = level
		logger.CfgLog.Infof("Logger level set to %s", level)
	}
}

// SetLogReportCaller sets the logger's ReportCaller field.
func (c *Config) SetLogReportCaller(reportCaller bool) {
	c.Lock()
	defer c.Unlock()

	if c.Logger == nil {
		logger.CfgLog.Warnf("Logger should not be nil")
		c.Logger = &Logger{
			Level:        "info",
			ReportCaller: reportCaller,
		}
	} else {
		c.Logger.ReportCaller = reportCaller
		logger.CfgLog.Infof("Logger ReportCaller set to %v", reportCaller)
	}
}

// GetLogEnable retrieves the logger's Enable field.
func (c *Config) GetLogEnable() bool {
	c.RLock()
	defer c.RUnlock()
	if c.Logger == nil {
		logger.CfgLog.Warnf("Logger should not be nil")
		return false
	}
	return c.Logger.Enable
}

// GetLogLevel retrieves the logger's Level field.
func (c *Config) GetLogLevel() string {
	c.RLock()
	defer c.RUnlock()
	if c.Logger == nil {
		logger.CfgLog.Warnf("Logger should not be nil")
		return "info"
	}
	return c.Logger.Level
}

// GetLogReportCaller retrieves the logger's ReportCaller field.
func (c *Config) GetLogReportCaller() bool {
	c.RLock()
	defer c.RUnlock()
	if c.Logger == nil {
		logger.CfgLog.Warnf("Logger should not be nil")
		return false
	}
	return c.Logger.ReportCaller
}

// SetLogEnable sets the logger's Enable field.
func (c *Config) SetLogEnable(enable bool) {
	c.Lock()
	defer c.Unlock()

	if c.Logger == nil {
		logger.CfgLog.Warnf("Logger should not be nil")
		c.Logger = &Logger{
			Enable: enable,
			Level:  "info",
		}
	} else {
		c.Logger.Enable = enable
		logger.CfgLog.Infof("Logger enable set to %v", enable)
	}
}
