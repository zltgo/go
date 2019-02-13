
var UserInfo ,globalSysset;

//获取用户信息
$.ajax({url:"/api/usr",async:false,data:{},success:function(data){
  UserInfo = data;
  if(UserInfo.Class=="系统管理员") UserInfo.Level = 4;
  else if(UserInfo.Class=="配置管理员") UserInfo.Level = 3;
  else if(UserInfo.Class=="普通用户") UserInfo.Level = 2;
  else if(UserInfo.Class=="游客") UserInfo.Level = 1;
  },
  error:procError
});

//获取系统信息
if(UserInfo.Level>=3)
  $.get("/api/conf",{},function(data){globalSysset = data;} );


var treemenus = Vue.extend({
  name:'item',
  template: '#item-template',
  replace: true,
  props: {
    model: Object
  },
  ready:function(){
    if(this.model.path=="/")
      {
      this.toggle();
      }
  },
  data: function () {
    return {
      open: false
    }
  },
  methods: {
    toggle: function () {
      if(!this.open)
      {
        var obj = this.model;
        localGet("/api/dir"+obj.path,{},function(data){
              obj.children = [];
            for(var key in data)
            {
             if(data[key].IsDir)
               obj.children.push({name:data[key].Name,isFolder:true,path:data[key].Path+data[key].Name+"/",children:[]});
            }
        });
      }
      this.open = !this.open;
      return true;
    },
    openfolder:function(){
      this.$route.router.go({ path: '/files',query:{path:this.model.path}});
    }
  }
})

var treemenu = Vue.extend({
  components:{
    "item":treemenus
    },
  template:'#treemenu-template',
  data: function (){
    return {
      searchtext:"",  
      treeData:{name:'主目录',children:[],isFolder:true,path:"/"}
    };
  },
  methods:{
    dosearch:function(){
      if(this.searchtext.trim())
        this.$route.router.go({ path: '/files',query:{path:'/',search:true,expr:this.searchtext}});
    }
  }
});


//Left sidebar
var dropmenu = Vue.extend({
  template: '#leftsidebar-template',
  data:function(){
    var tmplist = {
        'fileMange-menu':{'note':'下载管理','icon':'glyphicon-book','status':true,'submenu':[
          {'note':'下载排行','icon':'glyphicon-fire','status':true,'bvlink':true,'link':{path: '/dlcount'}},
          {'note':'下载记录','icon':'glyphicon-import','status':false,'bvlink':true,'link':{path: '/dllist'}},
          {'note':'下载趋势','icon':'glyphicon-align-left','status':true,'bvlink':true,'link':{path: '/dlchart'}}
          ]},
        'userMange-menu':{'note':'用户管理','icon':'glyphicon-wrench','status':false,'submenu':[
          {'note':'用户列表','icon':'glyphicon-list','status':true,'bvlink':true,'link':{ path: '/user'}},
          {'note':'添加用户','icon':'glyphicon-plus','status':false,'bvlink':true,'link':{ path: '/addusr'}},
          {'note':'系统设置','icon':'glyphicon-globe','status':false,'bvlink':true,'link':{ path: '/sysset'}}
          ]},
        'selfMange-menu':{'note':'个人管理','icon':'glyphicon-user','status':true,'submenu':[
          {'note':'信息设置','icon':'glyphicon-user','status':true,'bvlink':true,'link':{ path: '/usrset'}},
          {'note':'退出系统','icon':'glyphicon-log-out','status':true,'bvlink':false,'link':"/api/logout"}
          ]}
    };
    
    if(UserInfo.Level > 2)
    {
      tmplist['userMange-menu'].status=true;
      tmplist['fileMange-menu']['submenu'][1].status=true;
    }
    if(UserInfo.Level > 3)
    {
      tmplist['userMange-menu']['submenu'][1].status=true;
      tmplist['userMange-menu']['submenu'][2].status=true;
    }

    return {
      menuList:tmplist
    };
  },
  methods:{
    doclick:function(submenu){
      if(submenu.bvlink)
        this.$route.router.go(submenu.link);
      else
        window.location=submenu.link;
    }
  }
})


var leftmenu = Vue.extend({
  components:{
    "treemenu":treemenu,
    "dropmenu":dropmenu
    },
  template:"#leftmenu-template",
  data: function (){
    return {
      currentView: this.$route.name=="FileMG"?'treemenu':'dropmenu',
      cominfo:{treemenu:{note:"文件系统",icon:"glyphicon-list-alt",alttext:"切换至管理系统",
                          action:function(){this.currentView="dropmenu";
                                          $route.router.go({ path: '/usrset'});}},
               dropmenu:{note:"管理系统",icon:"glyphicon-cog",alttext:"切换至文件系统",
                          action:function(){this.currentView="treemenu";
                                          $route.router.go({ path: '/files',query:{path:'/'}});}}
        }
    };
  },
  computed:{
      item:function(){
        return this.cominfo[this.currentView];
      }
  },
  methods:{
    livenowcom:function(){
      if(this.currentView == "treemenu")
      {
        this.currentView = "dropmenu";
        this.$route.router.go({ path: '/usrset'});
      }
      else
      {
        this.$route.router.go({ path: '/files',query:{path:'/'}});
        this.currentView = "treemenu";
      }
    }
  }
});


