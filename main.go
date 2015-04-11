package main

import (
	"bytes"
	"code.google.com/p/go.net/websocket"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir("./views")))
	http.HandleFunc("/userLogin", userLoginHandler) //判断用户登陆并跳转
	http.HandleFunc("/chatRoom", chatRoomHandler)   //答应chat页面
	http.HandleFunc("/register", registerHandler)   //判断能否注册并跳转
	http.Handle("/chat", websocket.Handler(chatHandler))
	http.HandleFunc("/userManagerPrint", userManagerPrintHandler) //打印人员管理界面
	http.HandleFunc("/userMgr", userMgrHandler)                   //传送人员管理列表
	http.HandleFunc("/operUser", operUserHandler)                 //用户管理操作
	http.HandleFunc("/sysMgr", sysMgrHandler)                     //系统管理
	http.HandleFunc("/quit", quitHandler)
	http.HandleFunc("/help", helpHandler)
	http.Handle("/kefu", websocket.Handler(kefuHandler))
	http.HandleFunc("/kefuPage", kefuPageHandler)

	err := http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
func kefuPageHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("User")
	if err == nil {
		//fmt.Println(cookie.Value)
		files, _ := ioutil.ReadFile("./views/kefuAdmin.html")
		sour := []byte("{{%}}")
		replace := []byte(cookie.Value)
		respon := bytes.Replace(files, sour, replace, -1)
		w.Write(respon)
	} else {
		w.Write([]byte("Worry"))
	}
}

type kefu struct {
	//ws   *websocket.Conn
	name string //客服名
	busy int
	ip   string //与之聊天的人的IP
}

var (
	kefuList = make(map[*websocket.Conn]kefu)
	youke    = make(map[string]*websocket.Conn)
)

func kefuHandler(ws *websocket.Conn) {
	defer ws.Close()

HERE:
	var withWho *websocket.Conn //只对游客有用
	clnIP := ws.Request().RemoteAddr
	user := ws.Request().URL.Query()["user"][0]
	fmt.Println("1")

	db, _ := sql.Open("mysql", "root:root@/chatroom?charset=utf8")
	rows, _ := db.Query("select server from fuwu where server=?", user)

	server := ""
	for rows.Next() {
		rows.Scan(&server)
	}

	if "youke" == user {
		zhaodao := 0
		youke[clnIP] = ws
		for {
			for cln := range kefuList {
				if "" == kefuList[cln].ip {
					zhaodao = 1
					withWho = cln
					kefuList[cln] = kefu{kefuList[cln].name, 1, clnIP}
					websocket.Message.Send(ws, "找到名为"+kefuList[cln].name+"的客服")
					break
				}
			}
			if 1 == zhaodao {
				break
			} else {
				websocket.Message.Send(ws, "暂时找不到客服")
				time.Sleep(time.Second * 3)
			}
		}
	} else {
		kefuList[ws] = kefu{user, 0, ""}
	}

	if "" != server {

		for "" == kefuList[ws].ip { //等待游客访问
			for {
				var msg string
				toWs := youke[kefuList[ws].ip]
				err := websocket.Message.Receive(ws, &msg)
				if err != nil {
					delete(kefuList, ws)
					fmt.Println("一个客服掉线了")
					return
				}
				websocket.Message.Send(toWs, kefuList[ws].ip+": "+msg)
			}
		}

	} else {

		for {
			var msg string
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				delete(youke, clnIP)
				for aa := range kefuList {
					if clnIP == kefuList[aa].ip {
						kefuList[aa] = kefu{kefuList[aa].name, 1, ""}
					}
				}
				return
			}
			websocket.Message.Send(ws, clnIP+": "+msg)
			websocket.Message.Send(withWho, clnIP+": "+msg)
			_, ok := kefuList[withWho]
			if !ok {
				websocket.Message.Send(ws, "该客服掉线了，正在为你重新查找")
				goto HERE
			}
		}

	}

}

