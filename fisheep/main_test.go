package main

import (
	"encoding/json"
	"github.com/zltgo/api/client"
	"github.com/zltgo/fisheep/model"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/mgo.v2/bson"
)

var iphone = "15852637199"

func getCaptcha(t *testing.T, g *gin.Engine) ([]*http.Cookie, string) {
	// get captha
	req, _ := client.NewRequest("GET", "/captcha", nil)
	res := httptest.NewRecorder()
	req.RemoteAddr = "127.0.0.1:9527"
	g.ServeHTTP(res, req)

	if res.Code != 200 {
		t.Error("获取短信验证码错误：", res.Code)
		return nil, ""
	}
	return res.Result().Cookies(), res.Body.String()
}

func TestUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	g := publicRoutes()

	var tk jwt.AuthToken
	//delete recored
	model.MS.DB("").C(model.Users).RemoveAll(nil)

	// get sms
	req, _ := client.NewRequest("GET", "/sms?Phone="+iphone, nil)
	req.RemoteAddr = "127.0.0.1:9527"
	res := httptest.NewRecorder()
	g.ServeHTTP(res, req)
	if res.Code != 200 {
		t.Error("获取短信验证码错误：", res.Code)
		return
	}
	sms := res.Body.String()

	Convey("test login with sms\n", t, func() {
		form := structure.Form{}
		form.Set("Phone", iphone)
		form.Set("Sms", "wrong")
		r, _ := client.NewRequest("POST", "/login/usr", form)
		w := httptest.NewRecorder()
		r.RemoteAddr = "127.0.0.1:9527"
		g.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, model.StatusSmsError)

		form.Set("Sms", sms)
		r, _ = client.NewRequest("POST", "/login/usr", form)
		r.RemoteAddr = "127.0.0.1:9527"
		w2 := httptest.NewRecorder()
		g.ServeHTTP(w2, r)
		So(w2.Code, ShouldEqual, 200)

		err := json.NewDecoder(w2.Body).Decode(&tk)
		So(err, ShouldBeNil)
		t.Log(tk)
	})

	Convey("test reset password\n", t, func() {
		Convey("reset password with StatusCaptchaError\n", func() {
			form := structure.Form{}
			form.Set("Sms", sms)
			form.Set("Pwd", "11235813")
			form.Set("Captcha", "wrong")
			r, _ := client.NewRequest("PUT", "/usr/pwd", form)
			r.RemoteAddr = "127.0.0.1:9527"
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			w := httptest.NewRecorder()
			g.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, model.StatusCaptchaError)
		})

		Convey("reset password with StatusSmsError\n", func() {
			form := structure.Form{}
			form.Set("Sms", "wrong")
			form.Set("Pwd", "11235813")
			cks, captcha := getCaptcha(t, g)
			form.Set("Captcha", captcha)

			r, _ := client.NewRequest("PUT", "/usr/pwd", form)
			r.RemoteAddr = "127.0.0.1:9527"
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.AddCookie(cks[0])
			r.AddCookie(cks[1])
			w2 := httptest.NewRecorder()
			g.ServeHTTP(w2, r)
			So(w2.Code, ShouldEqual, model.StatusSmsError)
		})

		Convey("should accurately reset password with sms\n", func() {
			form := structure.Form{}
			form.Set("Sms", sms)
			form.Set("Pwd", "11235813")
			cks, captcha := getCaptcha(t, g)
			form.Set("Captcha", captcha)
			r, _ := client.NewRequest("PUT", "/usr/pwd", form)
			r.RemoteAddr = "127.0.0.1:9527"
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.AddCookie(cks[0])
			r.AddCookie(cks[1])

			w2 := httptest.NewRecorder()
			g.ServeHTTP(w2, r)
			So(w2.Code, ShouldEqual, 200)

			err := json.NewDecoder(w2.Body).Decode(&tk)
			So(err, ShouldBeNil)
			t.Log(tk)
		})
	})

	Convey("test login with pwd\n", t, func() {
		form := structure.Form{}
		form.Set("Phone", iphone)
		form.Set("Sms", "wrong")
		form.Set("Pwd", "wrongpwd")
		r, _ := client.NewRequest("POST", "/login/usr", form)
		w := httptest.NewRecorder()
		r.RemoteAddr = "127.0.0.1:9527"
		g.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, model.StatusPwdError)

		form.Set("Pwd", "11235813")
		r, _ = client.NewRequest("POST", "/login/usr", form)
		r.RemoteAddr = "127.0.0.1:9527"
		w2 := httptest.NewRecorder()
		g.ServeHTTP(w2, r)
		So(w2.Code, ShouldEqual, 200)

		err := json.NewDecoder(w2.Body).Decode(&tk)
		So(err, ShouldBeNil)
		t.Log(tk)
	})

	Convey("test reset phone\n", t, func() {
		Convey("should returns StatusCaptchaError \n", func() {
			form := structure.Form{}
			form.Set("Phone", iphone)
			form.Set("Sms", sms)
			form.Set("Pwd", "11235813")
			form.Set("Captcha", "wrong")
			r, _ := client.NewRequest("PUT", "/usr/phone", form)
			r.RemoteAddr = "127.0.0.1:9527"
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			w := httptest.NewRecorder()
			g.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, model.StatusCaptchaError)
		})

		Convey("should returns StatusSmsError \n", func() {
			form := structure.Form{}
			form.Set("Phone", iphone)
			form.Set("Sms", "wrong")
			form.Set("Pwd", "11235813")
			cks, captcha := getCaptcha(t, g)
			form.Set("Captcha", captcha)

			r, _ := client.NewRequest("PUT", "/usr/phone", form)
			r.RemoteAddr = "127.0.0.1:9527"
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.AddCookie(cks[0])
			r.AddCookie(cks[1])

			w2 := httptest.NewRecorder()
			g.ServeHTTP(w2, r)
			So(w2.Code, ShouldEqual, model.StatusSmsError)
		})

		Convey("should returns StatusPwdError \n", func() {
			form := structure.Form{}
			form.Set("Phone", iphone)
			form.Set("Sms", sms)
			form.Set("Pwd", "wrongpwd")
			cks, captcha := getCaptcha(t, g)
			form.Set("Captcha", captcha)

			r, _ := client.NewRequest("PUT", "/usr/phone", form)
			r.RemoteAddr = "127.0.0.1:9527"
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.AddCookie(cks[0])
			r.AddCookie(cks[1])

			w2 := httptest.NewRecorder()
			g.ServeHTTP(w2, r)
			So(w2.Code, ShouldEqual, model.StatusPwdError)
		})

		Convey("should return StatusRegisteredYet \n", func() {
			model.MS.DB("").C(model.Users).Insert(bson.M{"phone": "18626361318"})

			// get sms
			req, _ := client.NewRequest("GET", "/sms?Phone=18626361318", nil)
			req.RemoteAddr = "127.0.0.1:9527"
			res := httptest.NewRecorder()
			g.ServeHTTP(res, req)
			if res.Code != 200 {
				t.Error("获取短信验证码错误：", res.Code)
				return
			}

			form := structure.Form{}
			form.Set("Phone", "18626361318")
			form.Set("Sms", res.Body.String())
			form.Set("Pwd", "11235813")
			cks, captcha := getCaptcha(t, g)
			form.Set("Captcha", captcha)

			r, _ := client.NewRequest("PUT", "/usr/phone", form)
			r.RemoteAddr = "127.0.0.1:9527"
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.AddCookie(cks[0])
			r.AddCookie(cks[1])

			w2 := httptest.NewRecorder()
			g.ServeHTTP(w2, r)
			So(w2.Code, ShouldEqual, model.StatusRegisteredYet)
		})

		Convey("should accurately reset phone \n", func() {
			// get sms
			req, _ := client.NewRequest("GET", "/sms?Phone=15172436633", nil)
			req.RemoteAddr = "127.0.0.1:9527"
			res := httptest.NewRecorder()
			g.ServeHTTP(res, req)
			if res.Code != 200 {
				t.Error("获取短信验证码错误：", res.Code)
				return
			}

			form := structure.Form{}
			form.Set("Phone", "15172436633")
			form.Set("Sms", res.Body.String())
			form.Set("Pwd", "11235813")
			cks, captcha := getCaptcha(t, g)
			form.Set("Captcha", captcha)

			r, _ := client.NewRequest("PUT", "/usr/phone", form)
			r.RemoteAddr = "127.0.0.1:9527"
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.AddCookie(cks[0])
			r.AddCookie(cks[1])

			w2 := httptest.NewRecorder()
			g.ServeHTTP(w2, r)
			So(w2.Code, ShouldEqual, 200)
		})

		Convey("should accurately reset back \n", func() {

			form := structure.Form{}
			form.Set("Phone", iphone)
			form.Set("Sms", sms)
			form.Set("Pwd", "11235813")
			cks, captcha := getCaptcha(t, g)
			form.Set("Captcha", captcha)

			r, _ := client.NewRequest("PUT", "/usr/phone", form)
			r.RemoteAddr = "127.0.0.1:9527"
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.AddCookie(cks[0])
			r.AddCookie(cks[1])

			w2 := httptest.NewRecorder()
			g.ServeHTTP(w2, r)
			So(w2.Code, ShouldEqual, 200)

			err := json.NewDecoder(w2.Body).Decode(&tk)
			So(err, ShouldBeNil)
			t.Log(tk)
		})
	})

	Convey("test reset addrs\n", t, func() {
		af := model.AddrsForm{
			[]model.Address{
				{"123456789012345678901234", "张三", "18626361318", "湖北潜江"},
				{"123456789012345678901234", "李四", "15852637199", "湖南长沙"},
			},
		}

		r, _ := client.NewRequest("PUT", "/usr/addrs", af)
		w := httptest.NewRecorder()
		r.RemoteAddr = "127.0.0.1:9527"
		r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
		g.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, 200)

		var mp map[string]interface{}
		type addrs struct {
			Addrs model.Address
		}
		err := model.MS.DB("").C(model.Users).Find(addrs{
			model.Address{"123456789012345678901234", "张三", "18626361318", "湖北潜江"},
		}).One(&mp)
		t.Log(err, mp)
	})
}