var bottomtable = Vue.extend({
  template: '#bottomtable-template',
  replace: true,
  props: {
    searchInfo: Object,
    totalNum: Number
  },
  data: function () {
      return {
      itemsChoose:[1,2,5,10,25,50,100]
    };
  },
  
  computed:{
    totalpages:function(){return Math.ceil(this.totalNum/this.searchInfo.OnePageCount);},
    pageshow:function(){
      var tparray = [{note:"<<",page:1,class:"page-first"},
                     {note:"<", page:this.searchInfo.Page>1?this.searchInfo.Page-1:1,class:"page-pre"}];
      
      switch(this.totalpages){ 
        case 0:case 1: 
          tparray.push( {note:"1",page:1,class:"page-number"});
          break;
        case 2: 
          tparray.push( {note:"1",page:1,class:"page-number"});
          tparray.push( {note:"2",page:2,class:"page-number"});
          break;
        case 3:
          tparray.push( {note:"1",page:1,class:"page-number"});
          tparray.push( {note:"2",page:2,class:"page-number"});
          tparray.push( {note:"3",page:3,class:"page-number"});
          break;
        default:
          if(this.totalpages<0)
            break;
          if(this.searchInfo.Page<=2)
          {
            tparray.push( {note:"1",page:1,class:"page-number"});
            tparray.push( {note:"2",page:2,class:"page-number"});
            tparray.push( {note:"3",page:3,class:"page-number"});
          }
          else if(this.searchInfo.Page>=this.totalpages-1)
          {
            tparray.push( {note:this.totalpages-2,page:this.totalpages-2,class:"page-number"});
            tparray.push( {note:this.totalpages-1,page:this.totalpages-1,class:"page-number"});
            tparray.push( {note:this.totalpages,page:this.totalpages,class:"page-number"});
          }
          else
          {
            tparray.push( {note:this.searchInfo.Page-1,page:this.searchInfo.Page-1,class:"page-number"});
            tparray.push( {note:this.searchInfo.Page,page:this.searchInfo.Page,class:"page-number"});
            tparray.push( {note:this.searchInfo.Page+1,page:this.searchInfo.Page+1,class:"page-number"});
          }
          break;
      }

      tparray.push({note:">", page:this.searchInfo.Page<this.totalpages?this.searchInfo.Page+1:this.totalpages,class:"page-next"});
      tparray.push({note:">>",page:this.totalpages,class:"page-last"});
      
      return tparray;
    }
  }
})


var ContMenu = Vue.extend({
  template:'#contextmenu-template',
  replace: true,
  props: {
    bshow: Boolean,
    menus: Object,
    posx:Number,
    posy:Number
  },
  watch:{
  },
  computed:{
    posstyle:function(){
      
      return {
        cursor:'default',
        position:'absolute',
        top:this.posx+"px",
        left:this.posy+"px"
      };
    },
    getmenus:function(){
      return this.menus;
    }
  },
  methods:{
    postclick:function(obj){
      this.bshow = false;
      this.$dispatch('innercontextmenu', obj);
    },
    hasicon:function(item){
      return (!!item && !!item.icon && item.icon.length > 0);
    }
  }
});


