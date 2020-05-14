package tinymqtt

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gogap/config"
	"github.com/sirupsen/logrus"

	"github.com/gogap/tinymqtt/store"
	_ "github.com/gogap/tinymqtt/store/file"
	_ "github.com/gogap/tinymqtt/store/memory"
)

type SubscribeOption struct {
	Topic   string
	Qos     byte
	Handler mqtt.MessageHandler
}

type MQTTClient struct {
	client   mqtt.Client
	broker   string
	username string
	quiesce  uint

	subscribes []SubscribeOption
}

func NewMQTTClient(conf config.Configuration, subscribes ...SubscribeOption) (ret *MQTTClient, err error) {

	for i := 0; i < len(subscribes); i++ {
		if len(subscribes[i].Topic) == 0 {
			err = fmt.Errorf("Topic name could not be empty")
			return
		}
	}

	clientConf := conf.GetConfig("client")

	clientID := clientConf.GetString("client-id")

	brokerServer := clientConf.GetString("broker-server")
	keepAlive := clientConf.GetTimeDuration("keep-alive")
	pingTimeout := clientConf.GetTimeDuration("ping-timeout")
	cleanSession := clientConf.GetBoolean("clean-session")
	orderMatters := clientConf.GetBoolean("order_matters", true)
	quiesce := uint(clientConf.GetInt32("quiesce"))
	if quiesce == 0 {
		quiesce = 250 //ms
	}

	logrus.WithField("broker", brokerServer).Debugln("init mqtt client")

	credentialMode := clientConf.GetString("credential.mode", "normal")
	credentialName := clientConf.GetString("credential.name")

	if len(credentialName) == 0 {
		err = fmt.Errorf("credential name is empty")
		return
	}

	username := conf.GetString("credentials." + credentialName + ".username")
	password := conf.GetString("credentials." + credentialName + ".password")

	if credentialMode == "aliyun-signature" {

		instanceID := clientConf.GetString("instance-id")
		deviceID := clientConf.GetString("device-id")
		groupID := clientConf.GetString("group-id")

		if len(deviceID) == 0 {
			if deviceIDEnv := clientConf.GetString("device-id-env"); len(deviceIDEnv) > 0 {
				logrus.WithField("env", deviceIDEnv).Debugln("get device-id from env")
				deviceID = os.Getenv(deviceIDEnv)
			} else if deviceIDFile := clientConf.GetString("device-id-file"); len(deviceIDFile) > 0 {
				logrus.WithField("file", deviceIDFile).Debugln("get device-id from file")
				var deviceData []byte
				deviceData, err = ioutil.ReadFile(deviceIDFile)
				if err != nil {
					err = fmt.Errorf("read device id file failure, path: %s, err: %w", deviceIDFile, err)
					return
				}
				deviceID = strings.TrimSpace(string(deviceData))
			}
		}

		if len(deviceID) == 0 {
			err = fmt.Errorf("could not get device_id")
			return
		}

		accessKeyID := username
		accessKeySecret := password

		clientID, username, password, err = calcAliyunSignature(accessKeyID, accessKeySecret, instanceID, groupID, deviceID)
		if err != nil {
			return
		}
	}

	if len(clientID) == 0 {
		err = fmt.Errorf("client-id could not be empty")
		return
	}

	opts := mqtt.NewClientOptions()

	storeConf := clientConf.GetConfig("store")

	storeProvier := storeConf.GetString("provider")
	if len(storeProvier) > 0 {
		var mqttStore mqtt.Store
		mqttStore, err = store.NewStore(storeProvier, storeConf)
		if err != nil {
			return
		}
		opts.SetStore(mqttStore)
	}

	autoReconnect := clientConf.GetBoolean("auto-reconnect", true)

	opts.AddBroker(brokerServer)
	opts.SetClientID(clientID)
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.SetCleanSession(cleanSession)
	opts.SetKeepAlive(keepAlive)
	opts.SetPingTimeout(pingTimeout)
	opts.SetAutoReconnect(autoReconnect)
	opts.SetOrderMatters(orderMatters)

	client := mqtt.NewClient(opts)

	ret = &MQTTClient{
		client:     client,
		broker:     brokerServer,
		username:   username,
		quiesce:    quiesce,
		subscribes: subscribes,
	}

	return
}

