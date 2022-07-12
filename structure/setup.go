package structure

const (
	ShardNum         = 4
	AccountNum       = 2
	CLIENT_MAX       = 3
	SIGN_VERIFY_TIME = 4    //millisecond
	TX_NUM           = 3000 //per shard per catagory

	Server1 = "ws://172.17.0.2:8080"
	Server2 = "ws://172.168.66.14:8080"
	Server3 = "ws://172.168.66.14:8080"
)

var Source = InitController(ShardNum, AccountNum)

// var ServerSource = InitForwarding(Server1, Server2, Server3)