var FileCon = Vue.extend({
  template: '#filetable-template',
  components:{
    "context-menu":ContMenu
    },
  ready:function(){
      this.reflush();
      if(typeof(this.sysSet) == "undefined")
      {
        var obj = this;
        $.get("/api/conf",{},function(data){obj.sysSet = data;} );
      }
    },
  data: function () {
    var sortOrders = {}
    var thecolumns = {Path:'目录',Name:'名称',FileSize:'大小',ModTime:'修改时间'};
    for(var key in thecolumns)
    {
      sortOrders[key] = -1;
	  }
    var thisdata =[];
    var tpath = this.$route.query.path;
    if(!tpath || tpath.length < 2)
      tpath = "/";
    return {
      sysSet:globalSysset,
      tableforsearch:false,
      cache:{bcache:false,data:{"/":thisdata}},
      userLevel:UserInfo.Level,
      searchmode:false,
      columns:thecolumns,
      path:tpath,
      data:[],
      sortKey: 'Name',
      sortOrders: sortOrders,
      selectList:{},
      ReNameValue:'',
      filterKey:'',
      bEdit:false,
      uploadFolder:false,
      editDlgInfo:{title:'',placeholder:''},
      btnlist:{
      'reflush':{note:'刷新',icon:'glyphicon-refresh',level:1},
      'uplevel':{note:'向上',icon:'glyphicon-arrow-up',level:1},
      'preview':{note:'预览',icon:'glyphicon-film',level:2},
      'rename':{note:'重命名',icon:'glyphicon-pencil',level:3},
      'newfolder':{note:'新建',icon:'glyphicon-plus',level:3},
      'rmfile':{note:'删除',icon:'glyphicon-remove',level:3},
      'upload':{note:'上传',icon:'glyphicon-upload',level:3},
      'packetdown':{note:'下载',icon:'glyphicon-download',level:2},
//      'topos':{note:'定位',icon:'glyphicon-folder-open',level:1},
      'search':{note:'搜索',icon:'glyphicon-search',level:1}
      },
      procfun:{
        'Path':function(value){
          if(value.lastIndexOf('/') != value.length - 1)
            value = value+'/';
          return shortstr(value,30);
        },
        'Name':function(value){
          return shortstr(value,20);
        },
        'FileSize':function(tvalue){
          value = tvalue;
          if(value < 0 ) return "";
          if(value < 1024 ) return value + " 字节";
          if(value < 1024*1024 ){value=value/1024; return value.toFixed(1) + " Kb"};
          if(value < 1024*1024*1024*10 ){value=value/1024/1024; return value.toFixed(1) + " Mb"};
          value=value/1024/1024;
          return value.toFixed(1) + " Gb";
        },
        'ModTime':function(value){
          var d = new Date();
          d.setTime(value * 1000);
          return d.toLocaleString();
        }
      },
      editState:0,	//0:显示,1:重命名,2:新建文件夹,3:上传文件,4：在当前目录下检索
 
      bshowcontext:false,
      contextmenu:{memus:{},posx:5,posy:5}
    }
  },
  events:{
    vfkeydown:function(event){
      if(this.editState != 0)
        return true;
      switch(event.key)
      {
        case "Escape":
          this.bshowcontext = false;
          break;
        case "Delete":
          if(this.userLevel >= 3
              && this.buttonstatus("rmfile") == 2
              && confirm("即将删除文件，此操作不能恢复。"))
            this.rmFile();
          break;
        case "F2":
          if(this.userLevel >= 3
              && this.buttonstatus("rename") == 2)
            this.beginEdit(1);
          break;
        case "r":
          this.cache={};
          this.cancelselect();
          this.reflush();
          break;
        case "Enter":
          if(this.buttonstatus("topos") == 2
              && this.selectUnit.IsDir)
          {
            this.path = this.toRealPath(this.toRealPath(this.selectUnit.Path)+this.selectUnit.Name);
            this.reflush();
          }
          break;
      }
    }
  },
  watch:{
    $route:function(val){
      if(!!val.query.search)
      {
        if(val.query.expr != this.ReNameValue || val.query.path != this.path)
        {
          this.searchmode = val.query.search;
          this.ReNameValue = val.query.expr; 
          if(val.query.path.trim().length!=0)
            this.path = val.query.path;
          this.reflush();
          return;
        }
        return;
      }
      else
      {
        if(!!val.query.path && val.query.path != this.path)
        {
          if(val.query.path.trim().length==0)
            return;
          this.path = val.query.path;
          this.reflush();
          return;
        }
      }

      return;
    },
    editState:function (val){
      if(val)
      {
        $("#thedlg").modal("show");
        setTimeout(function(){$("#textinputforfile").select();},50);
      }
      else
      {
        this.bshowcontext = false;
        $("#thedlg").modal("hide");
      }
      return;
    }
  },
  computed:{
    acceptExt:function(){
      return this.uploadFolder?this.sysSet.ArchiveTable:this.sysSet.ExtTable;
    },
    selectName:function(){
        for(var key in this.selectList)
        {
         if(this.selectList[key] == true) return key;
        }
        return "";
      },
    selectUnit:function(){
        for(var key in this.data)
        {
         if(this.toRealPath(this.data[key].Path)+this.data[key].Name == this.selectName) return this.data[key];
        }
        return {};
      },
    urlvec:function(){
        var urlvec = [{url:"/",name:""}];
        var localpath = "/";
        var pathforview = this.path;
        if (this.selectName!="")
        {
          pathforview = this.selectName;
        }
        var vec = pathforview.split('/');
        for(var unt =1;unt< vec.length-1;unt++)
        {
          localpath = localpath+ vec[unt] + "/";
          urlvec[unt] = {url:localpath,name:vec[unt]};
        }
        return urlvec;
      }
  },
  methods: {
    openmenu:function(event){
      this.contextmenu.posx = event.layerY;
      this.contextmenu.posy = event.layerX;
      var tmpmenu = {
        'reflush':{note:'刷新',icon:'glyphicon-refresh',level:1},
        'preview':{note:'预览',icon:'glyphicon-film',level:2},
        'rename':{note:'重命名',icon:'glyphicon-pencil',level:3},
        'rmfile':{note:'删除',icon:'glyphicon-remove',level:3},
        'packetdown':{note:'下载',icon:'glyphicon-download',level:2}
      };
    

      if(this.tableforsearch)
        tmpmenu['topos'] = {note:'打开所在文件夹',icon:'glyphicon-screenshot',level:1};
      else
        tmpmenu['newfolder'] = {note:'新建',icon:'glyphicon-plus',level:3},
      
      this.contextmenu.memus = (new Function ("return {};")) ();
      for(var key in this.contextmenu.memus)
        delete this.contextmenu.memus[key];

      for(var key in tmpmenu)
      {
        if(tmpmenu[key].level <= this.userLevel && this.buttonstatus(key) == 2)
        {
          Vue.set(this.contextmenu.memus,key,tmpmenu[key]);
        }
      }
      this.bshowcontext = true;
    },
    cancelselect:function(){
      this.selectList = {};
    },
    buttonclick:function(btn){
      switch(btn)
      {
        case 'reflush':
          this.cache={};
          this.cancelselect();
          this.reflush();
          break;
        case 'uplevel':
          this.pathLeval(-1);
          break;
        case 'preview':
          this.viewfile();
          break;
        case 'rename':
          this.beginEdit(1);
          break;
        case 'newfolder':
          this.beginEdit(2);
          break;
        case 'rmfile':
          if(confirm("即将删除文件，此操作不能恢复。"))
            this.rmFile();
          break;
        case 'upload':
          this.beginEdit(3);
          break;
        case 'packetdown':
          if(this.selectUnit.FileSize > MaxFileDownload){
            this.$dispatch('warning-msg', {type:"danger",text:"文件夹大小超过限制，不能打包下载"});
            return ;}
            this.downLoadThis(this.selectUnit);
          break;
        case 'search':
          this.beginEdit(4);
          break;
        case 'topos':
          this.openPath(this.selectUnit.Path);
          break;
        default:
          alert("未定义的操作"+btn);
      }
    },
    buttonstatus:function(btn){
      switch(btn)
      {
        //文件操作类
        case 'rename':
          if(this.selectName == '')
            return 0;
          break;
        case 'rmfile':
          if(this.selectName == '')
            return 0;
          break;
        case 'upload':
          if(this.tableforsearch)
            return 0;
          break;
        case 'newfolder':
          if(this.tableforsearch)
            return 0;
          break;
        //下载类
        case 'packetdown':
          if(this.selectName == '')
            return 0;
          break;
        case 'preview':
          if(this.selectName == '')
            return 0;
          if(this.checktype(this.selectName)=='unknown')
            return 1;
          break;
        //查看类
        case 'uplevel':
          if(this.tableforsearch)
            return 0;
          break;
        case 'search':
          break;
        case 'topos':
          if(this.selectName == '')
            return 1;
          break;
        case 'reflush':
          return 2;
        default:
          return 0;
      }

      return 2;
    },
    toRealPath:function(path){
      if(path.length < 1)
        return '/';
      if(path[0]!= '/')
        path = '/'+path;
      if(path.lastIndexOf('/') != path.length - 1)
        path = path + '/';
      return path;
    },
    openPath:function(path){
      this.path = this.ShowInfo("Path",path);
      this.cancelselect();
      this.reflush();
    },
    checktype:function(file){
      var prelist = {
        'code':["txt","ini","xml","cpp","h","pro","js","html","css","csv","json","go"],
        'img':["png","jpg","jpeg","bmp","gif"],
        'audio':["mp3","wav"],
        'video':["mp4","flv"]
      }
      var ext = file.split('.');
      if(ext.length < 2)
        return "unknown";
      ext = ext.pop();
      for(var nti in prelist)
      {
        for(var psr in prelist[nti])
        {
          if(ext.toLowerCase() == prelist[nti][psr]) return nti;
        }
      }
      return "unknown";
    },
    endpreview:function(){
      $("#pre-body").empty();
    },
    viewfile:function(){
      $("#viewdlg").modal("show");
      if(this.selectName.length < 1) {$("#viewdlg").modal("hide");return false;}
      var dom = $("#pre-body");
      var url = "/api/file"+this.selectName;
      $("#filepre-title").text(this.selectName);
      var type = this.checktype(this.selectName);
      switch( type )
      {
      case 'code'://文本、代码
        $("#codepre").show();
        localGet(url,"",function(data){
          var $subdom = $("<pre></pre>");
          $subdom.attr("class","pre-scrollable linenums");
          $subdom.text(data);
          dom.empty();
          dom.append($subdom);
        });
        break;
      case 'img'://图片
        var $subdom = $("<img>");
        $subdom.attr("class","img-thumbnail");
        $subdom.attr("src",url);
        dom.empty();
        dom.append($subdom);
        break;
      case 'audio'://音频
      case 'video'://视频
        var $subdom = $("<"+type+">");
        $subdom.attr("src",url);
        $subdom.attr("controls","true");
        $subdom.attr("autoplay","true");
        $subdom.attr("width","100%");
        $subdom.attr("poster","image/prefilm.jpg");
        dom.empty();
        dom.append($subdom);
        break;
      default:
        dom.html("<p>不能预览</p>");
        break;
      }
    },
    getData:function(data){
      this.data = data;
    },
    reflush:function(){
        this.bshowcontext = false;
        if(this.searchmode)
        {
          var obj = this;
          //为了适应xxx的错误路径问题，临时修改
          var searchpath = this.toRealPath(this.path).substr(0,this.toRealPath(this.path).length-1);
          if(searchpath == '')
            searchpath = '/';

          tnowdir = "/api/files" + searchpath;
          localGet(tnowdir, {Expr:this.ReNameValue},this.getData);
          this.searchmode = false;
          
          this.$route.router.go({path:"/files",query:{search:true,path:searchpath,expr:this.ReNameValue}});
          this.tableforsearch = true;
          return;
        }

        this.$route.router.go({path:"/files",query:{path:this.path}});

        var obj = this;

        if(this.cache.bcache && this.toRealPath(this.path) in this.cache.data)
        {
          this.data = this.cache.data;
        }
        else
        {
          tnowdir = "/api/dir" + this.path;
          localGet(tnowdir, {Expr:this.ReNameValue},this.getData);
        }
        this.ReNameValue = "";
        this.tableforsearch = false;
        return;
    },

    sortBy: function (key) {
      this.sortKey = key
      this.sortOrders[key] = this.sortOrders[key] * -1
    },
	  downLoadThis:function (obj){
      if(obj.IsDir)
        droptoDownload("/api/archive" + obj.Path + obj.Name);
      else
        droptoDownload("/api/file" + obj.Path + obj.Name);

      return;
	},
	openThis:function (obj){
		if(obj.IsDir == false) this.downLoadThis(obj) ;
		else
    {
      this.cancelselect();
      this.path = this.toRealPath(this.toRealPath(obj.Path)+obj.Name);
      
      this.reflush();
    }
	},
	rmFile:function(){
		var obj = this;
		if(this.selectName.length==0) return;
		var url = this.selectName;
    
    if(this.selectUnit.IsDir)
      url = "/api/dir"+url;
    else
      url = "/api/file" + url;

		localDelete(url, "",this.reflush);
		return true;
	},
  checkuploadfile:function(){
      var fileItems = document.getElementById("dlg-for-upload-file").files;
      for(key = 0; key < fileItems.length; key ++)
      {
        var fileObj = fileItems[key];
        var extvec = this.acceptExt.split(",");
        for(var i=0;i<extvec.length;i++)
        {
          if(fileObj.name.lastIndexOf(extvec[i]) == fileObj.name.length - extvec[i].length)
            break;
        }
        if(i == extvec.length)
        {
          this.$dispatch('warning-msg', fileObj.name+" 后缀名不合法");
          continue;
        }
      }
  },
	postEdit:function(){
		var obj = this;
		if(this.editState == 0) return true;
		switch(this.editState)
		{
		case 1:
			var url = "/api/file" + this.selectName;
			localPut(url, {"NewName":this.ReNameValue},function (data){
        obj.reflush();
        obj.selectThis(obj.toRealPath(obj.path)+obj.ReNameValue);
			});
			break;
		case 2:
			localPost("/api/dir" + this.path.replace(" ","") + this.ReNameValue, {},obj.reflush);
			break;
		case 3:
      var fileItems = document.getElementById("dlg-for-upload-file").files;
      $("#thedlg").modal("hide");
      for(key = 0; key < fileItems.length; key ++)
      {
        var fileObj = fileItems[key];
        var extvec = this.acceptExt.split(",");
        for(var i=0;i<extvec.length;i++)
        {
          if(fileObj.name.lastIndexOf(extvec[i]) == fileObj.name.length - extvec[i].length)
            break;
        }
        if(i == extvec.length)
        {
          obj.$dispatch('warning-msg', {type:"danger",text:fileObj.name+" 后缀名不合法"});
          continue;
        }
        var url = this.toRealPath(this.path);
        pushtoUpload( fileObj, (this.uploadFolder?"/api/archive":"/api/file") + url + fileObj.name,
          function(data){
            obj.reflush();
            obj.$dispatch('warning-msg', {type:"success",text:data});
          },
          function(data){
            obj.$dispatch('warning-msg', {type:"danger",text:data});
          });
      }
      break;
		case 4:
      this.searchmode = true;
      this.reflush();
			break;
		default:
			return true;
		}
		this.ReNameValue = "";
		this.editState = 0;
		return true;
	},
	cancelEdit:function(){
		this.editState = 0;
		this.ReNameValue = "";
	},
	beginEdit:function(state){
    this.editState = 0;

		switch(state)
		{
		case 1:
			if(this.selectName == '') return;
      this.editDlgInfo.title="重命名";
      this.editDlgInfo.placeholder="重命名";
			this.ReNameValue = this.selectName;
      this.ReNameValue = this.ReNameValue.replace(this.path.replace(" ",""),"");
			break;
		case 2:
			this.ReNameValue = "新建目录";
      this.editDlgInfo.title="新建目录";
      this.editDlgInfo.placeholder="新建目录";
			break;
		case 3:
      this.editDlgInfo.title="上传文件/批量上传文件(夹)";
      this.editDlgInfo.placeholder="";
			break;
		case 4:
      this.editDlgInfo.title="搜索文件";
      this.editDlgInfo.placeholder="在当前目录下搜索";
			this.ReNameValue = "";
			break;
		default:
			return true;
		}

    var obj = this;
    setTimeout(function(){
      obj.editState = state;
      },50);
	},
	pathLeval: function (lev){
		//查找当前级别
		var pos = this.path.lastIndexOf('/') - 1;
		while(lev < 0 && pos >= 1)
		{
      this.selectList = {};
      this.selectList[this.path.substr(0,pos+1)] = true;
			pos = this.path.lastIndexOf('/',pos - 1);
			lev ++;
		}

		if(lev != 0)
    {
      this.$dispatch('warning-msg', "已经是最顶层目录");
      return;
    };
		
		this.path = this.path.substr(0,pos + 1);
    
    this.reflush();
	},
  ShowInfo:function (key,value){
    if(!(key in this.procfun)) return value;
    else return this.procfun[key](value);
  },
	selectThis:function (obj,event){
		if(event && !event.ctrlKey)
		  this.selectList = {};
		if(this.selectList[obj] == true) this.selectList[obj] = false;
		else this.selectList[obj]= true;
		return;
  	}
  }
});



