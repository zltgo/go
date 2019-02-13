
//读取Cookie
var autoname = getCookie("loginname");
var autopwd = getCookie("loginpwd");
var bautologin = getCookie("autologin");

var formvue;
//页面装载成功函数
$(document).ready(function(){
	
	//处理验证码状态
	PreProCaptcha();
	
	//读取Cookie


	//创建表单Vue
	formvue = new Vue({
		el: '#theform', 
		data:{
			"lf": {
				"Name":"",
				"Password":"",
				"CfmPassword":"",
				"RealName":"姓名",
				"Captcha":""
				},
			"noteinfo":{
				"Name":"",
				"Password":""
				}
			},

		methods:{
			"PrePost":function (psddata){
							if(this.lf.Name.length < 5)
								return false;
							if(this.lf.Password.length < 5)
								return false;
							if(this.lf.RealName.length < 1 || this.lf.RealName.length > 4)
								return false;
							return true;
					},

			"LogInfo":function (){
							var psddata = this.lf;
							if(this.PrePost(psddata) == false)
							{
								alert("信息输入错误，请检查");
								return;
							}
							try
							{
								localPost("/api/usr", psddata, function(data){
										delCookie("showChe");
                    alert("注册成功");

                    localPost("/api/login", psddata, function(data){
                      delCookie("showChe");
                      window.location="index.html";
                    },
                    function(){
                      window.location="sign-in.html";
                    });

									},
									function (XMLhttpRequest,textStatus,errorThrown)
									{
										switch(XMLhttpRequest.status)
										{
											case 600:
												alert("验证码不正确");
												setCookie("showChe","yes");
												PreProCaptcha();
												break;
											case 601:
												alert("用户名或密码不正确");
												break;
											case 603:
												setCookie("showChe","yes");
												PreProCaptcha();
												alert("用户名或密码不正确");
												break;
											default:
												alert(XMLhttpRequest.status + "注册失败，请检查");
												break;
												procError(XMLhttpRequest);

										}
										PreProCaptcha();
									}
								);
							}
							catch (err)
							{
								alert("注册出现了点问题");
							}
							return ;
				}
			}
		});
	
	if(bautologin=="yesitis")
	{
		formvue.lf.Name = autoname;
		formvue.lf.Password = autopwd;
		formvue.lf.rememberme = true;
	}
});