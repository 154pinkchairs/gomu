package yaml

import (
	//"bufio"
	//"bytes"
	"log"
	//"os"

	"github.com/issadarkthing/gomu"

	"github.com/gookit/config/v2/yamlv3"
)

func Parse(confFile string) (*Config, error) {
	config := Config{}

	err := config.BindStruct("conf", &config)
	if err != nil {
		r := recover()
		if r != nil {
			log.Println("recover from panic", r)
		}
		return nil, err
	}

	config.WithOptions(func(opt *config.Options) {
		opt.ConfigFile = confFile
		opt.Parser = yamlv3.Parser()
		opt.DecoderConfig.TagName = "yaml"
	})
}