//通用表格显示混合对象
var commonshow = {
  data: function (){
    return {
      procfun:{
        'FileSize':function(tvalue){
          value = tvalue;
          if(value < 0 ) return "";
          if(value < 1024 ) return value + " 字节";
          if(value < 1024*1024 ){value=value/1024; return value.toFixed(1) + " Kb"};
          if(value < 1024*1024*1024*10 ){value=value/1024/1024; return value.toFixed(1) + " Mb"};
          value=value/1024/1024;
          return value.toFixed(1) + " Gb";
        },
        'Path':function(value){
          return shortstr(value,30);
        },
        'Name':function(value){
          return shortstr(value,20);
        },
        'Time':function(value){
        var d = new Date();
        d.setTime(value * 1000);
        return d.toLocaleString();
        },
        'Ip':function(value){
          var ipstr = "";
          var i=0;
          while( i <4)
          {
            var num = new Number(value & 0XFF);
            if(i == 0) ipstr = num.toString();
            else ipstr = num.toString()+"."+ipstr;
            value = value>>8;
            i++;
          }
          return ipstr;
        }
      }
    };
  }
};


//通用表格混合对象
var commontable = {
  components:{
    "bottom-table":bottomtable
    },
  data: function (){
    return {
      tabledata:[],
      tabledatakey:'',
      searchInfo:{Page:1,OnePageCount:10},
      totalNum:0,
      tableurl:'',
      procfun:{
        'FileSize':function(tvalue){
          value = tvalue;
          if(value < 0 ) return "";
          if(value < 1024 ) return value + " 字节";
          if(value < 1024*1024 ){value=value/1024; return value.toFixed(1) + " Kb"};
          if(value < 1024*1024*1024*10 ){value=value/1024/1024; return value.toFixed(1) + " Mb"};
          value=value/1024/1024;
          return value.toFixed(1) + " Gb";
        }
      }
    };
  },
  ready: function (){
      this.reFlush();
    },
  watch:{
    searchInfo:{
      handler:function(val){ this.reFlush();},
      deep:true
     }
  },
  methods:{
    reFlush:function (){
      var obj = this;
      localGet(this.tableurl,this.searchInfo,function(data){
        obj.tabledata = data[obj.tabledatakey];
        obj.totalNum = data.Sum;
      })
    },
    ShowInfo:function (key,value){
      if(!(key in this.procfun)) return value;
      else return this.procfun[key](value);
    }
  }
}


