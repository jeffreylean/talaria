package script

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"plugin"
	"time"

	loaderpkg "github.com/kelindar/loader"
)

type mainFunc = func(map[string]interface{}) (interface{}, error)

type PluginLoader struct {
	Loader
	main         mainFunc
	functionName string
}

func NewPluginLoader(functionName string) *PluginLoader {
	return &PluginLoader{
		Loader:       Loader{loaderpkg.New()},
		functionName: functionName,
	}
}

func (h *PluginLoader) Load(uriOrCode string) (Handler, error) {
	log.Println("LoadGoPlugin: ", uriOrCode)
	// try to download it
	if err := h.watch(uriOrCode, h.updateGoPlugin); err != nil {
		return nil, err
	}

	return h, nil
}

func (h *PluginLoader) String() string { return pluginType }

func (h *PluginLoader) Value(row map[string]interface{}) (interface{}, error) {
	return h.main(row)
}

func (h *PluginLoader) updateGoPlugin(r io.Reader) error {
	tmpFileName := fmt.Sprintf("%s.so", time.Now().Format("20060102150405"))
	tmpFile, err := ioutil.TempFile("", tmpFileName)
	if err != nil {
		return err
	}

	written, err := io.Copy(tmpFile, r)
	if err != nil {
		return err
	}
	if written == 0 {
		return errors.New("PluginLoader load plugin content failed, content was empty")
	}

	log.Printf("updateGoPlugin: write to file %s, try to open %s: ", tmpFileName, tmpFile.Name())
	p, err := plugin.Open(tmpFile.Name())
	if err != nil {
		return err
	}

	f, err := p.Lookup(h.functionName)
	if err != nil {
		return err
	}

	ok := false
	h.main, ok = f.(mainFunc)
	if !ok {
		return errors.New("type assertions on plugin funtion failed")
	}
	return nil
}
