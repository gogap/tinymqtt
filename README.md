TINYMQTT
=========

```hocon
credentials = {
    c1 = {
        username = "username"
        password = "password"
    }
}

client {

    credential {
        mode = "normal" // "aliyun-signature"
        name = "c1"
    }


    // while credential.mode = aliyun-signature, choice one config item as following for device-id
    device-id      = "1001" // priority 1
    device-id-env  = "HOST" // priority 2
    device-id-file = "/data/local/tmp/mqtt_device_id"  // priority 3

    //while credential.mode = aliyun-signature
    group-id = "GID_TEST"

    client-id       = "" // no need while credential.mode = aliyun-signature
    broker-server   = "127.0.0.1:1883"

    keep-alive    = 3s
    ping-timeout  = 1s
    clean-session = false
    quiesce = 250

    store {
        provider  = memory
    }
}
```