var userCon = Vue.extend({
  template: '#usertable-template',
  replace: true,
  mixins:[commontable],
  ready:function(){
      if(typeof(this.Sysset) == "undefined")
      {
        var obj = this;
        $.get("/api/conf",{},function(data){obj.Sysset = data;} );
      }
    },
  data: function () {
    var thiscolumns = {'Uid':'ID','Name':'用户名','RealName':'姓名','Department':'单位','Class':'管理员组','LastIp':'上次登录位置','LastLoginTime':'上次登录时间'};
    return {
      columns:thiscolumns,
      nowSelect:{},
      Sysset:globalSysset,
      tabledatakey:'UsrList',
      tableurl:'/api/usrs',
      procfun:{
        'LastLoginTime':function(value){
        var d = new Date();
        d.setTime(value * 1000);
        return d.toLocaleString();
        },
        'LastIp':function(value){
          var ipstr = "";
          var i=0;
          while( i <4)
          {
            var num = new Number(value & 0XFF);
            if(i == 0) ipstr = num.toString();
            else ipstr = num.toString()+"."+ipstr;
            value = value>>8;
            i++;
          }
          return ipstr;
        }
      }
    }
  },
  methods:{
    edit:function (unt){
      this.nowSelect = unt;
      $("#modifyuserdlg").modal("show");
    },
    postEdit:function (){
      var obj = this;
      var nowselect = this.nowSelect;
      localPut("/api/usrs",this.nowSelect,function(){
        $("#modifyuserdlg").modal("hide");
        obj.$dispatch('warning-msg', {type:"success",text:nowselect.RealName + " 信息修改成功"});
        obj.reflush();
        });
    },
    deleteusr:function (unt){
      var obj = this;
      var objname = unt.RealName;
      localDelete("/api/usrs?UidList="+unt.Uid,{},function(){
        obj.$dispatch('warning-msg', {type:"success",text:objname+ " 删除成功"});
        obj.reFlush();
        },
        function(XMLhttpRequest){obj.$dispatch('warning-msg', {type:"danger",text:errStr(XMLhttpRequest.status)});});
    }
  }
});




