package config

import (
	"emperror.dev/errors"
	"github.com/BurntSushi/toml"
	"github.com/je4/utils/v2/pkg/config"
	"io/fs"
	"os"
)

type ClientBaseQuery struct {
	Field  string   `toml:"field"`
	Values []string `toml:"values"`
}

type Client struct {
	Name      string            `toml:"name"`
	Apikey    string            `toml:"apikey"`
	JWTSecret string            `toml:"jwtsecret"`
	Groups    []string          `toml:"groups"`
	OR        []ClientBaseQuery `toml:"or"`
	AND       []ClientBaseQuery `toml:"and"`
}

type ElasticSearchConfig struct {
	Endpoint []string         `toml:"endpoint"`
	Index    string           `toml:"index"`
	ApiKey   config.EnvString `toml:"apikey"`
	Debug    bool             `toml:"debug"`
}

type RevCatConfig struct {
	LocalAddr    string `toml:"localaddr"`
	ExternalAddr string `toml:"externaladdr"`
	TLSCert      string `toml:"tlscert"`
	TLSKey       string `toml:"tlskey"`

	LogFile  string `toml:"logfile"`
	LogLevel string `toml:"loglevel"`

	ElasticSearch ElasticSearchConfig `toml:"elasticsearch"`

	Client []*Client `toml:"client"`
}

func LoadRevCatConfig(fSys fs.FS, fp string, conf *RevCatConfig) error {
	if _, err := fs.Stat(fSys, fp); err != nil {
		path, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "cannot get current working directory")
		}
		fSys = os.DirFS(path)
		fp = "gndservice.toml"
	}
	data, err := fs.ReadFile(fSys, fp)
	if err != nil {
		return errors.Wrapf(err, "cannot read file [%v] %s", fSys, fp)
	}
	_, err = toml.Decode(string(data), conf)
	if err != nil {
		return errors.Wrapf(err, "error loading config file %v", fp)
	}
	return nil
}