func TestManager(t *testing.T) {
	gin.SetMode(gin.TestMode)
	g := privateRoutes()
	var tk jwt.AuthToken
	var gid string
	Convey("test POST:/login/manager\n", t, func() {
		form := structure.Form{}
		form.Set("Usr", model.DefaultSuperUsr)
		form.Set("Pwd", model.DefaultSuperPwd)
		cks, captcha := getCaptcha(t, g)
		form.Set("Captcha", captcha)

		r, _ := client.NewRequest("POST", "/login/manager", form)
		w := httptest.NewRecorder()
		r.RemoteAddr = "127.0.0.1:9527"
		r.AddCookie(cks[0])
		r.AddCookie(cks[1])
		g.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, 200)

		err := json.NewDecoder(w.Body).Decode(&tk)
		So(err, ShouldBeNil)
	})

	Convey("test POST:/manager\n", t, func() {
		Convey("should return http.StatusUnauthorized", func() {
			form := structure.Form{}
			form.Set("Usr", model.DefaultSuperUsr)
			form.Set("Pwd", model.DefaultSuperPwd)
			form.Set("Grp", model.SuperGrp)
			form.Set("AreaId", node.RootId)

			r, _ := client.NewRequest("POST", "/manager", form)
			w := httptest.NewRecorder()
			r.RemoteAddr = "127.0.0.1:9527"
			g.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusUnauthorized)
		})

		Convey("should return StatusRegisteredYet", func() {
			form := structure.Form{}
			form.Set("Usr", model.DefaultSuperUsr)
			form.Set("Pwd", model.DefaultSuperPwd)
			form.Set("Grp", model.SuperGrp)
			form.Set("AreaId", "123456789012345678901234")

			r, _ := client.NewRequest("POST", "/manager", form)
			w := httptest.NewRecorder()
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.RemoteAddr = "127.0.0.1:9527"
			g.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, model.StatusRegisteredYet)
		})

		Convey("should accurately add a manager", func() {
			//delete recored
			model.MS.DB("").C(model.Managers).Remove(bson.M{"usr": "大王"})

			form := structure.Form{}
			form.Set("Usr", "大王")
			form.Set("Pwd", model.DefaultSuperPwd)
			form.Set("Grp", model.SuperGrp)
			form.Set("AreaId", "123456789012345678901234")

			r, _ := client.NewRequest("POST", "/manager", form)
			w := httptest.NewRecorder()
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.RemoteAddr = "127.0.0.1:9527"
			g.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 200)
		})
	})

	Convey("test GET:/managers\n", t, func() {
		form := structure.Form{}
		r, _ := client.NewRequest("GET", "/managers", form)
		w := httptest.NewRecorder()
		r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
		r.RemoteAddr = "127.0.0.1:9527"
		g.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, 200)

		var mp interface{}
		err := json.NewDecoder(w.Body).Decode(&mp)
		So(err, ShouldBeNil)
		So(mp, ShouldHaveLength, 2)
		t.Log(mp)

		form.Set("UsrName", "大")
		r, _ = client.NewRequest("GET", "/managers", form)
		w2 := httptest.NewRecorder()
		r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
		r.RemoteAddr = "127.0.0.1:9527"
		g.ServeHTTP(w2, r)
		So(w2.Code, ShouldEqual, 200)

		err = json.NewDecoder(w2.Body).Decode(&mp)
		So(err, ShouldBeNil)
		So(mp, ShouldHaveLength, 1)
		t.Log(mp)
		gid = mp.([]interface{})[0].(map[string]interface{})["Id"].(string)
	})

	Convey("test PUT:/manager\n", t, func() {
		form := structure.Form{}
		form.Set("Pwd", "mudadamu")
		form.Set("Id", gid)
		form.Set("Grp", "ssr")

		r, _ := client.NewRequest("PUT", "/manager", form)
		w := httptest.NewRecorder()
		r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
		r.RemoteAddr = "127.0.0.1:9527"
		g.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, 200)

		var mp interface{}
		form.Set("UsrName", "大")
		r, _ = client.NewRequest("GET", "/managers", form)
		w2 := httptest.NewRecorder()
		r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
		r.RemoteAddr = "127.0.0.1:9527"
		g.ServeHTTP(w2, r)
		So(w2.Code, ShouldEqual, 200)

		err := json.NewDecoder(w2.Body).Decode(&mp)
		So(err, ShouldBeNil)
		So(mp, ShouldHaveLength, 1)
		t.Log(mp)
	})

	Convey("test DELETE:/manager\n", t, func() {
		Convey("test DELETE:/manager\n", func() {
			r, _ := client.NewRequest("DELETE", "/manager/"+gid, nil)
			w := httptest.NewRecorder()
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.RemoteAddr = "127.0.0.1:9527"
			g.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 200)

			r, _ = client.NewRequest("GET", "/managers", nil)
			w2 := httptest.NewRecorder()
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.RemoteAddr = "127.0.0.1:9527"
			g.ServeHTTP(w2, r)
			So(w2.Code, ShouldEqual, 200)

			var mp interface{}
			err := json.NewDecoder(w2.Body).Decode(&mp)
			So(err, ShouldBeNil)
			So(mp, ShouldHaveLength, 1)
			t.Log(mp)
		})
	})

	var aid string
	var china string
	model.MS.DB("").C(model.Areas).RemoveAll(nil)
	Convey("test areas\n", t, func() {
		Convey("test POST:/area\n", func() {
			form := structure.Form{}
			form.Set("Name", "中国")

			r, _ := client.NewRequest("POST", "/area", form)
			w := httptest.NewRecorder()
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.RemoteAddr = "127.0.0.1:9527"
			g.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 200)
		})

		Convey("test GET:/areas\n", func() {
			r, _ := client.NewRequest("GET", "/areas", nil)
			w := httptest.NewRecorder()
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.RemoteAddr = "127.0.0.1:9527"
			g.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 200)

			var mp interface{}
			err := json.NewDecoder(w.Body).Decode(&mp)
			So(err, ShouldBeNil)
			So(mp, ShouldHaveLength, 1)
			t.Log("添加中国", mp)

			aid = mp.([]interface{})[0].(map[string]interface{})["Id"].(string)
			china = aid
		})

		Convey("add area again\n", func() {
			form := structure.Form{}
			form.Set("Name", "台湾")
			form.Set("Pid", aid)

			r, _ := client.NewRequest("POST", "/area", form)
			w := httptest.NewRecorder()
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.RemoteAddr = "127.0.0.1:9527"
			g.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 200)

			// get id of "台湾"
			form.Set("Id", aid)
			r, _ = client.NewRequest("GET", "/areas", form)
			w2 := httptest.NewRecorder()
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.RemoteAddr = "127.0.0.1:9527"
			g.ServeHTTP(w2, r)
			So(w2.Code, ShouldEqual, 200)

			var mp interface{}
			err := json.NewDecoder(w2.Body).Decode(&mp)
			So(err, ShouldBeNil)
			So(mp, ShouldHaveLength, 1)
			t.Log("添加台湾", mp)

			aid = mp.([]interface{})[0].(map[string]interface{})["Id"].(string)
		})

		Convey("test PUT:/area\n", func() {
			// reset name
			form := structure.Form{}
			form.Set("Name", "美国")
			form.Set("Id", aid)
			r, _ := client.NewRequest("PUT", "/area/name", form)
			w := httptest.NewRecorder()
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.RemoteAddr = "127.0.0.1:9527"
			g.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 200)

			// reset pid to root
			r, _ = client.NewRequest("PUT", "/area/pid", form)
			w3 := httptest.NewRecorder()
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.RemoteAddr = "127.0.0.1:9527"
			g.ServeHTTP(w3, r)
			So(w.Code, ShouldEqual, 200)

			// get children of root
			form.Set("Id", "")
			r, _ = client.NewRequest("GET", "/areas", nil)
			w2 := httptest.NewRecorder()
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.RemoteAddr = "127.0.0.1:9527"
			g.ServeHTTP(w2, r)
			So(w2.Code, ShouldEqual, 200)

			var mp interface{}
			err := json.NewDecoder(w2.Body).Decode(&mp)
			So(err, ShouldBeNil)
			So(mp, ShouldHaveLength, 2)
			t.Log("改成美国", mp)
		})

		Convey("test DELETE:/area\n", func() {
			form := structure.Form{}
			form.Set("Id", aid)
			r, _ := client.NewRequest("DELETE", "/areas", form)
			w := httptest.NewRecorder()
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.RemoteAddr = "127.0.0.1:9527"
			g.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 200)

			// get children of root
			form.Set("Id", "")
			r, _ = client.NewRequest("GET", "/areas", nil)
			w2 := httptest.NewRecorder()
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.RemoteAddr = "127.0.0.1:9527"
			g.ServeHTTP(w2, r)
			So(w2.Code, ShouldEqual, 200)

			var mp interface{}
			err := json.NewDecoder(w2.Body).Decode(&mp)
			So(err, ShouldBeNil)
			So(mp, ShouldHaveLength, 1)
			t.Log("删除美国", mp)
		})
	})

	model.MS.DB("").C(model.Goods).RemoveAll(nil)
	Convey("test POST:/goods\n", t, func() {
		Convey("should accurately add a goods", func() {
			var gb model.GoodsDb

			gb.Title = "哈密瓜"
			gb.Describe = "产地直采/脆嫩香甜"
			gb.Price = 1000
			gb.Weight = "1斤"
			gb.Sold = 1
			gb.Store = 2
			gb.Attributes = []model.Attribute{
				model.Attribute{
					Name:  "产地",
					Value: "新疆",
				},
			}
			gb.AreaId = china
			gb.ProductId = "123456789012345678901234"
			gb.Flag = 2    // 1:下架，2 上架
			gb.Order = "a" //排序用

			r, _ := client.NewRequest("POST", "/goods", gb)
			w := httptest.NewRecorder()
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.RemoteAddr = "127.0.0.1:9527"
			g.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 200)

			gb.Title = "香瓜"
			gb.Price = 2000
			gb.Order = "ab"
			r, _ = client.NewRequest("POST", "/goods", gb)
			w2 := httptest.NewRecorder()
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.RemoteAddr = "127.0.0.1:9527"
			g.ServeHTTP(w2, r)
			So(w2.Code, ShouldEqual, 200)
		})

		Convey("should return StatusAreaErr", func() {
			var gb model.GoodsDb

			gb.Title = "哈密瓜"
			gb.Describe = "产地直采/脆嫩香甜"
			gb.Price = 1000
			gb.Weight = "1斤"
			gb.Sold = 1
			gb.Store = 2
			gb.Attributes = []model.Attribute{
				model.Attribute{
					Name:  "产地",
					Value: "新疆",
				},
			}
			gb.AreaId = "123456789012345678901234"
			gb.ProductId = "123456789012345678901234"
			gb.Flag = 2    // 1:下架，2 上架
			gb.Order = "b" //排序用

			r, _ := client.NewRequest("POST", "/goods", gb)
			w := httptest.NewRecorder()
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.RemoteAddr = "127.0.0.1:9527"
			g.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, model.StatusAreaErr)
		})
	})

	var goods []model.GoodsDb
	Convey("test GET and PUT\n", t, func() {
		form := structure.Form{}
		form.Set("GoodsName", "瓜")
		form.Add("Sort", "-price")
		form.Add("Sort", "sold")

		r, _ := client.NewRequest("GET", "/goods", form)
		w := httptest.NewRecorder()
		r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
		r.RemoteAddr = "127.0.0.1:9527"
		g.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, 200)

		err := json.NewDecoder(w.Body).Decode(&goods)
		So(err, ShouldBeNil)
		So(goods, ShouldHaveLength, 2)
		t.Log("按价格降序排列", goods)
	})

	Convey("test PUT:/goods\n", t, func() {
		// 改价格、名字
		goods[0].Price = 3000
		goods[0].Title = "西瓜"
		r, _ := client.NewRequest("PUT", "/goods", goods[0])
		w2 := httptest.NewRecorder()
		r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
		r.RemoteAddr = "127.0.0.1:9527"
		g.ServeHTTP(w2, r)
		So(w2.Code, ShouldEqual, 200)

		form := structure.Form{}
		form.Set("GoodsName", "瓜")
		form.Add("Sort", "order")
		form.Add("Sort", "price")

		r, _ = client.NewRequest("GET", "/goods", form)
		w3 := httptest.NewRecorder()
		r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
		r.RemoteAddr = "127.0.0.1:9527"
		g.ServeHTTP(w3, r)
		So(w3.Code, ShouldEqual, 200)

		err := json.NewDecoder(w3.Body).Decode(&goods)
		So(err, ShouldBeNil)
		So(goods, ShouldHaveLength, 2)
		t.Log("修改名字和价格", goods)
	})

	Convey("test DELETE:/goods\n", t, func() {
		r, _ := client.NewRequest("DELETE", "/goods/"+goods[1].Id.Hex(), nil)
		w := httptest.NewRecorder()
		r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
		r.RemoteAddr = "127.0.0.1:9527"
		g.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, 200)

		form := structure.Form{}
		form.Set("GoodsName", "瓜")
		form.Add("Sort", "order")
		form.Add("Sort", "price")

		r, _ = client.NewRequest("GET", "/goods", form)
		w3 := httptest.NewRecorder()
		r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
		r.RemoteAddr = "127.0.0.1:9527"
		g.ServeHTTP(w3, r)
		So(w3.Code, ShouldEqual, 200)

		err := json.NewDecoder(w3.Body).Decode(&goods)
		So(err, ShouldBeNil)
		So(goods, ShouldHaveLength, 1)
		t.Log("删除西瓜", goods)
	})

	var goodsid string
	Convey("test POST:/image\n", t, func() {
		Convey("should accurately upload a image\n", func() {
			// 添加商品
			var gb model.GoodsDb
			gb.Title = "哈密瓜"
			gb.Describe = "产地直采/脆嫩香甜"
			gb.Price = 1000
			gb.Weight = "1斤"
			gb.Sold = 1
			gb.Store = 2
			gb.Attributes = []model.Attribute{
				model.Attribute{
					Name:  "产地",
					Value: "新疆",
				},
			}
			gb.AreaId = china
			gb.ProductId = "123456789012345678901234"
			gb.Flag = 2    // 1:下架，2 上架
			gb.Order = "c" //排序用

			r, _ := client.NewRequest("POST", "/goods", gb)
			w := httptest.NewRecorder()
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.RemoteAddr = "127.0.0.1:9527"
			g.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 200)

			// 上传图片
			goodsid = w.Body.String()[1:25]
			r, err := client.NewUploadRequest("/image/"+goodsid, "File", "哈密瓜特写.jpg", "./test/upload_image.jpg")
			So(err, ShouldBeNil)

			w2 := httptest.NewRecorder()
			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.RemoteAddr = "127.0.0.1:9527"
			g.ServeHTTP(w2, r)
			So(w2.Code, ShouldEqual, 200)
		})
	})
}