var dlCountCon = Vue.extend({
  template: '#dlcounttable-template',
  replace: true,
  mixins:[commontable,commonshow],
  data: function () {
    var thiscolumns = {'Path':'路径','Cnt':'下载次数','FileSize':'文件大小','Time':'最近下载时间'};
    return {
      timeoption:{365:"1年",183:"半年",30:"1月",7:"1周",1:"1天"},
      columns:thiscolumns,
      searchInfo:{Page:1,OnePageCount:10,Day:'365'},
      tabledatakey:'CntList',
      tableurl:'/api/cnt/'
    }
  },
  methods:{
    download:function (unt){
        if(unt.IsDir)
        {
          if(unt.FileSize > MaxFileDownload){
            this.$dispatch('warning-msg', {type:"danger",text:"文件夹大小超过限制，不能打包下载"});
            return ;
          }
          droptoDownload("/api/archive" + unt.Path);
        }
        else
          droptoDownload("/api/file" + unt.Path);
    }
  }
});



var dlListCon = Vue.extend({
  template: '#dllistrtable-template',
  replace: true,
  mixins:[commontable,commonshow],
  data: function () {
    var thiscolumns = {'Path':'路径','FileSize':'大小','RealName':'姓名','Department':'单位','Ip':'IP','Time':'时间'};
    return {
      columns:thiscolumns,
      tabledatakey:'DownloadList',
      tableurl:'/api/downloads/',
      showfilter:false
    }
  },
  methods:{
    download:function (unt){
        if(unt.IsDir)
        {
          if(unt.FileSize > MaxFileDownload){
            this.$dispatch('warning-msg', {type:"danger",text:"文件夹大小超过限制，不能打包下载"});
            return ;
          }
          droptoDownload("/api/archive" + unt.Path);
        }
        else
          droptoDownload("/api/file" + unt.Path);
    }
  }
});




// define some components
var Empty = Vue.extend({
  template: '<p>This is an empty component!</p>'
})

