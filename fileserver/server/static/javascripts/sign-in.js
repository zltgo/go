
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
				"Captcha":"",
				"rememberme":false
				},
			"noteinfo":{
				"Name":"",
				"Password":"",
				"bShowCpt":false
				}
			},

		methods:{
			"LogInfo":function (psddata){
							try
							{
								localPost("/api/login", psddata, function(data){
										delCookie("showChe");
										window.location="index.html";
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
												alert("登陆错误，请检查");
												break;
												procError(XMLhttpRequest);

										}
										PreProCaptcha();
									}
								);
							}
							catch (err)
							{
								alert("登录出现了点问题");
							}
							return ;
				},
			"DoLogin":function (){
						try
						{					
							try
							{
								if(this.lf.rememberme)
								{
									setCookie("loginname",$("#usr").val());
									setCookie("loginpwd",$("#pwd").val());
									setCookie("autologin","yesitis");
								}
								else
								{
									delCookie("loginname");
									delCookie("loginpwd");
									delCookie("autologin");
								}						
							}
							catch (err)
							{
								alert("写入Cookie失败，请检查浏览器设置");
							}
							this.LogInfo(this.lf);
							return;							
						}
						catch (err)
						{
							window.location="500.html";
						}
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