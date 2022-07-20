package structure

const (
	ShardNum         = 2
	AccountNum       = 2
	CLIENT_MAX       = 3
	SIGN_VERIFY_TIME = 4    //millisecond
	TX_NUM           = 3000 //per shard per catagory

	Server1 = "192.168.199.151"
	Server2 = "172.168.66.14"
	Server3 = "172.168.66.14"
)

var Source = InitController(ShardNum, AccountNum)

// var ServerSource = InitForwarding(Server1, Server2, Server3)
