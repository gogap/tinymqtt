package store

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gogap/config"
)

type NewStoreFunc func(storeConf config.Configuration) (mqtt.Store, error)

var (
	stores = make(map[string]NewStoreFunc)
)

func RegisterStore(driverName string, fn NewStoreFunc) (err error) {

	if len(driverName) == 0 {
		err = fmt.Errorf("store driver name is empty")
		return
	}

	if fn == nil {
		err = fmt.Errorf("driver of %s's NewStoreFunc is nil")
		return
	}

	if _, exist := stores[driverName]; exist {
		err = fmt.Errorf("store driver of: %s, already registered")
		return
	}

	stores[driverName] = fn

	return
}

func NewStore(driverName string, storeConf config.Configuration) (store mqtt.Store, err error) {

	if len(driverName) == 0 {
		err = fmt.Errorf("store driver name is empty")
		return
	}

	fn, exist := stores[driverName]
	if !exist {
		err = fmt.Errorf("store driver of: %s, not registered", driverName)
		return
	}

	return fn(storeConf)
}
