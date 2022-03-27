package core

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/TheRebelOfBabylon/Conduit/utils"
	"github.com/google/go-cmp/cmp"
)

// TestInitConfigNoYAML ensures that if no .yaml is found, a default config is produced
func TestInitConfigNoYAML(t *testing.T) {
	home_dir := utils.AppDataDir("conduit", false)
	config_file_path := path.Join(home_dir, config_file_name)
	// check if config.yaml exists, if yes, we copy and rename it, delete the config.yaml and then teardown function recopies the copy as config.yaml
	if utils.FileExists(config_file_path) {
		// Open config file
		config_copy_path := path.Join(home_dir, "config.copy.yaml")
		config_file, err := ioutil.ReadFile(config_file_path)
		if err != nil {
			t.Fatalf("Error opening config.yaml: %v", err)
		}
		err = ioutil.WriteFile(config_copy_path, config_file, 0666)
		if err != nil {
			t.Fatalf("Error copying config.yaml to config.copy.yaml: %v", err)
		}
		err = os.Remove(config_file_path)
		if err != nil {
			t.Fatalf("Errror deleting original config.yaml: %v", err)
		}
		defer func() {
			config_copy_file, err := ioutil.ReadFile(config_copy_path)
			if err != nil {
				t.Fatalf("Error opening config.copy.yaml: %v", err)
			}
			err = ioutil.WriteFile(config_file_path, config_copy_file, 0666)
			if err != nil {
				t.Fatalf("Error copying config.copy.yaml to config.yaml: %v", err)
			}
			err = os.Remove(config_copy_path)
			if err != nil {
				t.Fatalf("Errror deleting config.copy.yaml: %v", err)
			}
		}()
	}
	config, err := InitConfig()
	if err != nil {
		t.Errorf("%s", err)
	}
	defaultCfg := default_config()
	if !cmp.Equal(*config, *defaultCfg) {
		t.Errorf("InitConfig did not produce a default config when config.yaml was not present")
	}
}

// TestInitConfigFromYAML ensures that a InitConfig properly reads config files
func TestInitConfigFromYAML(t *testing.T) {
	homeDir := utils.AppDataDir("conduit", false)
	// first check if home dir exists
	if !utils.FileExists(homeDir) {
		err := os.Mkdir(homeDir, 0775)
		if err != nil {
			t.Fatalf("Error creating fmtd directory: %v", err)
		}
	}

	config_file_path := path.Join(homeDir, config_file_name)
	// check if config.yaml exists, if yes, we copy and rename it, delete the config.yaml and then teardown function recopies the copy as config.yaml
	if utils.FileExists(config_file_path) {
		// Open config file
		config_copy_path := path.Join(homeDir, "config.copy.yaml")
		config_file_bytes, err := ioutil.ReadFile(config_file_path)
		if err != nil {
			t.Fatalf("Error opening config.yaml: %v", err)
		}
		err = ioutil.WriteFile(config_copy_path, config_file_bytes, 0666)
		if err != nil {
			t.Fatalf("Error copying config.yaml to config.copy.yaml: %v", err)
		}
		err = os.Remove(config_file_path)
		if err != nil {
			t.Fatalf("Errror deleting original config.yaml: %v", err)
		}
		// create blank config test file
		_, err = os.OpenFile(config_file_path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			t.Fatalf("Error creating config.yaml: %v", err)
		}
		defer func() {
			os.Remove(config_file_path)
			config_copy_file, err := ioutil.ReadFile(config_copy_path)
			if err != nil {
				t.Fatalf("Error opening config.copy.yaml: %v", err)
			}
			err = ioutil.WriteFile(config_file_path, config_copy_file, 0666)
			if err != nil {
				t.Fatalf("Error copying config.copy.yaml to config.yaml: %v", err)
			}
			err = os.Remove(config_copy_path)
			if err != nil {
				t.Fatalf("Errror deleting config.copy.yaml: %v", err)
			}
		}()
	} else {
		defer os.Remove(config_file_path)
	}
	// create blank config test file
	config_file, err := os.OpenFile(config_file_path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		t.Fatalf("Error creating/opening config.yaml: %v", err)
	}
	// write to yaml file
	d_config := Config{
		DefaultDir:    true,
		ConduitDir:    default_dir(), // this is not OS agnostic
		ConsoleOutput: true,
	}
	_, err = config_file.WriteString(fmt.Sprintf("DefaultLogDir: %v\n", d_config.DefaultDir))
	if err != nil {
		t.Errorf("%s", err)
	}
	_, err = config_file.WriteString(fmt.Sprintf("LogFileDir: %v\n", d_config.ConduitDir))
	if err != nil {
		t.Errorf("%s", err)
	}
	_, err = config_file.WriteString(fmt.Sprintf("ConsoleOutput: %v\n", d_config.ConsoleOutput))
	if err != nil {
		t.Errorf("%s", err)
	}
	config_file.Sync()
	config_file.Close()
	config, err := InitConfig()
	if err != nil {
		t.Errorf("%s", err)
	}
	if !cmp.Equal(*config, d_config) {
		t.Errorf("InitConfig did not properly read the config file: %v", *config)
	}
}

// TestDefaultDir tests that default_dir returns the expected default directory
func TestDefaultDir(t *testing.T) {
	home_dir := utils.AppDataDir("conduit", false)
	log_dir := home_dir
	if log_dir != default_dir() {
		t.Errorf("default_log_dir not returning expected directory. Expected: %s\tReceived: %s", log_dir, default_dir())
	}
}

// TestDefaultConfig checks if default_config does return the expected default config struct
func TestDefaultConfig(t *testing.T) {
	d_config := Config{
		DefaultDir:    true,
		ConduitDir:    default_dir(),
		ConsoleOutput: true,
	}
	defaultCfg := default_config()
	if !cmp.Equal(d_config, *defaultCfg) {
		t.Errorf("default_config not returning expected config. Expected: %v\tReceived: %v", d_config, default_config())
	}
}
