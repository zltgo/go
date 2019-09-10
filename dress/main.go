package main

import (
	"github.com/99designs/gqlgen/handler"
	"github.com/friendsofgo/graphiql"
	"github.com/fvbock/endless"
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
	"net/http"
)

func main() {
	//read config and create model
	cfg := conf.ReadCfg("./conf/conf.yaml")

	m, err := model.NewModel(cfg.Model)
	if err != nil {
		log.Fatal(err)
	}
	defer m.CloseDB()

	server := NewServer(m, cfg)
	log.Println(endless.ListenAndServe(cfg.ServeAddr, server))
}


//Create a server with gin and graphql
func NewServer(m *model.Model, cfg *conf.Conf) *gin.Engine {
	// Setting up Gin
	r := gin.Default()
	r.ForwardedByClientIP = false//important for getting remote address

	//cors, allow all origins
	r.Use(NewCrosMiddleware())
	//see models.Ratelimit of IP
	//r.Use(NewRatelimiter(cfg.RateLimitIP.MaxEntries, cfg.RateLimitIP.RateOpts))

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
