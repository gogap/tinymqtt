package memory

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gogap/config"
	"github.com/gogap/tinymqtt/store"
)

func init() {
	store.RegisterStore("memory", NewInMemoryStore)
}

func NewInMemoryStore(storeConf config.Configuration) (mqtt.Store, error) {
	store := mqtt.NewMemoryStore()
	return store, nil
}