func helpHandler(w http.ResponseWriter, r *http.Request) {
	userName := r.PostFormValue("username")
	topic := r.PostFormValue("topic")
	contain := r.PostFormValue("contain")

	bodyContain := "user:" + userName + "\r\n" + contain
	user := "zfliu_10@126.com"
	password := "@szlhstlslzf525@"
	host := "smtp.126.com:25"
	to := "zfliu_10@126.com"
	subject := topic
	body := `
       <html>
       <body>
       <h3>
       ` + bodyContain + `
       </h3>
       </body>
       </html>
       `
	fmt.Println("send email")

	hp := strings.Split(host, ":")
	auth := smtp.PlainAuth("", user, password, hp[0])
	var content_type string

	content_type = "Content-Type: text/" + "html" + "; charset=UTF-8"

	msg := []byte("To: " + to + "\r\nFrom: " + user + "<" + user +
		">\r\nSubject: " + subject + "\r\n" + content_type + "\r\n\r\n" + body)
	send_to := strings.Split(to, ";")

	err := smtp.SendMail(host, auth, user, send_to, msg)

	if err != nil {
		fmt.Println("Send mail error!")
		fmt.Println(err)
	} else {
		fmt.Println("Send mail success!")
		respon :=
			`<html><h1>发送成功，我们会尽快帮你处理</h1><h1>2秒后将跳转到主页面</h1>
		<script language=\"javascript\"> 
		function go( ) { 
			window.location="index.html"; 
		} 
		window.setTimeout("go()",1000);//2秒后执行函数go 
		</script>
		</html>`
		w.Write([]byte(respon))
		//http.Redirect(w, r, "index.html", http.StatusFound)
	}

}

func sysMgrHandler(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("User")
	userName := cookie.Value

	//fmt.Println(userName)
	db, _ := sql.Open("mysql", "root:root@/chatroom?charset=utf8")
	rows, _ := db.Query("select uid,admin from users where username=?", userName)
	defer db.Close()
	id := 0
	admin := 0
	for rows.Next() {
		rows.Scan(&id, &admin)
	}
	if id > 0 {
		if 1 == admin {
			files, _ := ioutil.ReadFile("./views/sysManage.html")
			sour := []byte("{{%}}")
			replace := []byte(cookie.Value)
			respon := bytes.Replace(files, sour, replace, -1)
			w.Write(respon)
		} else {
			w.Write([]byte("非法操作"))
		}
	} else {
		w.Write([]byte("404"))
	}
}

func quitHandler(w http.ResponseWriter, r *http.Request) {

	respon, _ := ioutil.ReadFile("./views/index.html")
	cookie := http.Cookie{
		Name:   "User",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, &cookie)
	w.Write(respon)
}

func operUserHandler(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("User")
	what := r.URL.Query()["type"][0]
	//fmt.Println(cookie.Value)
	//fmt.Println(what)
	switch what {
	case "add":
		userName := r.URL.Query()["name"][0]
		passWord := r.URL.Query()["pw"][0]
		//fmt.Println(userName + passWord)
		db, _ := sql.Open("mysql", "root:root@/chatroom?charset=utf8")
		rows, _ := db.Query("select uid from users where username=?", userName)
		id := 0
		for rows.Next() {
			rows.Scan(&id)
		}
		if id > 0 {
			w.Write([]byte("用户名已存在"))
		} else {
			db.Exec("insert into users(username,password) values(?,?)", userName, passWord)
			w.Write([]byte("添加成功"))
		}
		db.Close()
	case "del":
		userName := r.URL.Query()["name"][0]
		//fmt.Println(userName)
		db, _ := sql.Open("mysql", "root:root@/chatroom?charset=utf8")
		if userName == cookie.Value {
			w.Write([]byte("不能删除自己"))
		} else {
			db.Exec("delete from users where username=?", userName)
			db.Close()
			w.Write([]byte("删除成功"))
		}
		db.Close()
	case "upd":
		//fmt.Println(r.URL)
		userName := r.URL.Query()["name"][0]
		passWord := r.URL.Query()["password"][0]
		var canlogin string
		canlogin = r.URL.Query()["canlogin"][0]
		//fmt.Println(canlogin)
		db, _ := sql.Open("mysql", "root:root@/chatroom?charset=utf8")
		db.Exec("update users set password=?,canlogin=? where username=?", passWord, canlogin, userName)
		//fmt.Print(userName + passWord)
		db.Close()
		w.Write([]byte("更新成功"))
	default:
		w.Write([]byte("发生了一些为止的错误"))
	}
}

