package test

import (
	"fmt"
	"github.com/99designs/gqlgen/handler"
	"github.com/friendsofgo/graphiql"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/zltgo/dress/conf"
	"github.com/zltgo/dress/graph"
	"github.com/zltgo/dress/model"
	"github.com/zltgo/webkit/cache/lru"
	"github.com/zltgo/webkit/ginx"
	gm "github.com/zltgo/webkit/graphorm"
	"github.com/zltgo/webkit/jwt"
	"github.com/zltgo/webkit/ratelimit"
	"log"
	"math/rand"
	"net/http"
	"time"
)

var admin = &model.AuthInfo{
	Empno: "root",
	Role:  model.AdminRole,
}


//Create a server with gin and graphql
func NewServer(m *model.Model, cfg *conf.Conf) *gin.Engine {
	// Setting up Gin
	r := gin.Default()
	r.ForwardedByClientIP = false//important for getting remote address

	//cors, allow all origins
	r.Use(NewCrosMiddleware())
	//r.Use(NewRatelimiter(cfg.RateLimitIP.MaxEntries, cfg.RateLimitIP.RateOpts))
	r.Use(func(c *gin.Context) {
		c.Request.RemoteAddr = fmt.Sprintf("%d.%d.%d.%d:%d", rand.Intn(256),rand.Intn(256),rand.Intn(256),rand.Intn(256), rand.Intn(65535))

	})

	auth := jwt.NewAuth(cfg.AuthOpts)
	r.Use(func(c *gin.Context) {
		c.Set("model", m)
		c.Set("auth", auth)
		ginx.GinContextToHttpContext(c)
	})

	graphqlHandler := NewGraphqlHandler(cfg.Graphql.ComplexityLimit, cfg.Graphql.CacheSize)
	playgroundHandler := handler.Playground("GraphQL playground", "/graphql")
	graphiqlHandler, err := graphiql.NewGraphiqlHandler("/graphql")
	if err != nil {
		panic(err)
	}
	r.POST("/graphql", gin.WrapH(graphqlHandler))
	r.GET("/playground", gin.WrapH(playgroundHandler))
	r.GET("/graphiql", gin.WrapH(graphiqlHandler)) //the same as playground

	return r
}

func NewRatelimiter(maxEntries int, rateOpts []int) gin.HandlerFunc{
	lc := lru.New(maxEntries)
	rates := ratelimit.SecOpts(rateOpts...)
	return ginx.NewRatelimiter(lc, rates)
}

func NewCrosMiddleware() gin.HandlerFunc{
	//cors, allow all origins
	cfg := cors.DefaultConfig()
	cfg.AllowAllOrigins = true
	cfg.AllowHeaders = append(cfg.AllowHeaders, "ACCESS-TOKEN", "REFRESH-TOKEN")
	return cors.New(cfg)
}


func NewGraphqlHandler(complexityLimit, cacheSize int) http.Handler {
	graphCfg := graph.Config{Resolvers: &graph.Resolver{}}

	//限制查询复杂度，将数组展开计算
	//countComplexity := func(childComplexity, count int) int {
	//return count * childComplexity
	//}
	//graphCfg.Complexity.Query.Posts = countComplexity
	//graphCfg.Complexity.Post.Related = countComplexity

	return handler.GraphQL(
		graph.NewExecutableSchema(graphCfg),
		handler.ComplexityLimit(complexityLimit),  //限制递归查询次数为100次，数组计算长度
		handler.IntrospectionEnabled(false),       //关闭内省
		handler.CacheSize(cacheSize),              //自带一个缓存,只是缓存解析查询语句的中间结果
		handler.RecoverFunc(gm.Recover),          //把未知错误panic,由gin框架来处理
		handler.ErrorPresenter(gm.ErrorPresenter), //自动将错误信息交给gin框架处理
	)
}

func TimeFunc(sec int64, nsec int64) func() time.Time {
	return func() time.Time {
		return time.Unix(sec, nsec)
	}
}

type Model struct {
	*model.Model
}

func (m Model) AddAccount(realname, mobile, corp string) gm.UUID {
	admin := &model.AuthInfo{
		Empno: "root",
		Role:  model.AdminRole,
	}

	ac, code, err := m.CreateAccount(admin, &model.NewAccount{
		Empno:            "000",
		Password:         "000000000",
		RealName:         realname,
		Mobile:           mobile,
		Corp:             corp,
		MaxStores:        10,
		MaxUsersPerStore: 100,
		ExpiresAt:        jwt.TimeNow().Add(time.Hour * 24),
		Remarks:          fmt.Sprintf("公司：%s, 手机：%s", corp, mobile),
	})
	if err != nil {
		log.Println(code, err)
		return gm.ZeroUUID()
	}
	m.AddStoreAddUser(ac.ID, "分店0", "0")
	m.AddStoreAddUser(ac.ID, "分店1", "1")
	m.AddStoreAddUser(ac.ID, "分店2", "2")

	return ac.ID
}

func (m Model) AddStoreAddUser(accountId gm.UUID, name string, empnoPrefix string) {
	st, code, err := m.CreateStore(admin, &model.NewStore{
		AccountID: accountId,
		Name:      name,
		Remarks:   fmt.Sprintf("分店名：%s", name),
	})
	if err != nil {
		log.Println(code, err)
		return
	}

	//add employee
	m.AddUser(accountId, st.ID, empnoPrefix+"001", "000000001", "张0", "级别0")
	m.AddUser(accountId, st.ID, empnoPrefix+"002", "000000002", "张1", "级别1")
	m.AddUser(accountId, st.ID, empnoPrefix+"003", "000000003", "张2", "级别2")
}

func (m Model) AddUser(accountId, storeId gm.UUID, empno, password, realName, role string) {
	_, code, err := m.CreateUser(admin, &model.NewUser{
		AccountID: accountId,
		Empno:     empno,
		Password:  password,
		RealName:  realName,
		Role:      role,
		StoreID:   storeId,
	})
	if err != nil {
		log.Println(code, err)
	}
}
