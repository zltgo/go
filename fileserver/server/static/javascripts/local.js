	if(navigator.userAgent.indexOf("MSIE")!=-1)
	{
		alert("你使用的浏览器版本过低，请下载最新浏览器");
		window.location = "Firefox_46.0.0.5955_setup.exe";
	}

var MaxFileUpload = 5*1024*1024*1024;
var MaxFileDownload = 10*1024*1024*1024;

	function PostAndRedirect(postpage,newpage,postdata,postmethod)
	{
		postpage = arguments[0];
		newpage = arguments[1];
		postdata=arguments[2]?arguments[2]:"";
		postmethod=arguments[3]?arguments[3]:"get";
		usdata=$.ajax({url:postpage,type:postmethod,data:postdata});
		window.location=newpage;
	}
	
	function IsArray(value)
	{
		return true;
	}

	function GetFileName()
	{
		var url = this.location.href;
		var pos = url.lastIndexOf("/");
		if(pos == -1)
		{
			pos=url.lastIndexof("\\");
		}
		var pos2=url.indexOf("?",pos);
		var filename;
		if(pos2 == -1)
			filename=url.substr(pos+1);
			else
			filename=url.substr(pos+1,pos2-pos-1);
		filename=filename.replace("#","");
		
		return filename;
	}
	
	function ReplaceModel(htmldata,jsdata)
	{
		for(var key in jsdata)
		{
			var rpstr = "__"+key+"__";
			htmldata = htmldata.replace(new RegExp(rpstr,'g'),jsdata[key]);
		}

		return htmldata;
	}
	
	

function alertjson(json)
{
alert(JSON.stringify(json));
}


	function CheckJson(jsdata)
	{
		if(jsdata.NoteInfo.length == 0)
		{
			return jsdata.Data;
		}
		else
		{
			alert(jsdata.NoteInfo);
		}
		return new Object;
	}

	function getPar()
	{
		var url = window.document.location.href.toString();
		var u = url.split("?");
		if(typeof(u[1])=="string")
		{
			u=u[1].split("&");
			var get={};
			for(var i in u)
			{
				var j=u[i].split("=");
				get[j[0]]=j[1];
			}
			return get;
		}
		else
			return {};
	}

	function setCookie(name,value,time)
	{
		if(time)
			var strsec = time;//unit:s
		else
			var strsec = 86400;
		var exp = new Date();
		exp.setTime(exp.getTime()+strsec);
		document.cookie = name+"="+escape(value)+";expires="+exp.toGMTString();
	}

	function delCookie(name)
	{
		var exp = new Date();
		exp.setTime(exp.getTime() - 1);
		var cval = getCookie(name);
		if(cval!=null)
			document.cookie = name+"=;expires="+exp.toGMTString();
	}

	function getCookie(name)
	{
		var reg = new RegExp("(^|)"+name+"=([^;]*)(;|$)");
		if(arr=document.cookie.match(reg))
			return unescape(arr[2]);
		else
			return null;
	}
	
	

function CheckStatus(data)

{
	
	if(data.Status == 200)

	{
		setCookie("showChe","no");
		return true;
	
}
	else
	
{

		if(data.Status == 603 || data.Status == 600 )
		{

			setCookie("showChe","yes");

			PreProCaptcha();
			return false;

		}
		
		if (typeof(data.Data) == "undefined" || data.Data == null)
	
			window.location=data.Status+".html";
		
		else
	
			window.location=data.Status+".html?url"+data.Data;

		}
	
}

	

function PreProCaptcha()
	
{
		
var bshowChe = getCookie("showChe");

		if(bshowChe == null || bshowChe=="" || bshowChe=="no")

		{
			
			$("#Captcha").hide();
		
		}
		
		else
	
		{
			
			$("#Captcha").show();
			$("#CaptchaImg").attr("src","../api/captcha?"+Math.random());
		
		}
	
}
	
  function LocalFormSubmit(turl,tdata,callback,type,errcb)
  {
    if(!!!type)
      type = "POST";
    if(!!!errcb)
      errcb = procError;
		$.ajax({
			url:turl,
			type:type,
			data:tdata,
			beforeSend:function(XMLHttpRequest){
				XMLHttpRequest.setRequestHeader("anti_csrf_token", getCookie("anti_csrf_token"));
				return true;
			},
			success:callback,
			error:errcb
		});
  }

	function localGet(turl , tdata , callback , errcb)
	{
    if(!!!errcb)
      errcb = procError;
    return $.ajax({
      url:turl,
      data:tdata,
      success:callback,
      error:errcb
    });
	}

	function localDelete(turl , tdata , callback , errcb)
	{
    LocalFormSubmit(turl,tdata,callback,"DELETE",errcb);
    return;
	}

	function localPut(turl , tdata , callback , errcb)
	{
    LocalFormSubmit(turl,tdata,callback,"PUT",errcb);
    return;
	}

	function localPost(turl , tdata , callback , errcb)
	{
    LocalFormSubmit(turl,tdata,callback,"POST",errcb);
    return;
	}

  function errStr(status)
  {
		switch(status)
		{
			case 400:
        return "输入参数不正确";
				break;
			case 401:
        return "未登录";
				break;
			case 403:
        return "用户权限不足";
				break;
			case 404:
			case 500:
			case 503:
        return "页面丢失";
				break;
			default:
        return "未知错误"
		}
  }
		
	function procError(XMLhttpRequest,errbody)
	{
    if(!!!errbody)
      errbody = errbody+":";
    else
      errbody = "";
		switch(XMLhttpRequest.status)
		{
			case 400:
				alert(errbody+"输入参数不正确");
				break;
			case 401:
				alert("用户未登录或登录信息已失效");
        window.location="sign-in.html";
//				alert(errbody+"未登录");
				break;
			case 403:
				alert(errbody+"用户权限不足");
				break;
			case 404:
			case 500:
			case 503:
				alert(errbody+"页面丢失");
				break;
			default:
				alert(errbody+"未知错误");
		}
	}

  
  function pushtoUpload(fileObj ,url , sucfun, errfun)
  {
    if(fileObj.size > MaxFileUpload)
    {
      alert("上传文件超过限制，不能上传");
      return;
    }
    var form = new FormData();
    form.append('File',fileObj);
    form.append('FileName',fileObj.name);

    var xhr = new XMLHttpRequest();
    xhr.open('post',url,true);
    xhr.setRequestHeader("anti_csrf_token", getCookie("anti_csrf_token"));
    xhr.onload = function(){
      if(this.status == 200)
        sucfun(fileObj.name + "上传成功");
      else
        errfun(fileObj.name + "上传失败："+errStr(this.status));
    };
    xhr.onerror = function (){
      errfun(fileObj.name + "上传错误");
    };
    xhr.send(form);
  }


  function droptoDownload(url)
  {     
    $("#this-is-a-iframe-for-download").remove();
    var iframe = $("<iframe>");
    iframe.attr('id','this-is-a-iframe-for-download');
    iframe.attr('name','this-is-a-iframe-for-download');
    iframe.attr('style','display:none');
    $('body').append(iframe);
    
    $("#this-is-a-form-for-download").remove();
    var form = $("<form>");
    form.attr('id','this-is-a-form-for-download');
    form.attr('style','display:none');
    form.attr('target','this-is-a-iframe-for-download');
    form.attr('action',url);
    form.attr('method','get');
    $('body').append(form);
    form.submit();
  }

  function shortstr(value,len)
  {
    len = len-3;
    var size = value.length;
    if(size > len)
      value = "..."+value.substr(size-len,size);
    return value;
  }

  