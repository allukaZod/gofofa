package gofofa

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
)

type accountInfo struct {
	Email          string   `json:"email"`
	Key            string   `json:"key"`
	VIP            bool     `json:"isvip"`
	VIPLevel       VipLevel `json:"vip_level"`        // vip level
	FCoin          int      `json:"fcoin"`            // fcoin count
	RemainApiQuery int      `json:"remain_api_query"` // available query
	RemainApiData  int      `json:"remain_api_data"`  // available data amount
}

var (
	validAccounts = []accountInfo{
		{"a@a.com", "11111", false, VipLevelNone, 0, 0, 0},      // 注册用户
		{"b@b.com", "22222", true, VipLevelNormal, 10, 0, 0},    // 普通会员
		{"c@c.com", "33333", true, VipLevelAdvanced, 0, 0, 0},   // 高级会员
		{"d@d.com", "44444", true, VipLevelEnterprise, 0, 0, 0}, // 企业会员
		{"e@e.com", "55555", false, VipLevelNone, 10, 0, 0},     // 注册用户有F币

		{"g@g.com", "77777", true, VipLevelSubPersonal, 10, 10, 100}, // 订阅个人
		{"h@h.com", "88888", true, VipLevelSubPro, 0, 0, 0},          // 订阅专业
		{"i@i.com", "99999", true, VipLevelSubBuss, 0, 0, 0},         // 订阅商业

		{"red@fofa.info", "10001", true, VipLevelRed, 0, 0, 0},     // 红队
		{"sub@fofa.info", "10002", true, VipLevelStudent, 0, 0, 0}, // 教育

		{"never@fofa.info", "10003", true, VipLevelNever, 0, 0, 0}, // 不可能的等级
	}

	queryHander = func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/info/my":
			account := checkAccount(r.FormValue("email"), r.FormValue("key"))
			if account == nil {
				w.Write([]byte(`{"error":true,"errmsg":"[-700] Account Invalid"}`))
				return
			}
			b, _ := json.Marshal(account)
			w.Write(b)
			//w.Write([]byte(`{"error":false,"email":"` + account.Email + `","fcoin":` + strconv.Itoa(account.FCoin) + `,"isvip":` + strconv.FormatBool(account.VIP) + `,"vip_level":` + strconv.Itoa(int(account.VIPLevel)) + `}`))

		case "/api/v1/search/all":
			account := checkAccount(r.FormValue("email"), r.FormValue("key"))
			if account == nil {
				w.Write([]byte(`{"error":true,"errmsg":"[-700] Account Invalid"}`))
				return
			}

			// 参数错误
			q, err := base64.StdEncoding.DecodeString(r.FormValue("qbase64"))
			if err != nil || len(q) == 0 {
				w.Write([]byte(`{"error":true,"errmsg":"[-4] Params Error"}`))
				return
			}

			// 注册会员权限不够的错误返回
			if account.VIPLevel != VipLevelEnterprise && strings.Contains(r.FormValue("fields"), "fid") {
				w.Write([]byte(`{"error":true,"errmsg":"[820001] 没有权限搜索fid字段"}`))
				return
			}

			// 0 长度直接返回错误
			if r.FormValue("size") == "0" {
				w.Write([]byte(`{"error":true,"errmsg":"[51] The Size value ` + "`" + `0` + "`" + ` must be between 1 and 10000"}`))
				return
			}

			switch string(q) {
			case "aaa=bbb":
				// 不支持的语法
				w.Write([]byte(`{"error":true,"errmsg":"[820000] FOFA Query Syntax Incorrect"}`))
				return
			case "port=100000":
				// 构造0个数据
				w.Write([]byte(`{"error":false,"size":0,"page":1,"mode":"extended","query":"port=\"100000\"","results":[]}`))
				return
			case "port=100001":
				// 构造非正常格式
				w.Write([]byte(`{"error":false,"size":0,"page":1,"mode":"extended","query":"port=\"100000\"","results":"test"}`))
				return
			case "port=50000":
				// 数据不够
				w.Write([]byte(`{"error":false,"size":9,"page":1,"mode":"extended","query":"port=\"50000\"","results":["118.190.75.134","34.83.32.116","117.4.67.26",":0",":0",":0","176.198.13.22","81.174.169.62","23.42.6.133"]}`))
				return
			case "port=80":
				switch r.FormValue("fields") {
				case "host":
					// 测试单个字段
					w.Write([]byte(`{"error":false,"size":470262270,"page":1,"mode":"extended","query":"port=\"80\"","results":["118.190.75.134","34.83.32.116","117.4.67.26",":0",":0",":0","176.198.13.22","81.174.169.62","23.42.6.133","webdisk.dutadamaijawatengah.id"]}`))
					return
				case "ip,port", "":
					// 多字段测试
					switch r.FormValue("size") {
					case "1":
						// 主要用于取数据量
						w.Write([]byte(`{"error":false,"size":12345678,"page":1,"mode":"extended","query":"port=\"80\"","results":[["94.130.128.248","80"]]}`))
					case "10":
						w.Write([]byte(`{"error":false,"size":470293950,"page":1,"mode":"extended","query":"port=\"80\"","results":[["94.130.128.248","80"],["186.6.19.151","80"],["72.247.70.195","80"],["18.66.199.67","80"],["91.122.52.148","80"],["113.23.57.252","80"],["54.144.154.222","80"],["188.223.2.247","80"],["50.213.108.254","80"],["34.237.16.144","80"]]}`))
					case "100":
						w.Write([]byte(`{"error":false,"size":470293950,"page":1,"mode":"extended","query":"port=\"80\"","results":[["108.138.246.93","80"],["52.128.141.110","80"],["103.230.59.134","80"],["122.114.69.40","80"],["62.244.6.72","80"],["34.192.58.186","80"],["140.248.147.104","80"],["13.67.133.203","80"],["54.210.102.226","80"],["45.39.132.86","80"],["178.154.226.2","80"],["61.60.167.3","80"],["144.24.180.239","80"],["188.107.149.238","80"],["152.136.158.163","80"],["171.107.184.166","80"],["195.4.138.56","80"],["185.26.157.113","80"],["184.106.212.76","80"],["195.4.138.56","80"],["138.68.241.205","80"],["176.35.87.4","80"],["165.22.55.31","80"],["5.196.29.205","80"],["184.106.212.76","80"],["178.79.169.83","80"],["66.29.132.114","80"],["208.109.67.142","80"],["116.202.113.38","80"],["185.39.147.182","80"],["104.21.61.158","80"],["23.63.146.82","80"],["13.226.80.37","80"],["218.92.220.94","80"],["216.188.204.27","80"],["64.227.111.74","80"],["54.192.219.37","80"],["199.3.18.10","80"],["177.8.60.102","80"],["194.99.46.195","80"],["54.163.237.67","80"],["50.87.248.237","80"],["23.206.223.15","80"],["172.67.132.241","80"],["18.118.127.67","80"],["93.113.147.35","80"],["18.116.168.156","80"],["195.4.138.56","80"],["217.120.42.56","80"],["156.226.211.183","80"],["72.247.226.203","80"],["38.63.11.237","80"],["172.96.182.239","80"],["186.106.216.172","80"],["82.118.214.242","80"],["189.128.161.107","80"],["82.146.169.152","80"],["185.229.112.135","80"],["14.255.48.96","80"],["186.62.196.124","80"],["95.168.171.235","80"],["54.187.142.21","80"],["35.194.245.35","80"],["94.46.14.148","80"],["115.73.182.162","80"],["34.208.186.163","80"],["75.98.163.225","80"],["81.21.234.21","80"],["3.24.226.145","80"],["67.186.81.216","80"],["144.76.142.17","80"],["186.104.161.201","80"],["31.44.224.52","80"],["147.182.240.237","80"],["188.166.172.171","80"],["94.182.196.158","80"],["93.185.15.37","80"],["186.6.171.155","80"],["46.10.116.24","80"],["59.31.124.61","80"],["121.29.2.234","80"],["46.23.176.147","80"],["185.37.226.108","80"],["3.209.70.41","80"],["122.55.67.112","80"],["2.193.194.96","80"],["190.32.208.91","80"],["189.215.3.76","80"],["45.33.16.216","80"],["63.42.2.205","80"],["98.129.11.47","80"],["39.1.88.148","80"],["40.97.221.86","80"],["185.204.92.201","80"],["62.174.19.115","80"],["23.33.246.47","80"],["23.223.188.174","80"],["174.138.165.214","80"],["200.124.28.142","80"],["156.234.146.115","80"]]}`))
					case "1000":
						w.Write([]byte(`{"error":false,"size":470293950,"page":1,"mode":"extended","query":"port=\"80\"","results":[["181.204.169.58","80"],["35.164.188.212","80"],["190.167.74.48","80"],["103.255.21.211","80"],["46.141.99.42","80"],["40.67.172.102","80"],["177.154.77.48","80"],["37.138.157.123","80"],["5.235.165.44","80"],["198.211.18.169","80"],["45.225.34.39","80"],["192.34.93.143","80"],["176.44.91.35","80"],["13.245.201.212","80"],["87.150.242.205","80"],["120.157.34.124","80"],["70.37.193.66","80"],["75.169.3.92","80"],["190.22.219.74","80"],["90.157.155.197","80"],["90.213.18.4","80"],["23.76.104.51","80"],["178.90.132.150","80"],["81.152.210.94","80"],["104.106.39.235","80"],["46.101.74.127","80"],["121.36.107.33","80"],["52.44.254.179","80"],["163.197.252.212","80"],["154.212.180.217","80"],["85.159.214.95","80"],["23.55.34.22","80"],["5.94.180.218","80"],["185.38.19.75","80"],["68.66.207.250","80"],["23.40.219.53","80"],["15.164.98.34","80"],["23.57.172.71","80"],["185.107.213.25","80"],["118.195.140.199","80"],["197.12.226.45","80"],["166.1.57.91","80"],["5.226.160.158","80"],["154.53.72.93","80"],["34.235.7.180","80"],["157.230.116.235","80"],["66.165.229.247","80"],["54.180.139.65","80"],["184.29.243.139","80"],["34.89.215.45","80"],["23.35.226.75","80"],["62.210.131.218","80"],["14.234.179.207","80"],["23.10.10.19","80"],["96.6.35.130","80"],["38.103.2.186","80"],["154.23.52.204","80"],["184.29.186.129","80"],["23.56.188.41","80"],["13.235.20.17","80"],["184.169.54.32","80"],["52.77.4.187","80"],["184.86.80.122","80"],["38.22.57.164","80"],["209.145.48.69","80"],["154.9.53.108","80"],["184.31.76.247","80"],["173.222.24.224","80"],["154.9.53.41","80"],["23.9.214.189","80"],["23.62.165.177","80"],["194.186.52.154","80"],["63.117.127.33","80"],["23.244.73.22","80"],["104.64.255.154","80"],["18.213.29.97","80"],["207.180.229.227","80"],["184.24.137.63","80"],["154.53.72.5","80"],["185.31.240.191","80"],["154.210.49.91","80"],["59.25.87.71","80"],["20.67.216.125","80"],["90.177.0.51","80"],["66.175.137.52","80"],["209.42.193.238","80"],["45.168.10.4","80"],["193.23.250.124","80"],["92.122.182.54","80"],["60.43.217.194","80"],["59.115.163.178","80"],["156.241.91.131","80"],["198.23.199.249","80"],["74.208.90.25","80"],["87.174.135.126","80"],["47.206.102.86","80"],["54.157.204.166","80"],["23.33.20.70","80"],["136.243.51.174","80"],["94.255.164.122","80"],["52.16.112.195","80"],["218.39.195.59","80"],["153.150.102.252","80"],["156.250.119.36","80"],["89.175.54.67","80"],["95.47.55.129","80"],["112.223.226.202","80"],["178.91.146.140","80"],["2.22.213.117","80"],["83.48.92.88","80"],["23.42.91.239","80"],["184.87.28.187","80"],["216.157.107.194","80"],["94.102.1.66","80"],["192.99.183.17","80"],["195.24.67.116","80"],["188.127.224.24","80"],["52.84.81.122","80"],["193.189.139.44","80"],["23.42.91.232","80"],["208.84.118.217","80"],["35.244.147.180","80"],["104.149.251.27","80"],["185.151.106.115","80"],["154.53.72.121","80"],["189.112.173.216","80"],["101.201.238.183","80"],["213.79.69.138","80"],["69.192.63.81","80"],["156.244.54.217","80"],["46.30.63.137","80"],["185.35.210.190","80"],["178.88.45.2","80"],["187.194.250.122","80"],["213.175.51.18","80"],["23.38.35.105","80"],["116.105.60.129","80"],["23.41.234.123","80"],["154.9.52.236","80"],["91.212.43.32","80"],["41.251.183.183","80"],["121.184.250.176","80"],["23.51.169.146","80"],["66.27.102.111","80"],["208.65.99.211","80"],["5.238.246.232","80"],["82.160.175.161","80"],["154.23.188.10","80"],["114.215.51.250","80"],["92.35.204.70","80"],["200.170.82.151","80"],["104.82.220.159","80"],["216.26.200.58","80"],["109.106.240.250","80"],["54.85.125.155","80"],["204.44.75.149","80"],["121.42.133.51","80"],["66.70.215.118","80"],["217.103.186.88","80"],["82.34.35.195","80"],["51.89.239.32","80"],["192.126.134.90","80"],["156.232.241.167","80"],["207.246.66.153","80"],["194.166.144.92","80"],["32.141.34.14","80"],["185.55.59.251","80"],["3.22.224.98","80"],["198.46.160.193","80"],["147.135.227.32","80"],["173.239.92.39","80"],["3.66.102.253","80"],["84.33.5.176","80"],["43.132.177.219","80"],["114.125.80.158","80"],["185.11.166.71","80"],["88.221.251.37","80"],["84.56.103.172","80"],["184.29.232.99","80"],["167.114.138.179","80"],["52.83.175.131","80"],["18.194.110.174","80"],["104.239.217.238","80"],["35.241.77.177","80"],["160.124.134.149","80"],["174.139.101.71","80"],["184.87.28.249","80"],["23.54.98.8","80"],["134.209.187.33","80"],["152.195.35.103","80"],["39.108.17.113","80"],["23.32.224.86","80"],["23.8.205.33","80"],["54.214.164.232","80"],["134.209.17.86","80"],["52.39.44.216","80"],["103.149.36.14","80"],["46.242.166.45","80"],["46.21.102.125","80"],["13.59.101.212","80"],["72.167.45.84","80"],["34.149.162.89","80"],["173.243.30.54","80"],["104.17.195.155","80"],["154.55.194.6","80"],["14.166.2.98","80"],["104.119.217.191","80"],["154.12.54.214","80"],["34.226.2.223","80"],["92.47.43.236","80"],["23.4.148.83","80"],["156.226.44.43","80"],["14.161.150.95","80"],["54.93.132.6","80"],["39.107.81.30","80"],["184.85.156.58","80"],["206.226.69.36","80"],["23.2.134.117","80"],["129.204.70.131","80"],["34.80.16.30","80"],["13.33.68.178","80"],["94.156.189.73","80"],["104.21.26.38","80"],["158.228.152.22","80"],["42.202.37.198","80"],["173.232.153.10","80"],["107.174.248.84","80"],["161.77.178.102","80"],["104.108.95.181","80"],["54.38.242.185","80"],["46.254.19.126","80"],["45.39.142.72","80"],["185.119.27.15","80"],["154.210.49.27","80"],["194.116.43.34","80"],["121.199.16.177","80"],["71.145.252.84","80"],["192.241.120.20","80"],["185.125.29.240","80"],["145.14.148.52","80"],["104.72.87.126","80"],["104.66.150.87","80"],["73.212.201.92","80"],["67.7.134.100","80"],["194.55.15.204","80"],["184.31.105.80","80"],["82.165.114.159","80"],["23.219.253.116","80"],["185.31.240.146","80"],["66.115.66.69","80"],["217.169.24.123","80"],["154.9.52.198","80"],["199.201.77.38","80"],["96.39.96.226","80"],["3.210.6.112","80"],["200.53.0.33","80"],["107.165.239.250","80"],["192.186.97.70","80"],["42.202.37.142","80"],["203.25.173.47","80"],["23.1.146.99","80"],["116.62.100.80","80"],["174.54.124.252","80"],["18.163.156.129","80"],["153.122.177.139","80"],["209.23.194.122","80"],["54.189.18.64","80"],["94.236.218.138","80"],["120.24.227.211","80"],["203.214.155.174","80"],["104.109.84.204","80"],["45.79.39.63","80"],["117.203.38.206","80"],["54.231.227.121","80"],["188.68.58.39","80"],["20.187.250.20","80"],["185.32.57.211","80"],["154.9.53.128","80"],["188.65.194.224","80"],["154.213.243.55","80"],["23.58.252.252","80"],["34.193.7.22","80"],["184.29.204.74","80"],["42.202.37.165","80"],["147.255.196.205","80"],["87.98.159.25","80"],["84.16.90.186","80"],["132.148.194.154","80"],["23.22.28.34","80"],["104.83.222.174","80"],["23.45.167.3","80"],["106.12.166.247","80"],["117.194.222.207","80"],["194.186.224.211","80"],["2.23.224.92","80"],["199.241.97.10","80"],["187.228.72.26","80"],["155.159.44.234","80"],["122.28.54.49","80"],["183.232.15.115","80"],["96.16.160.120","80"],["120.221.236.233","80"],["3.104.116.133","80"],["52.31.120.134","80"],["104.87.160.140","80"],["185.108.181.82","80"],["194.163.149.135","80"],["207.104.54.251","80"],["45.223.21.177","80"],["23.231.227.24","80"],["155.159.44.240","80"],["52.5.222.81","80"],["219.88.207.61","80"],["34.149.163.14","80"],["23.41.36.209","80"],["23.14.207.111","80"],["8.135.7.63","80"],["3.123.250.154","80"],["54.157.12.15","80"],["120.224.45.232","80"],["156.230.237.196","80"],["116.58.232.151","80"],["119.59.122.122","80"],["45.39.20.10","80"],["18.215.94.97","80"],["213.176.40.10","80"],["195.249.161.84","80"],["156.245.181.162","80"],["93.244.83.127","80"],["156.240.199.56","80"],["38.26.138.138","80"],["146.222.49.227","80"],["104.113.44.48","80"],["18.135.86.76","80"],["164.92.69.161","80"],["66.39.11.67","80"],["34.250.207.18","80"],["184.150.42.12","80"],["15.206.6.95","80"],["119.91.137.197","80"],["23.64.230.232","80"],["59.120.36.38","80"],["151.101.159.120","80"],["23.63.34.209","80"],["101.133.203.216","80"],["54.171.144.132","80"],["185.118.122.110","80"],["165.84.207.192","80"],["8.129.186.37","80"],["103.230.235.4","80"],["2.21.84.156","80"],["211.34.235.49","80"],["54.205.99.6","80"],["107.163.240.185","80"],["54.73.120.113","80"],["103.129.255.233","80"],["121.196.42.185","80"],["122.228.60.157","80"],["46.101.30.19","80"],["154.36.191.218","80"],["89.115.227.41","80"],["185.74.4.102","80"],["54.231.227.125","80"],["96.16.155.7","80"],["185.65.176.22","80"],["54.188.88.92","80"],["185.207.197.206","80"],["213.93.157.54","80"],["46.101.73.246","80"],["104.122.26.168","80"],["120.78.120.27","80"],["136.0.172.52","80"],["23.52.224.192","80"],["146.59.238.71","80"],["54.210.165.71","80"],["185.119.58.137","80"],["34.235.199.226","80"],["170.61.71.73","80"],["178.128.204.43","80"],["121.162.112.48","80"],["104.110.132.24","80"],["104.206.193.163","80"],["107.165.113.196","80"],["62.109.24.232","80"],["104.94.77.87","80"],["202.61.201.160","80"],["104.76.25.81","80"],["154.9.53.62","80"],["154.220.9.77","80"],["132.148.194.230","80"],["104.122.48.126","80"],["23.42.165.45","80"],["152.70.200.16","80"],["85.28.254.81","80"],["154.213.242.223","80"],["66.154.96.215","80"],["195.154.43.221","80"],["206.67.234.54","80"],["121.67.248.162","80"],["34.120.2.195","80"],["184.174.120.254","80"],["210.45.192.243","80"],["23.58.35.169","80"],["148.68.93.92","80"],["52.237.234.155","80"],["212.175.95.150","80"],["65.108.76.202","80"],["86.106.30.249","80"],["184.174.71.141","80"],["83.194.155.87","80"],["52.220.190.160","80"],["47.99.0.226","80"],["185.158.238.12","80"],["45.38.83.213","80"],["35.190.117.182","80"],["156.255.185.62","80"],["185.38.19.123","80"],["20.73.26.205","80"],["120.224.45.229","80"],["39.123.24.147","80"],["23.205.183.21","80"],["149.96.243.169","80"],["185.78.167.107","80"],["185.62.174.186","80"],["83.164.168.204","80"],["174.138.100.178","80"],["63.241.251.167","80"],["104.71.113.13","80"],["154.212.113.186","80"],["3.36.229.17","80"],["18.171.9.9","80"],["23.80.183.12","80"],["186.93.22.157","80"],["101.36.222.242","80"],["118.193.76.9","80"],["185.25.102.157","80"],["107.12.89.11","80"],["198.12.66.203","80"],["121.170.180.58","80"],["147.182.219.141","80"],["59.39.0.43","80"],["95.188.92.157","80"],["67.199.76.190","80"],["184.51.224.209","80"],["201.159.195.238","80"],["211.149.226.156","80"],["152.0.95.176","80"],["115.74.212.39","80"],["36.95.166.10","80"],["34.96.122.224","80"],["14.192.50.19","80"],["156.244.53.5","80"],["23.56.236.118","80"],["23.37.25.228","80"],["23.34.216.111","80"],["194.113.106.104","80"],["206.237.200.193","80"],["58.229.163.171","80"],["54.182.215.213","80"],["23.43.220.173","80"],["217.160.6.192","80"],["196.196.160.51","80"],["160.121.144.25","80"],["185.127.94.112","80"],["35.214.56.104","80"],["156.229.240.250","80"],["165.22.134.1","80"],["120.25.176.4","80"],["23.211.43.42","80"],["34.197.116.234","80"],["75.101.232.135","80"],["184.31.105.66","80"],["78.47.114.19","80"],["23.5.84.86","80"],["104.114.151.100","80"],["23.213.213.39","80"],["23.12.167.159","80"],["162.214.81.170","80"],["99.79.53.25","80"],["67.210.213.126","80"],["195.4.138.60","80"],["79.170.40.4","80"],["67.210.213.126","80"],["195.4.138.60","80"],["185.30.32.88","80"],["104.18.42.220","80"],["54.39.191.172","80"],["164.160.91.36","80"],["195.4.138.60","80"],["217.60.219.112","80"],["47.207.24.139","80"],["173.254.104.160","80"],["157.230.40.100","80"],["34.206.156.234","80"],["176.119.57.203","80"],["144.76.55.212","80"],["45.76.6.155","80"],["146.148.44.68","80"],["104.21.64.234","80"],["18.169.172.106","80"],["166.23.251.82","80"],["185.65.137.205","80"],["185.30.32.249","80"],["64.41.138.195","80"],["185.30.32.88","80"],["104.21.41.37","80"],["178.63.46.161","80"],["164.160.91.36","80"],["212.85.35.206","80"],["195.4.138.60","80"],["99.34.8.170","80"],["216.55.149.9","80"],["185.229.111.112","80"],["162.144.22.117","80"],["200.152.32.127","80"],["167.114.15.225","80"],["99.34.8.170","80"],["109.234.161.178","80"],["217.147.220.11","80"],["113.43.143.180","80"],["104.198.242.183","80"],["185.229.111.112","80"],["132.216.25.80","80"],["144.76.55.212","80"],["195.4.138.60","80"],["35.208.2.21","80"],["104.21.57.20","80"],["202.35.124.136","80"],["54.39.191.172","80"],["152.204.44.22","80"],["164.160.91.36","80"],["160.0.215.199","80"],["5.172.130.135","80"],["67.222.14.133","80"],["64.41.159.183","80"],["18.188.221.17","80"],["173.197.6.90","80"],["140.248.171.27","80"],["37.59.100.149","80"],["156.250.104.54","80"],["52.166.117.154","80"],["109.105.217.130","80"],["34.95.104.29","80"],["185.229.111.112","80"],["43.250.142.4","80"],["163.171.208.211","80"],["156.230.235.126","80"],["3.132.117.172","80"],["62.3.72.99","80"],["146.148.44.68","80"],["62.11.165.230","80"],["142.34.226.87","80"],["18.67.22.31","80"],["171.229.198.205","80"],["192.185.158.219","80"],["54.211.233.126","80"],["20.85.173.144","80"],["34.231.51.99","80"],["23.84.79.200","80"],["24.52.241.42","80"],["175.99.110.121","80"],["38.140.192.108","80"],["63.251.38.141","80"],["38.63.12.70","80"],["178.168.35.157","80"],["90.46.130.89","80"],["173.208.165.2","80"],["61.93.33.175","80"],["149.47.234.141","80"],["69.159.247.103","80"],["188.214.210.51","80"],["156.234.146.193","80"],["52.191.196.228","80"],["20.122.211.110","80"],["20.26.0.144","80"],["166.167.220.121","80"],["23.24.215.153","80"],["188.254.149.195","80"],["89.97.209.133","80"],["20.231.68.187","80"],["54.39.243.203","80"],["191.37.247.250","80"],["188.159.40.203","80"],["217.116.232.214","80"],["37.81.179.40","80"],["66.98.17.100","80"],["86.222.166.165","80"],["85.230.226.14","80"],["201.105.139.26","80"],["200.88.16.11","80"],["36.158.241.105","80"],["93.70.139.12","80"],["189.180.118.44","80"],["14.139.221.204","80"],["66.170.6.172","80"],["195.32.63.58","80"],["45.243.42.220","80"],["76.217.168.239","80"],["176.57.3.61","80"],["185.184.242.205","80"],["197.253.124.140","80"],["2.191.0.69","80"],["54.165.80.73","80"],["171.226.232.246","80"],["54.210.209.32","80"],["81.187.191.185","80"],["201.121.138.101","80"],["200.35.40.193","80"],["24.42.33.134","80"],["195.201.74.90","80"],["109.66.95.106","80"],["188.191.226.50","80"],["166.167.220.135","80"],["174.2.77.10","80"],["13.59.221.208","80"],["187.144.99.171","80"],["102.91.3.130","80"],["123.127.175.7","80"],["24.3.14.87","80"],["75.118.7.76","80"],["161.70.14.132","80"],["170.78.39.105","80"],["185.103.115.16","80"],["157.230.233.182","80"],["3.127.243.41","80"],["195.175.53.246","80"],["208.84.119.10","80"],["82.55.79.236","80"],["188.187.122.104","80"],["78.188.5.144","80"],["157.97.132.132","80"],["52.23.83.150","80"],["152.204.18.21","80"],["104.36.178.210","80"],["52.83.175.176","80"],["163.74.95.20","80"],["128.143.71.163","80"],["46.100.93.190","80"],["70.184.191.82","80"],["37.116.86.3","80"],["177.92.202.204","80"],["206.132.225.26","80"],["85.215.218.198","80"],["104.36.178.140","80"],["54.38.209.129","80"],["46.65.68.144","80"],["189.8.253.125","80"],["103.242.54.7","80"],["187.45.195.13","80"],["86.107.46.162","80"],["34.225.66.191","80"],["128.173.145.207","80"],["119.234.135.80","80"],["83.48.221.174","80"],["15.236.8.236","80"],["189.142.188.106","80"],["185.209.222.246","80"],["83.168.204.196","80"],["213.211.108.238","80"],["201.108.106.82","80"],["190.69.113.95","80"],["66.214.90.102","80"],["95.57.101.27","80"],["20.84.221.65","80"],["188.166.112.177","80"],["178.128.158.143","80"],["217.42.105.204","80"],["20.93.125.140","80"],["20.54.214.130","80"],["5.232.163.130","80"],["38.29.215.9","80"],["5.251.170.219","80"],["77.68.89.155","80"],["217.133.21.253","80"],["208.78.41.242","80"],["185.31.79.221","80"],["13.79.34.186","80"],["189.152.248.168","80"],["187.172.240.96","80"],["178.33.119.214","80"],["181.121.176.227","80"],["72.167.45.102","80"],["91.235.53.87","80"],["45.248.122.35","80"],["181.60.88.108","80"],["82.102.102.52","80"],["216.246.190.70","80"],["178.19.163.74","80"],["173.26.121.184","80"],["185.46.55.162","80"],["148.101.231.197","80"],["82.200.190.154","80"],["178.238.138.16","80"],["37.151.4.235","80"],["94.102.1.29","80"],["174.30.111.98","80"],["189.131.199.135","80"],["13.58.165.142","80"],["178.44.131.115","80"],["185.31.79.230","80"],["37.140.198.14","80"],["24.214.60.149","80"],["103.206.223.226","80"],["118.240.29.89","80"],["1.164.95.200","80"],["212.152.114.92","80"],["89.143.105.70","80"],["91.98.140.200","80"],["69.162.183.76","80"],["46.23.176.219","80"],["189.152.169.0","80"],["190.22.75.89","80"],["184.147.41.49","80"],["213.137.39.109","80"],["82.62.93.170","80"],["52.57.175.92","80"],["20.201.76.251","80"],["145.82.36.73","80"],["186.145.234.141","80"],["184.73.116.237","80"],["148.72.238.179","80"],["20.113.55.66","80"],["54.147.70.191","80"],["23.45.26.44","80"],["154.12.54.210","80"],["154.31.55.177","80"],["23.44.37.242","80"],["70.132.12.100","80"],["43.128.240.45","80"],["185.170.62.236","80"],["95.61.38.182","80"],["23.62.166.55","80"],["80.74.140.26","80"],["156.242.182.39","80"],["84.16.90.174","80"],["14.163.34.97","80"],["79.215.216.18","80"],["42.119.240.237","80"],["58.230.231.147","80"],["83.206.249.4","80"],["3.139.2.67","80"],["160.72.128.118","80"],["212.236.186.157","80"],["157.90.79.124","80"],["2.194.194.84","80"],["66.220.9.181","80"],["67.222.99.50","80"],["92.79.99.146","80"],["176.88.16.212","80"],["59.95.161.154","80"],["54.215.39.230","80"],["66.38.74.192","80"],["168.138.144.133","80"],["213.42.52.177","80"],["120.101.65.227","80"],["200.26.189.191","80"],["121.176.210.191","80"],["208.86.184.65","80"],["37.187.16.237","80"],["121.43.183.210","80"],["52.20.99.129","80"],["189.180.182.252","80"],["202.218.21.20","80"],["87.120.101.52","80"],["167.56.224.223","80"],["192.99.168.72","80"],["86.163.121.198","80"],["108.188.37.23","80"],["54.85.115.175","80"],["52.210.221.134","80"],["117.203.160.139","80"],["5.76.97.222","80"],["188.28.178.197","80"],["186.0.163.179","80"],["54.205.127.117","80"],["51.75.130.104","80"],["147.182.220.20","80"],["81.214.111.181","80"],["116.118.2.161","80"],["178.32.84.32","80"],["5.76.98.40","80"],["94.6.67.37","80"],["179.37.16.207","80"],["41.32.247.94","80"],["176.178.149.38","80"],["47.96.92.241","80"],["156.193.49.87","80"],["151.231.100.184","80"],["185.52.251.6","80"],["192.249.121.56","80"],["133.110.151.223","80"],["147.139.186.230","80"],["134.236.85.53","80"],["13.124.252.18","80"],["185.95.15.136","80"],["90.108.223.79","80"],["54.179.10.32","80"],["67.225.153.46","80"],["190.213.236.65","80"],["146.148.132.231","80"],["82.98.164.30","80"],["61.228.145.20","80"],["104.18.128.156","80"],["165.73.234.119","80"],["213.155.117.18","80"],["3.228.207.74","80"],["23.75.235.30","80"],["18.215.211.232","80"],["116.105.55.153","80"],["111.231.210.130","80"],["185.4.67.45","80"],["65.9.92.227","80"],["20.193.128.78","80"],["184.168.117.157","80"],["173.232.152.217","80"],["151.231.100.204","80"],["188.147.167.151","80"],["172.121.89.140","80"],["189.128.157.30","80"],["185.95.14.183","80"],["173.249.32.203","80"],["213.32.111.138","80"],["47.52.172.217","80"],["52.54.195.249","80"],["156.245.245.36","80"],["185.246.66.127","80"],["54.206.63.230","80"],["94.46.14.33","80"],["212.129.61.66","80"],["94.103.232.179","80"],["94.122.174.117","80"],["34.89.183.111","80"],["192.185.29.225","80"],["52.45.29.80","80"],["52.221.98.238","80"],["148.69.248.154","80"],["59.110.162.85","80"],["34.111.128.166","80"],["45.177.14.149","80"],["164.90.235.47","80"],["54.93.160.111","80"],["106.52.240.99","80"],["38.131.158.243","80"],["185.237.223.181","80"],["3.227.78.92","80"],["98.129.164.112","80"],["104.120.154.82","80"],["121.204.148.15","80"],["47.243.143.20","80"],["195.234.0.39","80"],["156.244.245.177","80"],["207.226.152.43","80"],["121.40.182.242","80"],["95.100.185.22","80"],["14.180.43.212","80"],["201.236.175.10","80"],["147.182.241.159","80"],["200.7.170.203","80"],["37.151.75.96","80"],["104.131.31.95","80"],["190.5.234.122","80"],["13.226.233.235","80"],["52.0.8.48","80"],["151.232.229.133","80"],["94.253.64.9","80"],["66.35.111.125","80"],["119.218.119.173","80"],["189.128.200.109","80"],["51.52.107.75","80"],["185.37.226.208","80"],["46.105.212.236","80"],["23.200.116.77","80"],["154.12.54.157","80"],["46.30.62.218","80"],["184.29.232.66","80"],["154.9.53.22","80"],["185.221.197.12","80"],["188.114.89.118","80"],["3.141.67.72","80"],["70.68.144.5","80"],["41.65.4.167","80"],["121.42.5.131","80"],["154.19.97.239","80"],["52.69.41.87","80"],["183.105.224.208","80"],["99.61.21.142","80"],["37.49.103.86","80"],["77.93.229.187","80"],["72.167.45.110","80"],["15.160.74.29","80"],["66.214.42.218","80"],["154.47.69.155","80"],["51.195.129.9","80"],["31.210.209.131","80"],["148.6.2.125","80"],["34.239.137.164","80"],["154.36.249.2","80"],["117.200.225.131","80"],["185.34.32.67","80"],["173.232.152.208","80"],["116.202.23.38","80"],["23.40.219.52","80"],["104.248.222.156","80"],["52.4.138.5","80"],["37.120.191.229","80"],["154.88.86.72","80"],["94.237.67.89","80"],["23.212.122.8","80"],["41.40.215.196","80"],["42.117.80.170","80"],["35.212.197.78","80"],["92.36.208.172","80"],["42.117.80.188","80"],["51.254.104.187","80"],["13.230.251.182","80"],["23.10.102.90","80"],["222.114.122.29","80"],["154.204.112.219","80"],["136.226.51.78","80"],["139.162.186.89","80"],["166.1.57.179","80"],["23.35.33.163","80"],["24.95.58.92","80"],["147.92.41.19","80"],["23.58.109.19","80"],["188.132.201.28","80"],["38.22.58.26","80"],["14.241.119.47","80"],["181.41.228.12","80"],["120.77.235.57","80"],["66.117.15.55","80"],["163.197.253.3","80"],["154.209.112.142","80"],["184.84.248.16","80"],["157.88.14.181","80"],["52.1.136.172","80"],["23.13.11.146","80"],["77.21.131.101","80"],["196.61.36.198","80"],["41.215.92.43","80"],["3.120.249.68","80"],["212.83.157.39","80"],["94.16.7.82","80"],["149.57.168.189","80"],["68.177.189.219","80"],["188.166.215.180","80"],["185.170.63.36","80"],["51.159.27.177","80"],["184.168.114.69","80"],["201.160.32.23","80"],["174.99.146.92","80"],["42.117.135.224","80"],["68.67.77.93","80"],["189.154.41.141","80"],["14.237.128.220","80"],["34.143.44.140","80"],["66.190.144.104","80"],["66.119.213.144","80"],["45.238.37.5","80"],["103.138.4.214","80"],["20.204.68.244","80"],["60.49.63.14","80"],["157.230.16.237","80"],["15.222.228.216","80"],["162.13.188.4","80"],["13.231.68.232","80"],["68.233.77.119","80"],["3.25.33.224","80"],["52.76.183.233","80"],["198.57.214.115","80"],["18.66.145.157","80"],["3.141.94.202","80"],["213.45.178.143","80"],["106.12.166.231","80"],["95.179.57.199","80"],["3.120.56.203","80"],["95.128.1.22","80"],["156.242.182.17","80"],["34.210.251.192","80"],["37.97.245.39","80"],["175.239.119.162","80"],["186.231.220.135","80"],["185.52.251.25","80"],["20.212.81.23","80"],["85.192.44.70","80"],["206.189.27.249","80"],["193.120.233.131","80"],["154.19.114.189","80"],["184.25.74.10","80"],["45.207.92.228","80"],["37.97.245.59","80"],["84.33.5.79","80"],["194.50.250.7","80"],["102.65.251.229","80"],["23.53.41.247","80"],["219.76.203.140","80"],["158.51.9.114","80"],["87.98.159.16","80"],["184.87.32.175","80"],["23.23.82.215","80"]]}`))
					}

					return
				}
			case "port=5354":
				switch r.FormValue("full") {
				case "false":
					w.Write([]byte(`{"error":false,"size":12345678,"page":1,"mode":"extended","query":"port=\"5453\"","results":[["94.130.128.248","5453"]]}`))
				case "true":
					w.Write([]byte(`{"error":false,"size":12345678,"page":1,"mode":"extended","query":"port=\"5453\"","results":[["94.130.128.248","5453"], ["94.130.128.124","5453"]]}`))
				}
			}
		case "/api/v1/search/stats":
			account := checkAccount(r.FormValue("email"), r.FormValue("key"))
			if account == nil {
				w.Write([]byte(`{"error":true,"errmsg":"[-700] Account Invalid"}`))
				return
			}

			// 构造错误
			if r.FormValue("size") == "0" {
				w.Write([]byte(`{"error":true,"errmsg":"[51] The Size value ` + "`" + `0` + "`" + ` must be between 5 and 10000"}`))
				return
			}

			switch r.FormValue("fields") {
			case "title":
				w.Write([]byte(`{"error":false,"distinct":{"ip":144828930,"title":33994578},"aggs":{"countries":[],"title":[{"code":"cG9ydD0iODAiICYmIHRpdGxlPT0iMzAxIE1vdmVkIFBlcm1hbmVudGx5Ig==","count":25983408,"name":"301 Moved Permanently"},{"code":"cG9ydD0iODAiICYmIHRpdGxlPT0iNTAyIEJhZCBHYXRld2F5Ig==","count":25233607,"name":"502 Bad Gateway"},{"code":"cG9ydD0iODAiICYmIHRpdGxlPT0iSW52YWxpZCBVUkwi","count":11063458,"name":"Invalid URL"},{"code":"cG9ydD0iODAiICYmIHRpdGxlPT0iNDA0IE5vdCBGb3VuZCI=","count":6826085,"name":"404 Not Found"},{"code":"cG9ydD0iODAiICYmIHRpdGxlPT0iNDAzIEZvcmJpZGRlbiI=","count":6401684,"name":"403 Forbidden"}]},"lastupdatetime":"2022-05-18 20:00:00"}`))
				return
			case "title,country", "":
				w.Write([]byte(`{"error":false,"distinct":{"ip":144828930,"title":33994578},"aggs":{"countries":[{"code":"cG9ydD0iODAiICYmIGNvdW50cnk9IlVTIg==","count":154746752,"name":"United States of America","name_code":"US","regions":[{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iIg==","count":67483283,"name":"Unknown"},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iQ2FsaWZvcm5pYSI=","count":20006414,"name":"California"},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iVmlyZ2luaWEi","count":16777201,"name":"Virginia"},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iSWxsaW5vaXMi","count":14185380,"name":"Illinois"},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iT3JlZ29uIg==","count":5078669,"name":"Oregon"}]},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iSEsi","count":48914928,"name":"Hong Kong Special Administrative Region","name_code":"HK","regions":[{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iIg==","count":34727205,"name":"Unknown"},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iQ2VudHJhbCBhbmQgV2VzdGVybiBEaXN0cmljdCI=","count":12571956,"name":"Central and Western District"},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iU2FpIEt1bmcgRGlzdHJpY3Qi","count":931412,"name":"Sai Kung District"},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iS293bG9vbiBDaXR5Ig==","count":231094,"name":"Kowloon City"},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iU2hhIFRpbiI=","count":161509,"name":"Sha Tin"}]},{"code":"cG9ydD0iODAiICYmIGNvdW50cnk9IkNOIg==","count":36835641,"name":"China","name_code":"CN","regions":[{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iIg==","count":17640379,"name":"Unknown"},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iQmVpamluZyI=","count":4846156,"name":"Beijing"},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iWmhlamlhbmci","count":4422439,"name":"Zhejiang"},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iR3Vhbmdkb25nIg==","count":3171394,"name":"Guangdong"},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iU2hhbmdoYWki","count":2186671,"name":"Shanghai"}]},{"code":"cG9ydD0iODAiICYmIGNvdW50cnk9IkRFIg==","count":26973035,"name":"Germany","name_code":"DE","regions":[{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iIg==","count":6921394,"name":"Unknown"},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iSGVzc2Ui","count":5888986,"name":"Hesse"},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iTm9ydGggUmhpbmUtV2VzdHBoYWxpYSI=","count":3192657,"name":"North Rhine-Westphalia"},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iQmF2YXJpYSI=","count":3176589,"name":"Bavaria"},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iQmFkZW4tV8O8cnR0ZW1iZXJnIg==","count":1633263,"name":"Baden-Württemberg"}]},{"code":"cG9ydD0iODAiICYmIGNvdW50cnk9IkpQIg==","count":12824018,"name":"Japan","name_code":"JP","regions":[{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iVG9reW8i","count":5406913,"name":"Tokyo"},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iIg==","count":3982455,"name":"Unknown"},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0ixYxzYWthIg==","count":959777,"name":"Ōsaka"},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iS2FuYWdhd2Ei","count":303862,"name":"Kanagawa"},{"code":"cG9ydD0iODAiICYmIHJlZ2lvbj0iU2FpdGFtYSI=","count":206553,"name":"Saitama"}]}],"title":[{"code":"cG9ydD0iODAiICYmIHRpdGxlPT0iMzAxIE1vdmVkIFBlcm1hbmVudGx5Ig==","count":25983454,"name":"301 Moved Permanently"},{"code":"cG9ydD0iODAiICYmIHRpdGxlPT0iNTAyIEJhZCBHYXRld2F5Ig==","count":25233610,"name":"502 Bad Gateway"},{"code":"cG9ydD0iODAiICYmIHRpdGxlPT0iSW52YWxpZCBVUkwi","count":11063458,"name":"Invalid URL"},{"code":"cG9ydD0iODAiICYmIHRpdGxlPT0iNDA0IE5vdCBGb3VuZCI=","count":6826166,"name":"404 Not Found"},{"code":"cG9ydD0iODAiICYmIHRpdGxlPT0iNDAzIEZvcmJpZGRlbiI=","count":6401700,"name":"403 Forbidden"}]},"lastupdatetime":"2022-05-18 21:00:00"}`))
				return
			}
		case "/api/v1/host/1.1.1.1", "/api/v1/host/fofa.info":
			w.Write([]byte(`{
  "error": false,
  "host": "1.1.1.1",
  "ip": "1.1.1.1",
  "asn": 6805,
  "org": "Telefonica Germany",
  "country_name": "Germany",
  "country_code": "DE",
  "protocol": [
    "sip",
    "http",
    "https"
  ],
  "port": [
    5060,
    8089,
    7170,
    443
  ],
  "category": [
    "CMS"
  ],
  "product": [
    "Synology-WebStation"
  ],
  "update_time": "2022-05-24 12:00:00"
}`))
			return
		case "/api/v1/search/next":
			next := "1"
			if id := r.URL.Query().Get("next"); id != "" {
				next = id
			}
			i, _ := strconv.Atoi(next)
			var results [][]string
			for j := 0; j < 10; j++ {
				data := []string{
					fmt.Sprintf("%d.%d.%d.%d", i, i, i, i),
					strconv.Itoa(80 + i + j),
				}
				if r.URL.Query().Get("fields") == "host,ip,port" {
					data = append([]string{fmt.Sprintf("http://%d.%d.%d.%d", i, i, i, i)}, data...)
				}
				results = append(results, data)
			}

			ret := map[string]interface{}{
				"error":   false,
				"size":    100,
				"mode":    "extended",
				"query":   "title=\"百度\"", // 查询语句
				"results": results,
			}
			if i == 10 {
				ret["next"] = ""
			} else {
				ret["next"] = strconv.Itoa(i + 1) // 下一次查询的id，翻页需要带上
			}
			b, _ := json.Marshal(ret)
			w.Write(b)
			return
		}
	}
)