func userManagerPrintHandler(w http.ResponseWriter, r *http.Request) {

	cookie, _ := r.Cookie("User")
	userName := cookie.Value

	//fmt.Println(userName)
	db, _ := sql.Open("mysql", "root:root@/chatroom?charset=utf8")
	rows, _ := db.Query("select uid,admin from users where username=?", userName)
	defer db.Close()
	id := 0
	admin := 0
	for rows.Next() {
		rows.Scan(&id, &admin)
	}
	if id > 0 {
		if 1 == admin {
			files, _ := ioutil.ReadFile("./views/userManage.html")
			sour := []byte("{{%}}")
			replace := []byte(cookie.Value)
			respon := bytes.Replace(files, sour, replace, -1)
			w.Write(respon)
		} else {
			w.Write([]byte("你没有权限"))
		}
	} else {
		w.Write([]byte("非法操作"))
	}
}

func userLoginHandler(w http.ResponseWriter, r *http.Request) {

	userName := r.PostFormValue("Username")
	passWord := r.PostFormValue("Password")

	db, _ := sql.Open("mysql", "root:root@/chatroom?charset=utf8")
	stmt, _ := db.Prepare(("select uid,canlogin from users where username=? and password=?"))
	rows, _ := stmt.Query(userName, passWord)

	user_id := 0
	var canlogin string
	for rows.Next() {
		rows.Scan(&user_id, &canlogin)
	}

	if user_id > 0 {
		if "1" == canlogin {
			t := time.Now().Format("2006-01-02")
			db.Exec("update users set time=? where uid=?", t, user_id)
			cookie := http.Cookie{
				Name:   "User",
				Value:  userName,
				Path:   "/",
				MaxAge: 86400,
			}
			http.SetCookie(w, &cookie)
			r.AddCookie(&cookie)
			//fmt.Println(cookie)
			http.Redirect(w, r, "/chatRoom", http.StatusFound)
		} else {
			respon := `
				你被管理员禁止登陆
			`
			w.Write([]byte(respon))
		}
	} else {
		page := "登陆失败"
		w.Write([]byte(page))
	}
	db.Close()
}

func userMgrHandler(w http.ResponseWriter, r *http.Request) {

	cookie, _ := r.Cookie("User")
	userName := cookie.Value

	//fmt.Println(userName)
	db, _ := sql.Open("mysql", "root:root@/chatroom?charset=utf8")
	rows, _ := db.Query("select uid,admin from users where username=?", userName)
	defer db.Close()
	id := 0
	admin := 0
	for rows.Next() {
		rows.Scan(&id, &admin)
	}

	//fmt.Println(id)
	respon := "[" //发送到客户端的东西
	//var JSONrespon UserInfo
	if id > 0 {
		if 1 == admin {
			rows, _ = db.Query("select username,password,canlogin,time from users")
			for rows.Next() {
				var un, ps, cl, t string
				rows.Scan(&un, &ps, &cl, &t)
				respon += "{"
				respon += "\"name\":\"" + un + "\","
				respon += "\"password\":\"" + ps + "\","
				respon += "\"canlogin\":\"" + cl + "\","
				respon += "\"time\":\"" + t + "\""
				respon += "},"
			}
			respon += "]"
			respon = strings.Replace(respon, ",]", "]", 1)
			//respon := `[{"id":"1","name" : "aaa","password" : "111","canlogin" : "1"},
			//			{"id":"2","name" : "bbb","password" : "111","canlogin" : "1"},
			//			{"id":"3","name" : "ccc","password" : "111","canlogin" : "1"}]`
			//fmt.Println(respon)
			w.Write([]byte(respon))
		} else {
			w.Write([]byte("非法操作"))
		}
	} else {
		w.Write([]byte("404"))
	}
}

func chatRoomHandler(w http.ResponseWriter, r *http.Request) {

	cookie, err := r.Cookie("User")
	if err == nil {
		//fmt.Println(cookie.Value)
		files, _ := ioutil.ReadFile("./views/chat.html")
		sour := []byte("{{%}}")
		replace := []byte(cookie.Value)
		respon := bytes.Replace(files, sour, replace, -1)
		w.Write(respon)
	} else {
		w.Write([]byte("Worry"))
	}
}