func hidePassword(pwd string, l int) string {
	return pwd[0:l] + "*****" + pwd[len(pwd)-l:]
}

func calcAliyunSignature(accessKeyID, accessKeySecret, instanceID, groupID, deviceID string) (clientID, username, password string, err error) {

	if len(accessKeyID) == 0 || len(accessKeySecret) == 0 {
		err = fmt.Errorf("accessKeyID or accessKeySecret is empty")
		return
	}

	if len(instanceID) == 0 {
		err = fmt.Errorf("instanceID is empty")
		return
	}

	if len(groupID) == 0 {
		err = fmt.Errorf("groupID is empty")
		return
	}

	if len(deviceID) == 0 {
		err = fmt.Errorf("deviceID is empty")
		return
	}

	logrus.WithFields(
		logrus.Fields{
			"access_key_id":     accessKeyID,
			"access_key_secret": hidePassword(accessKeySecret, 4),
			"instance_id":       instanceID,
			"group_id":          groupID,
			"device_id":         deviceID,
		},
	).Debugln("cacl aliyun signature")

	username = "Signature" + "|" + accessKeyID + "|" + instanceID
	clientID = groupID + "@@@" + deviceID

	pwdHmac := hmac.New(sha1.New, []byte(accessKeySecret))
	pwdHmac.Write([]byte(clientID))
	pwdBytes := pwdHmac.Sum(nil)

	password = base64.StdEncoding.EncodeToString(pwdBytes)

	logrus.WithFields(
		logrus.Fields{
			"username":  username,
			"password":  hidePassword(password, 4),
			"client_id": clientID,
		},
	).Debugln("username and password calced")

	return
}

func (p *MQTTClient) Start() (err error) {

	logrus.Debugln("MQTTClient begin to connect to server")

	if token := p.client.Connect(); token.Wait() && token.Error() != nil {
		err = token.Error()
		return
	}

	logrus.Debugln("MQTTClient client started()")

	for i := 0; i < len(p.subscribes); i++ {
		if token := p.client.Subscribe(p.subscribes[i].Topic, p.subscribes[i].Qos, p.subscribes[i].Handler); token.Wait() && token.Error() != nil {
			err = token.Error()
			return
		}
	}

	logrus.Debugln("MQTTClient client subscribed successful")

	return
}

func (p *MQTTClient) Stop() (err error) {

	var topicList []string

	for i := 0; i < len(p.subscribes); i++ {
		topicList = append(topicList, p.subscribes[i].Topic)
	}

	if token := p.client.Unsubscribe(topicList...); token.Wait() && token.Error() != nil {
		err = token.Error()
		p.client.Disconnect(p.quiesce)
		return
	}

	p.client.Disconnect(p.quiesce)

	return
}

type MQSendResult struct {
	Topic    string `json:"topic"`
	Broker   string `json:"broker"`
	Username string `json:"username"`
	MsgID    uint16 `json:"msg_id"`
	Qos      byte   `json:"qos"`
	Retained bool   `json:"retained"`
}

func (p *MQTTClient) SendMessage(topic string, qos byte, retained bool, msg []byte) (ret MQSendResult, err error) {

	token := p.client.Publish(topic, qos, retained, msg)

	if token.Wait() && token.Error() != nil {
		err = token.Error()
		return
	}

	pubToken := token.(*mqtt.PublishToken)

	ret = MQSendResult{
		Topic:    topic,
		Broker:   p.broker,
		Username: p.username,
		MsgID:    pubToken.MessageID(),
		Qos:      qos,
		Retained: retained,
	}

	return
}
