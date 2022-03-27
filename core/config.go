package core

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"reflect"

	"github.com/TheRebelOfBabylon/Conduit/utils"
	yaml "gopkg.in/yaml.v2"
)

// Config is the object which will hold all of the config parameters
type Config struct {
	DefaultDir    bool   `yaml:"DefaultDir"`
	ConduitDir    string `yaml:"ConduitDir"`
	ConsoleOutput bool   `yaml:"ConsoleOutput"`
}

var (
	config_file_name string = "config.yaml"
	default_dir             = func() string {
		return utils.AppDataDir("conduit", false)
	}
	default_config = func() *Config {
		return &Config{
			DefaultDir:    true,
			ConduitDir:    default_dir(),
			ConsoleOutput: true,
		}
	}
	conduitDirFlag *string = flag.String(
		"conduit-dir",
		default_dir(),
		fmt.Sprintf(`The base directory that contains conduit's data, logs,
		configuration file, etc. (default: %v)`, default_dir()),
	)
	consoleOutputFlag *bool = flag.Bool(
		"console-output",
		true,
		"Select true or false to print log information to the console",
	)
)

// InitConfig returns the `Config` struct with either default values, values specified in `config.yaml` or command line flags
func InitConfig() (*Config, error) {
	// Check if fmtd directory exists, if no then create it
	if !utils.FileExists(utils.AppDataDir("conduit", false)) {
		err := os.Mkdir(utils.AppDataDir("conduit", false), 0775)
		if err != nil {
			log.Println(err)
		}
	}
	config := &Config{}
	if utils.FileExists(path.Join(default_dir(), config_file_name)) {
		filename, _ := filepath.Abs(path.Join(default_dir(), config_file_name))
		config_file, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Println(err)
			return default_config(), nil
		}
		err = yaml.Unmarshal(config_file, config)
		if err != nil {
			log.Println(err)
			config = default_config()
		} else {
			// Need to check if any config parameters aren't defined in `config.yaml` and assign them a default value
			config = check_yaml_config(config)
		}
	} else {
		config = default_config()
	}
	// now to parse the flags
	argparse(config)
	return config, nil
}

// argparse parses the conduit command flags and stores values in the Config struct
func argparse(cfg *Config) {
	flag.Parse()
	if utils.FileExists(*conduitDirFlag) {
		cfg.ConduitDir = *conduitDirFlag
	}
	cfg.ConsoleOutput = *consoleOutputFlag
}

// change_field changes the value of a specified field from the config struct
func change_field(field reflect.Value, new_value interface{}) {
	if field.IsValid() {
		if field.CanSet() {
			f := field.Kind()
			switch f {
			case reflect.String:
				if v, ok := new_value.(string); ok {
					field.SetString(v)
				} else {
					log.Fatal(fmt.Sprintf("Type of new_value: %v does not match the type of the field: string", new_value))
				}
			case reflect.Bool:
				if v, ok := new_value.(bool); ok {
					field.SetBool(v)
				} else {
					log.Fatal(fmt.Sprintf("Type of new_value: %v does not match the type of the field: bool", new_value))
				}
			case reflect.Int64:
				if v, ok := new_value.(int64); ok {
					field.SetInt(v)
				} else {
					log.Fatal(fmt.Sprintf("Type of new_value: %v does not match the type of the field: int64", new_value))
				}
			}
		}
	}
}

// check_yaml_config iterates over the Config struct fields and changes blank fields to default values
func check_yaml_config(config *Config) *Config {
	pv := reflect.ValueOf(config)
	v := pv.Elem()
	field_names := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		field_name := field_names.Field(i).Name
		switch field_name {
		case "ConduitDir":
			if f.String() == "" {
				change_field(f, default_dir())
				dld := v.FieldByName("DefaultDir")
				change_field(dld, true)
			}
		}
	}
	return config
}
