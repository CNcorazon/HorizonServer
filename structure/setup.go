package structure

const ShardNum = 4
const AccountNum = 2

const CLIENT_MAX = 3

var Source = InitController(ShardNum, AccountNum)

const SIGN_VERIFY_TIME = 4 //millisecond
const TX_NUM = 3000        //per shard per catagory
