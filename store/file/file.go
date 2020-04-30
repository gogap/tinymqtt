package file

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gogap/config"
	"github.com/gogap/tinymqtt/store"
)

func init() {
	store.RegisterStore("file", NewFileStore)
}

func NewFileStore(storeConf config.Configuration) (mqtt.Store, error) {
	dir := storeConf.GetString("directory")
	store := mqtt.NewFileStore(dir)
	return store, nil
}