// define some components
var Addusr = Vue.extend({
  template: '#form-addusr-template',
  data: function(){
    return {
        User:{Name:"",RealName:"",Department:"","Password":"","Class":""},
        CheckPasswords:"",
        Sysset:globalSysset
        }
     },
  computed:{
       validation:function(){
         return {
            Name:this.User.Name.length==0||this.User.Name.length>=5,
            RealName:this.User.RealName.length==0||(this.User.RealName.length>=2),
            Password:this.User.Password.length==0||this.User.Password.length>=5,
            Department:this.User.Department.length==0||this.User.Department.length>=3,
            Class:this.User.Class.length==0||this.User.Class.length>=2,
            CheckPasswords:this.CheckPasswords.length==0||this.User.Password==this.CheckPasswords
         };
       }
     },
  methods:{
      checkform:function(keys){
        return this.User[keys].length>5;
      },
      postEdit:function(){
        var obj = this;
        localPost("/api/usrs",this.User,
          function()
          {
            obj.$dispatch('warning-msg', {type:"success",text:"用户 "+obj.User.RealName+" 添加成功"});
            obj.$route.router.go({path:"/user"});
          },
          procError);
      }
    }
})


var UserSet = Vue.extend({
  template: '#userset-template',
  data: function(){
    var thisdata ={};
    var jqobj = $.ajax({url:"/api/usr",async:false,data:{},success:function(data){
      thisdata = data;
    },
    error:procError
    });
    return {
        User:thisdata,
        ResetPwd:{OldPassword:"",NewPassword:""},
        CheckPasswords:""
        }
     },
  computed:{
       validation:function(){
         return {
            Name:this.User.Name.length==0||this.User.Name.length>=5,
            RealName:this.User.RealName.length==0||(this.User.RealName.length>=5),
            Password:this.User.Password.length==0||this.User.Password>=5,
            Department:this.User.Department.length==0||this.User.Department>=5,
            CheckPasswords:this.CheckPasswords.length==0||this.User.Password==this.CheckPasswords
         };
       }
     },
  methods:{
      checkform:function(keys){
        return this.User[keys].length>5;
      },
      postEdit:function(){
        var obj = this;
        localPut("/api/usr",this.ResetPwd,
          function(){
            obj.$dispatch('warning-msg', {type:"success",text:"密码修改成功"});
          });
      }
    }
})


var dlChartCon = Vue.extend({
  template: '#dlchart-template',
  ready:function(){
      this.resizeChart("majorchart");
      for(var key in this.chartObject)
      {
        if(key != this.mainchart)
          this.resizeChart(key);
        this.requestData(key);
      }
  },
  data:function(){
    var defaultdata = function (){
      return{
        labels : [],
        datasets : [
        {
        label: "",
        fillColor : "rgba(24, 82, 255, 0.2)",
        strokeColor : "rgba(48, 164, 255, 1)",
        pointColor : "rgba(48, 164, 255, 1)",
        pointStrokeColor : "#fff",
        pointHighlightFill : "#fff",
        pointHighlightStroke : "rgba(48, 164, 255, 1)",
        data : []
        }
        ]
      };
    };

    var charobjdef = function(title,url,param,type,paramopt){
      return {
        title:title,
        data:defaultdata(),
        unt:null,
        url:url,
        param:param,
        type:type?type:"Line",
        paramopt:paramopt?paramopt:[]
      };
    };

    var d = new Date();
    var nowyear = d.getFullYear();
    var optyear = {0:"不限"};
    for (var i=0;i<3 ;i++ )
    {
      optyear[nowyear-i] = nowyear-i;
    }


    return {
      showlegend:true,
      mainchart:"tendencychart",
      chartObject:{
        tendencychart:charobjdef("下载流量变化趋势","/api/graph/",{Flag:"Day",Day:1,Limit:10},"Line"),
        monthchart:charobjdef("月统计信息","/api/graph/",{Flag:"Month",Year:0},"Bar"),
        weekchart:charobjdef("周统计信息","/api/graph/",{Flag:"Weekday",Year:0},"Bar")
//        yearchart:charobjdef("年度统计信息","/api/graph/",{Flag:"Year",Year:0},"Bar")
      },
      optdayunit:{1:"1天",3:"3天",7:"1周",15:"半月",30:"1月"},
      optyearlimit:optyear,
      optcharttype:{Line:"折线图",Bar:"柱状图",Radar:"雷达图"}
    };
  },
  computed:{
    optdaylast:function(){
      var numsopt = [5,7,10,12,15,30];
      var showtitles = {7:"1周",15:"半月",30:"1月",180:"半年",360:"1年"};

      var retlast = {};
      for(var i=0;i<numsopt.length;i++)
      {
        var tdays = numsopt[i]*this.chartObject.tendencychart.param.Day;
        retlast[numsopt[i]] = tdays+"天";
      }
      return retlast;
    }
  },
  watch:{
    mainchart:function(val,oldval){
      var obj = this;
      this.redraw(val);
      this.resizeChart(oldval);
      this.redraw(oldval);
    }
  },
  methods:{
    toggleLegend:function(){
      this.showlegend = !this.showlegend;
      var obj = this;
      obj.resizeChart("majorchart");
      obj.requestData(this.mainchart);
    },
    changethis: function(event){
      $(event.currentTarget).next().toggle();
    },
    updataParm :function(chartname,parmstring,data){
      if( !(chartname in this.chartObject))
      {
        return;
      }
      
      var paramstr = "this.chartObject."+chartname+"."+parmstring;
      var testret;
      try{
        testret = eval(paramstr);
        if(typeof(testret) == "undefined")
          return;
      }
      catch(exception)
      {
        return;
      }

      eval("this.chartObject."+chartname+"."+parmstring + " = data");
      
      this.requestData(chartname);

    },
    redraw:function(chartname){
      if( !( chartname in this.chartObject))
      {
        return;
      }
      var chartdrawid = chartname;
      if(chartname == this.mainchart)
        chartdrawid = "majorchart";
      //重绘图表之变态方式，通过删除本DOM再添加DOM方式进行
      var ordom = $("#"+chartdrawid);
      var canvas = ordom.clone();
      ordom.replaceWith(canvas);

      var obj = document.getElementById(chartdrawid);
    
      if( !!!obj) return;
      
      this.chartObject[chartname].unt = new Chart(obj.getContext("2d"));

      var option = {
        responsive: true
      };

      eval("this.chartObject[chartname].unt."+this.chartObject[chartname].type+"(this.chartObject[chartname].data,option)");
    },
    showlabel:function(dataobj){
      var labels = [];
      switch(dataobj.Flag)
      {
        case "Year":
          var rt=[];
          for(var k =0;k<dataobj.PointList.length;k++)
          {
            rt.push(dataobj.PointList[k].Year);
          }
          return rt;
        case "Month":
          return ['一月','二月','三月','四月','五月','六月','七月','八月','九月','十月','十一月','十二月'];
        case "Weekday":
          return ['星期日','星期一','星期二','星期三','星期四','星期五','星期六'];
        case "Day": 
          for(var i = 0;i< dataobj.PointList.length;i++)
          {
            labels[i] = dataobj.PointList[i].Year + "-" + dataobj.PointList[i].Month + "-" +
              dataobj.PointList[i].Day;
          }
          return labels;
        default:
          return [];
      }
    },
    resizeChart:function(chartname){
      var parentwidth = $("#"+chartname).parent().width();
      var obj = $("#"+chartname);
      obj.attr("width",parentwidth);
      obj.css({width:parentwidth});
    },
    requestData:function(chartname){
      if( !( chartname in this.chartObject))
      {
        return;
      }
      
      var thiscom = this;
      localGet(this.chartObject[chartname].url,this.chartObject[chartname].param,function (data){
        //解析数据

        thiscom.chartObject[chartname].data.labels = thiscom.showlabel(data);
        thiscom.chartObject[chartname].data.datasets[0].data = [];
        for(var i = 0;i< data.PointList.length;i++)
        {
          thiscom.chartObject[chartname].data.datasets[0].data[i] = data.PointList[i].Cnt;
        }
        thiscom.redraw(chartname);
      });

    }
  }
})