func registerHandler(w http.ResponseWriter, r *http.Request) {

	//fmt.Println(r.Form)
	//fmt.Println(r.URL)
	userName := r.URL.Query()["name"][0]
	passWord := r.URL.Query()["password"][0]
	again := r.URL.Query()["again"][0]
	//fmt.Println(userName + " 111")

	if passWord != again {
		respon := "两次输入密码不一致，请重新输入"
		w.Write([]byte(respon))
		return
	}

	db, _ := sql.Open("mysql", "root:root@/chatroom?charset=utf8")
	rows, _ := db.Query("select uid from users where username=?", userName)

	id := 0
	for rows.Next() {
		rows.Scan(&id)
	}
	//fmt.Println(id)
	if id > 0 {
		//respBuf, _ := ioutil.ReadFile("./views/register2.html")
		respBuf := "用户已存在"
		w.Write([]byte(respBuf))
	} else {
		db.Exec("insert into users(username,password) values(?,?)", userName, passWord)
		//cookie := http.Cookie{
		//	Name:   "User",
		//	Value:  userName,
		//	Path:   "/",
		//	MaxAge: 86400,
		//}
		//http.SetCookie(w, &cookie)
		//http.Redirect(w, r, "/chatRoom", http.StatusFound)
		db.Close()
		respon := "注册成功"
		w.Write([]byte(respon))
	}
}

type info struct {
	ws      *websocket.Conn
	ip      string
	relogin int
}

var (
	connClns = make(map[string]info)
)

