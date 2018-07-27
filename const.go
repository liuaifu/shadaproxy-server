package main

const PKT_HEARTBEAT uint32 = 0x0
const PKT_REPORT_KEY uint32 = 0x1
const PKT_CONNECT uint32 = 0x2
const PKT_FORWARD_CLIENT_DATA uint32 = 0x3
const PKT_FORWARD_SERVER_DATA uint32 = 0x4

type Head struct {
	pkt_type    uint32
	body_length uint32
	result      int32 //请求时无意义，应答填消息处理结果，1表示成功，其它为失败
}