// define some components
var SysSet = Vue.extend({
  template: '#form-sysset-template',
  data: function(){

    var thisdata ={};
    var jqobj = $.ajax({url:"/api/conf",async:false,data:{},success:function(data){thisdata = data;}});

    return {
      globalset:thisdata
     };
  }
})

var WarnBar = Vue.extend({
  template: '#warnbar-template',
  data: function(){
    return {
      showstate:0,
      warninfo:{}
     };
  },
  ready:function(){
    $("#alertbar").css({width:$("#alertbar").parent().width()});
  },
  computed:
  {
    divclass:function(){
      switch(this.warninfo.type)
      {
        case "success":
          return "bg-success";
        case "warning":
          return "bg-warning";
        case "danger":
          return "bg-danger";
        default:
          return "bg-primary";
      }
    },
    iconclass:function(){
      switch(this.warninfo.type)
      {
        case "success":
          return "glyphicon-check";
        case "warning":
          return "glyphicon-warning-sign";
        case "danger":
          return "glyphicon-exclamation-sign";
        default:
          return "glyphicon-info-sign";
      }
    }
  },
  watch:{
    showstate:function(val){
      if(val==0) $("#alertbar").fadeOut(600);
      if(val==1) {$("#alertbar").stop();$("#alertbar").fadeTo(400,0.9);}
      if(val==2) {$("#alertbar").stop(); $("#alertbar").css({opacity:1});}
    }
  },
  events:{
    dispwarn:function(wsinfo){
      if(typeof(wsinfo)=="string")
        this.warninfo = {type:"warning",text:wsinfo};
      else
        this.warninfo = wsinfo;
      if(this.showstate==0) this.showstate=1;
      var obj = this;
      setTimeout(function(){if(obj.showstate!=2) obj.showstate=0;},2000);
    }
  }
})



// the router needs a root component to render.
// for demo purposes, we will just use an empty one
// because we are using the HTML as the app template.
var App = Vue.extend({
  components:{
    "modal-slidemenu":leftmenu,
    "modal-warnbar":WarnBar
    },
  methods:{
    dispkeyinfo:function(event){
      this.$broadcast("vfkeydown",event);
      },
    showWarn:function(warnstr){
      this.$broadcast("dispwarn",warnstr);
      }
  }
});
// create a router instance
// you can pass in additional options here, but
// let's keep it simple for now.
var router = new VueRouter()

// define some routes.
// each route should map to a component.
// we'll talk about nested routes later.
router.map({
  '/': {
    name:'FileMG',
    component: FileCon
  },
  '/empty': {
    component: Empty
  },
  '/usrset': {
    component: UserSet
  },
  '/files': {
    name:'FileMG',
    component: FileCon
  },
  '/user': {
    component: userCon
  },
  '/addusr': {
    component: Addusr
  },
  '/dlcount': {
    component: dlCountCon
  },
  '/dllist': {
    component: dlListCon
  },
  'dlchart': {
    component: dlChartCon
  },
  '/sysset': {
    component: SysSet
  }
})

// now we can start the app!
// router will create an instance of App and mount to
// the element matching the selector #app.
router.start(App, '#bodyid');

// define the item component