func chatHandler(ws *websocket.Conn) {
	defer ws.Close()

	clnIP := ws.Request().RemoteAddr
	user := ws.Request().URL.Query().Get("user")
	//fmt.Println(clnIP)
	//------------------
	value, exists := connClns[user]
	if exists {
		fmt.Println("下线前ip：" + value.ip)
		websocket.Message.Send(connClns[user].ws, "你的帐号在别的地方登陆,你被迫下线")
		connClns[user].ws.Close()
		//delete(connClns, user)
		connClns[user] = info{ws, clnIP, 1}
		fmt.Println("顶号后ip：" + connClns[user].ip)
	} else {
		connClns[user] = info{ws, clnIP, 0}
		//fmt.Println(clnIP)
	}
	//------------------
	//connClns[user] = info{ws, clnIP}

	db, _ := sql.Open("mysql", "root:root@/chatroom?charset=utf8")
	rows, _ := db.Query("select friend from friends where username=?", user)
	online := "/online"
	offline := "/offline"
	friends := "friendsList"
	var scan string
	var toAll []string
	for rows.Next() {
		rows.Scan(&scan)
		va, ex := connClns[scan]
		if ex {
			online = online + "/" + scan
			fmt.Println(va.ip)
		} else {
			offline = offline + "/" + scan
		}
		//friends = friends + "/" + scan
		toAll = append(toAll, scan)
	}
	friends += online + offline
	//fmt.Println(user + "的好友" + friends)
	websocket.Message.Send(ws, friends) //对自己加载好友列表

	fmt.Println(toAll)
	for i := 0; i < len(toAll); i++ { //告诉其它人应该刷新好友列表
		_, ex := connClns[toAll[i]]
		fmt.Print(toAll[i])
		if ex {
			websocket.Message.Send(connClns[toAll[i]].ws, "shouldRefresh")
			fmt.Println("通知" + toAll[i] + "shouldRefresh")
		} else {
			fmt.Println(ex)
		}
	}
	//----------------离线通知----------------
	{
		rows, _ = db.Query("select wfrom,wto,msg from offlinemsg where wto=?", user)
		var wfrom string
		var wto string
		var wmsg string
		for rows.Next() {
			rows.Scan(&wfrom, &wto, &wmsg)
			websocket.Message.Send(ws, wfrom+"在你离线时对你说："+wmsg)
		}
		db.Exec("delete from offlinemsg where wto=?", user)
		db.Close()
	}
	//---------------------------------------

	for {

		var sour string
		var hand []string
		err := websocket.Message.Receive(ws, &sour)
		fmt.Println("收到：" + user + "," + sour)
		if err != nil {

			for i := 0; i < len(toAll); i++ { //告诉其它人应该刷新好友列表
				_, ex := connClns[toAll[i]]
				if ex {
					websocket.Message.Send(connClns[toAll[i]].ws, "shouldRefresh")
				}
			}
			if 1 == connClns[user].relogin {
				connClns[user] = info{connClns[user].ws, connClns[user].ip, 0}
			} else {
				delete(connClns, user)
			}
			return
		}
		hand = strings.Split(sour, ",")
		fmt.Println(hand)

		switch hand[0] {
		case "shouldRefresh":
			{
				db, _ := sql.Open("mysql", "root:root@/chatroom?charset=utf8")
				rows, _ := db.Query("select friend from friends where username=?", user)
				online := "/online"
				offline := "/offline"
				friends := "friendsList"
				scan = ""
				for rows.Next() {
					rows.Scan(&scan)
					va, ex := connClns[scan]
					if ex {
						online = online + "/" + scan
						fmt.Println(va.ip)
					} else {
						offline = offline + "/" + scan
					}
					//friends = friends + "/" + scan
				}
				friends += online + offline
				fmt.Println(user + "的好友" + friends)
				db.Close()
				websocket.Message.Send(ws, friends) //对自己加载好友列表
			}

		//添加好友
		case "addFriend":
			{
				username := hand[1]
				db, _ := sql.Open("mysql", "root:root@/chatroom?charset=utf8")
				rows, _ := db.Query("select uid from users where username = ?", username)

				user_id := 0
				for rows.Next() {
					rows.Scan(&user_id)
				}
				rows.Close()

				rows, _ = db.Query("select username,friend from friends where username=? and friend=?", user, username)
				haveFriend := ""
				haveName := ""
				for rows.Next() {
					rows.Scan(&haveName, &haveFriend)
					fmt.Println(haveName + "  " + haveFriend)
				}

				if user_id > 0 { //要添加的人存在
					switch username {
					case user:
						websocket.Message.Send(ws, "addFriendFail/不能添加自己")
					case haveFriend:
						websocket.Message.Send(ws, "addFriendFail/要添加的好友已存在")
					default:
						db.Exec("insert into friends(username,friend) value(?,?)", user, username)
						//db.Exec("insert into friends(username,friend) value(?,?)", username, user)
						friends = friends + "/" + username
						toAll = append(toAll, username)
						websocket.Message.Send(ws, friends) //好友列表
						websocket.Message.Send(ws, "添加好友："+hand[1]+"成功")
						fmt.Println("添加好友成功")
						websocket.Message.Send(ws, "shouldRefresh")
						//websocket.Message.Send(connClns[username].ws, user+"将你添加为好友")
						//websocket.Message.Send(connClns[username].ws, user+"将你添加为好友")
					}
				} else {
					websocket.Message.Send(ws, "addFriendFail/要添加的好友不存在")
					fmt.Println("添加好友失败")
				}
				db.Close()
			}
		//删除好友
		case "delFriend":
			{
				db, _ := sql.Open("mysql", "root:root@/chatroom?charset=utf8")
				for i := 1; i < len(hand); i++ {
					db.Exec("delete from friends where username=? and friend=?", user, hand[i])
					websocket.Message.Send(ws, "成功删除:"+hand[i])
					friends = strings.Replace(friends, "/"+hand[i], "", 1)
					//friend2 = strings.Replace(friends, "friendsList", "flref", 1)
					fmt.Println(friends)
					for j := range toAll {
						if hand[i] == toAll[j] {
							toAll = append(toAll[:j], toAll[j+1:]...)
						}
					}
					websocket.Message.Send(ws, friends)
					fmt.Printf("删除%s成功", hand[i])
				}
				db.Close()
			}
		default:
			{
				parSendMsg := "你对"
				for i := 1; i < len(hand); i++ {
					if connClns[hand[i]].ip != "" {
						websocket.Message.Send(connClns[hand[i]].ws, user+"对你说:"+hand[0])
					} else {
						db, _ := sql.Open("mysql", "root:root@/chatroom?charset=utf8")
						db.Exec("insert into offlinemsg(wfrom,wto,msg) value(?,?,?)", user, hand[i], hand[0])
						//fmt.Println("fffffffff")
						db.Close()
					}
					parSendMsg += hand[i] + ","
				}
				parSendMsg += "说:" + hand[0]
				websocket.Message.Send(ws, parSendMsg)
			}
		}
	}
}