func checkAccount(email, key string) *accountInfo {
	for _, validAccount := range validAccounts {
		if email == validAccount.Email &&
			key == validAccount.Key {
			return &validAccount
		}
	}
	return nil
}

type testHook struct {
	f func(e *logrus.Entry)
}

func (th *testHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (th *testHook) Fire(e *logrus.Entry) error {
	th.f(e)
	return nil
}

func TestNewClient(t *testing.T) {
	defer func() {
		os.Unsetenv("FOFA_CLIENT_URL")
		os.Unsetenv("FOFA_SERVER")
		os.Unsetenv("FOFA_EMAIL")
		os.Unsetenv("FOFA_KEY")
	}()

	var cli *Client
	var err error

	// 异常的环境变量
	os.Setenv("FOFA_CLIENT_URL", "\x7f")
	cli, err = NewClient(WithURL(""))
	assert.Error(t, err)
	assert.Nil(t, cli)

	// url异常
	os.Unsetenv("FOFA_CLIENT_URL")
	cli, err = NewClient(WithURL("\x7F"))
	assert.Error(t, err)
	assert.Nil(t, cli)

	ts := httptest.NewServer(http.HandlerFunc(queryHander))
	defer ts.Close()

	// 都正常有环境变量没有参数，取环境变量
	account := validAccounts[1]
	fofaURL := ts.URL + "/?email=" + account.Email + "&key=" + account.Key + "&version=v1"
	os.Setenv("FOFA_CLIENT_URL", fofaURL)
	cli, err = NewClient(WithURL(""))
	assert.Nil(t, err)
	assert.Equal(t, fofaURL, cli.URL())

	// 都正常有参数，以参数为主
	account = validAccounts[2]
	fofaURLNew := ts.URL + "/?email=" + account.Email + "&key=" + account.Key + "&version=v1"
	os.Setenv("FOFA_CLIENT_URL", fofaURL)
	cli, err = NewClient(WithURL(fofaURLNew))
	assert.Nil(t, err)
	assert.Equal(t, fofaURLNew, cli.URL())

	// 日志处理器替换
	var logs []string
	logger := logrus.New()
	logger.AddHook(&testHook{f: func(e *logrus.Entry) {
		logs = append(logs, e.Message)
	}})
	cli, err = NewClient(WithURL(ts.URL+"/?email="+account.Email+"&key=1&version=v1"), WithLogger(logger))
	assert.True(t, len(logs) > 0)

	// 账号调试信息
	account = validAccounts[2]
	invalidUrl := "https://" + strings.Split(ts.URL, "://")[1] + "/?email=" + account.Email + "&key=" + account.Key + "&version=v1"
	cli, err = NewClient(WithURL(invalidUrl), WithAccountDebug(true))
	var u string
	u, err = url.QueryUnescape(err.Error())
	assert.Nil(t, err)
	assert.Contains(t, u, account.Email)
	cli, err = NewClient(WithURL(invalidUrl))
	u, err = url.QueryUnescape(err.Error())
	assert.Nil(t, err)
	assert.NotContains(t, u, account.Email)
}
